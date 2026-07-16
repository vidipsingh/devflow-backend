package service

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"devflow-backend/internal/database"
	"devflow-backend/internal/models"
	"devflow-backend/internal/repository"

	"github.com/gabriel-vasile/mimetype"
	"go.mongodb.org/mongo-driver/v2/bson"
)

var ErrFileNotFound = errors.New("file not found")

const treeCacheTTL = 60 * time.Second
const blobCacheTTL = 120 * time.Second

func UploadFile(ctx context.Context, ownerID bson.ObjectID, ownerName string, repoSlug string, req models.UploadFileRequest) (*models.RepoCommit, error) {
	// 1. Resolve repository
	repo, err := repository.FindRepoByOwnerAndSlug(ctx, ownerID, repoSlug)
	if err != nil || repo == nil {
		return nil, ErrFileNotFound
	}

	branch := req.Branch
	if branch == "" {
		branch = repo.DefaultBranch
	}

	// 2. Decode base64 content
	rawContent, err := base64.StdEncoding.DecodeString(req.Content)
	if err != nil {
		return nil, fmt.Errorf("invalid base64 content: %w", err)
	}

	// 3. Compute SHA256 of content
	sum := sha256.Sum256(rawContent)
	sha := fmt.Sprintf("%x", sum)

	// 4. Detect MIME type
	mime := mimetype.Detect(rawContent)
	mimeStr := mime.String()
	encoding := "utf-8"
	if !strings.HasPrefix(mimeStr, "text/") {
		encoding = "binary"
	}

	// 5. Upload content to GridFS
	gridFSName := fmt.Sprintf("%s/%s/%s", repo.ID.Hex(), branch, req.Path)
	gridFSID, err := database.GridFSUpload(ctx, gridFSName, rawContent)
	if err != nil {
		return nil, fmt.Errorf("gridfs upload failed: %w", err)
	}

	// 6. Create commit record
	commitMsg := req.Message
	if commitMsg == "" {
		commitMsg = fmt.Sprintf("Upload %s", path.Base(req.Path))
	}
	commitID := bson.NewObjectID()
	shortHash := commitID.Hex()[:7]

	commit := &models.RepoCommit{
        ID:         commitID,
        RepoID:     repo.ID,
        Branch:     branch,
        Message:    commitMsg,
        AuthorID:   ownerID,
        AuthorName: ownerName,
        ShortHash:  shortHash,
        FilePaths:  []string{req.Path},
        Additions:  len(strings.Split(string(rawContent), "\n")),
        Deletions:  0,
    }
	if err := repository.InsertCommit(ctx, commit); err != nil {
		return nil, err
	}

	// 7. Upsert file metadata in repo_files
    now := time.Now()
    fileMeta := &models.RepoFile{
        RepoID:      repo.ID,
        Path:        req.Path,
        Name:        path.Base(req.Path),
        Dir:         path.Dir(req.Path),
        Size:        int64(len(rawContent)),
        MimeType:    mimeStr,
        Encoding:    encoding,
        SHA:         sha,
        GridFSID:    gridFSID,
        Branch:      branch,
        CommitID:    commitID,
        IsDirectory: false,
        CreatedAt:   now,
        UpdatedAt:   now,
    }
    if err := repository.UpsertFile(ctx, fileMeta); err != nil {
        return nil, err
    }

	// 8. Ensure all parent directories are recorded
	if err := ensureDirectories(ctx, repo.ID, branch, req.Path, commitID, now); err != nil {
		return nil, err
	}

	// 9. Invalidate Redis cache for this repo/branch tree
	cachePattern := fmt.Sprintf("tree:%s:%s:*", repo.ID.Hex(), branch)
	database.RedisDelPattern(ctx, cachePattern)
	blobKey := fmt.Sprintf("blob:%s:%s:*", repo.ID.Hex(), req.Path)
	database.RedisDel(ctx, blobKey)

	return commit, nil
}

// ensureDirectories upserts a directory entry for every parent path of filePath.
func ensureDirectories(ctx context.Context, repoID bson.ObjectID, branch, filePath string, commitID bson.ObjectID, now time.Time) error {
	dir := path.Dir(filePath)
	for dir != "." && dir != "/" && dir != "" {
		dirMeta := &models.RepoFile{
            RepoID:      repoID,
            Path:        dir,
            Name:        path.Base(dir),
            Dir:         path.Dir(dir),
            Branch:      branch,
            CommitID:    commitID,
            IsDirectory: true,
            CreatedAt:   now,
            UpdatedAt:   now,
		}
		if err := repository.UpsertFile(ctx, dirMeta); err != nil {
			return err
		}
		dir = path.Dir(dir)
	}
	return nil
}

// GetTree returns the file listing for a given directory path in the repo
func GetTree(ctx context.Context, ownerID bson.ObjectID, repoSlug, branch, dirPath string) ([]models.FileTreeEntry, error) {
	repo, err := repository.FindRepoByOwnerAndSlug(ctx, ownerID, repoSlug)
	if err != nil || repo == nil {
		return nil, ErrRepoNotFound
	}
	if branch == "" {
		branch = repo.DefaultBranch
	}

	// Normalise dirPath
	if dirPath == "" || dirPath == "/" {
		dirPath = "."
	}

	// Check Redis Cache
	cacheKey := fmt.Sprintf("tree:%s:%s:%s", repo.ID.Hex(), branch, dirPath)
	if cached, ok := database.RedisGet(ctx, cacheKey); ok {
		var entries []models.FileTreeEntry
		if err := json.Unmarshal([]byte(cached), &entries); err == nil {
			return entries, nil
		}
	}

	// Fetch from DB
	files, err := repository.FindFilesByDir(ctx, repo.ID, branch, dirPath)
	if err != nil {
		return nil, err
	}

	// Get last commit for each file
	commits, _ := repository.FindCommitsByRepo(ctx, repo.ID, branch, 1)
	var lastCommit *models.CommitSummary
	if len(commits) > 0 {
		c := commits[0]
		lastCommit = &models.CommitSummary{
            Hash:    c.ShortHash,
            Message: c.Message,
            Author:  c.AuthorName,
            Date:    c.CreatedAt,
        }
	}

	entries := make([]models.FileTreeEntry, 0, len(files))
	for _, f := range files {
		t := "file"
		if f.IsDirectory {
			t = "dir"
		}
		entries = append(entries, models.FileTreeEntry{
            Name:       f.Name,
            Path:       f.Path,
            Type:       t,
            Size:       f.Size,
            MimeType:   f.MimeType,
            SHA:        f.SHA,
            LastCommit: lastCommit,
        })
	}

	// Store in Redis
	if data, err := json.Marshal(entries); err == nil {
		database.RedisSet(ctx, cacheKey, string(data), treeCacheTTL)
	}

	return entries, nil
}

// GetBlob returns the raw file content for a given path
func GetBlob(ctx context.Context, ownerID bson.ObjectID, repoSlug, branch, filePath string) ([]byte, *models.RepoFile, error) {
	repo, err := repository.FindRepoByOwnerAndSlug(ctx, ownerID, repoSlug)
	if err != nil || repo == nil {
		return nil, nil, ErrRepoNotFound
	}
	if branch == "" {
		branch = repo.DefaultBranch
	}

	// Check Redis blob cache
	cacheKey := fmt.Sprintf("blob:%s:%s:%s", repo.ID.Hex(), branch, filePath)
	if cached, ok := database.RedisGet(ctx, cacheKey); ok {
		meta, _ := repository.FindFileByPath(ctx, repo.ID, branch, filePath)
		return []byte(cached), meta, nil
	}

	// Fetch file metadata from DB
	meta, err := repository.FindFileByPath(ctx, repo.ID, branch, filePath)
	if err != nil {
		return nil, nil, ErrFileNotFound
	}

	// Fetch binary content from GridFS
	content, err := database.GridFSDownload(ctx, meta.GridFSID)
	if err != nil {
		return nil, nil, fmt.Errorf("gridfs download failed: %w", err)
	}

	// Only cache text files in Redis
	if meta.Encoding == "utf-8" {
		database.RedisSet(ctx, cacheKey, string(content), blobCacheTTL)
	}

	return content, meta, nil
}

// GetCommits returns recent commits for a repo branch
func GetCommits(ctx context.Context, ownerID bson.ObjectID, repoSlug, branch string, limit int64) ([]models.RepoCommit, error) {
	repo, err := repository.FindRepoByOwnerAndSlug(ctx, ownerID, repoSlug)
	if err != nil || repo == nil {
		return nil, ErrRepoNotFound
	}
	if branch == "" {
		branch = repo.DefaultBranch
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return repository.FindCommitsByRepo(ctx, repo.ID, branch, limit)
}

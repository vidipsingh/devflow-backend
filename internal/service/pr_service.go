package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"devflow-backend/internal/models"
	"devflow-backend/internal/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	ErrPRNotFound   = errors.New("pull request not found")
	ErrPRForbidden  = errors.New("not authorized")
	ErrPRDuplicate  = errors.New("PR already exists for this branch pair")
	ErrPRNotMergeable = errors.New("PR cannot be merged in current state")
)

func ListPRs(ctx context.Context, ownerID bson.ObjectID, repoSlug, state string) ([]models.PullRequest, error) {
	repo, err := repository.FindRepoByOwnerAndSlug(ctx, ownerID, repoSlug)
	if err != nil || repo == nil {
		return nil, ErrRepoNotFound
	}
	return repository.FindPRsRepo(ctx, repo.ID, state)
}

func CreatePR(ctx context.Context, ownerID bson.ObjectID, ownerName, repoSlug string, req models.CreatePRRequest) (*models.PullRequest, error) {
	repo, err := repository.FindRepoByOwnerAndSlug(ctx, ownerID, repoSlug)
	if err != nil || repo == nil {
		return nil, ErrRepoNotFound
	}

	number, err := repository.NextPRNumber(ctx, repo.ID)
	if err != nil {
		return nil, err
	}

	// Compute changed files by comparing commits on headBranch vs baseBranch
	headCommits, _ := repository.FindCommitsByRepo(ctx, repo.ID, req.HeadBranch, 100)
	baseCommits, _ := repository.FindCommitsByRepo(ctx, repo.ID, req.BaseBranch, 100)

	baseHashes := map[string]bool{}
	for _, c := range baseCommits {
		baseHashes[c.ShortHash] = true
	}

	changedFiles := map[string]bool{}
	additions := 0
	deletions := 0
	for _, c := range headCommits {
		if baseHashes[c.ShortHash] {
			continue
		}
		for _, f := range c.FilePaths {
			changedFiles[f] = true
		}
		additions += c.Additions
		deletions += c.Deletions
	}

	files := make([]string, 0, len(changedFiles))
	for f := range changedFiles {
		files = append(files, f)
	}

	pr := &models.PullRequest{
		Number:       number,
		RepoID:       repo.ID,
		RepoSlug:     repoSlug,
		Title:        req.Title,
		Body:         req.Body,
		State:        "open",
		HeadBranch:   req.HeadBranch,
		BaseBranch:   req.BaseBranch,
		AuthorID:     ownerID,
		AuthorName:   ownerName,
		Labels:       req.Labels,
		Comments:     []models.PRComment{},
		IsDraft:      req.IsDraft,
		IsMergeable:  true,
		ChangedFiles: files,
		Additions:    additions,
		Deletions:    deletions,
	}

	if err := repository.InsertPR(ctx, pr); err != nil {
		return nil, err
	}

	_ = repository.IncrementRepoStat(ctx, repo.ID, "openPRs", 1)
	return pr, nil
}

func GetPR(ctx context.Context, ownerID bson.ObjectID, repoSlug string, number int) (*models.PullRequest, error) {
	repo, err := repository.FindRepoByOwnerAndSlug(ctx, ownerID, repoSlug)
	if err != nil || repo == nil {
		return nil, ErrRepoNotFound
	}
	pr, err := repository.FindPRByNumber(ctx, repo.ID, number)
	if err != nil {
		return nil, ErrPRNotFound
	}
	return pr, nil
}

func UpdatePR(ctx context.Context, ownerID bson.ObjectID, repoSlug string, number int, req models.UpdatePRRequest) (*models.PullRequest, error) {
	repo, err := repository.FindRepoByOwnerAndSlug(ctx, ownerID, repoSlug)
	if err != nil || repo == nil {
		return nil, ErrRepoNotFound
	}
	pr, err := repository.FindPRByNumber(ctx, repo.ID, number)
	if err != nil {
		return nil, ErrPRNotFound
	}
	if pr.AuthorID != ownerID {
		return nil, ErrPRNotFound
	}

	upd := bson.M{}
	if req.Title != nil { upd["title"] = *req.Title }
	if req.Body != nil { upd["body"] = *req.Body }
	if req.State != nil {
		upd["state"] = *req.State
		if *req.State == "closed" {
			now := time.Now()
			upd["closedAt"] = now
			_ = repository.IncrementRepoStat(ctx, repo.ID, "openPRs", -1)
		}
	}
	if req.IsDraft != nil { upd["isDraft"] = *req.IsDraft }
	if req.Labels != nil { upd["labels"] = req.Labels }

	if err := repository.UpdatePR(ctx, pr.ID, upd); err != nil {
		return nil, err
	}
	return repository.FindPRByNumber(ctx, repo.ID, number)
}

func MergePR(ctx context.Context, ownerID bson.ObjectID, repoSlug string, number int, method string) (*models.PullRequest, error) {
	repo, err := repository.FindRepoByOwnerAndSlug(ctx, ownerID, repoSlug)
	if err != nil || repo == nil {
		return nil, ErrRepoNotFound
	}
	pr, err := repository.FindPRByNumber(ctx, repo.ID, number)
	if err != nil {
		return nil, ErrPRNotFound
	}
	if pr.State != "open" {
		return nil, ErrPRNotMergeable
	}

	// Copy all files from headBranch to baseBranch
	headFiles, err := repository.FindAllFilesByBranch(ctx, repo.ID, pr.HeadBranch)
	if err != nil {
		return nil, fmt.Errorf("failed to read head branch files: %w", err)
	}

	now := time.Now()
	for i := range headFiles {
		headFiles[i].Branch = pr.BaseBranch
		headFiles[i].UpdatedAt = now
		_ = repository.UpsertFile(ctx, &headFiles[i])
	}

	// Create merge commit
	mergeMsg := fmt.Sprintf("Merge pull requests %d from '%s' from %s into %s", pr.Number, pr.Title, pr.HeadBranch, pr.BaseBranch)
	commitID := bson.NewObjectID()
	commit := &models.RepoCommit{
		ID:         commitID,
		RepoID:     repo.ID,
		Branch:     pr.BaseBranch,
		Message:    mergeMsg,
		AuthorID:   ownerID,
		ShortHash:  commitID.Hex()[:7],
		FilePaths:  pr.ChangedFiles,
		Additions:  pr.Additions,
		Deletions:  pr.Deletions,
	}
	_ = repository.InsertCommit(ctx, commit)

	// Mark PR merged
	_ = repository.UpdatePR(ctx, pr.ID, bson.M{
		"state": "merged",
		"mergedAt": now,
		"mergedBy": ownerID,
	})
	_ = repository.IncrementRepoStat(ctx, repo.ID, "openPRs", -1)

	return repository.FindPRByNumber(ctx, repo.ID, number)
}

func DeletePR(ctx context.Context, ownerID bson.ObjectID, repoSlug string, number int) error {
	repo, err := repository.FindRepoByOwnerAndSlug(ctx, ownerID, repoSlug)
	if err != nil || repo == nil {
		return ErrRepoNotFound
	}
	pr, err := repository.FindPRByNumber(ctx, repo.ID, number)
	if err != nil {
		return ErrPRNotFound
	}
	if pr.AuthorID != ownerID {
		return ErrPRForbidden
	}
	return repository.DeletePR(ctx, pr.ID)
}

func AddPRCommentDirect(ctx context.Context, prID bson.ObjectID, c models.PRComment) error {
    return repository.AddPRComment(ctx, prID, c)
}

func UpdatePRCommentDirect(ctx context.Context, prID, commentID bson.ObjectID, body string) error {
    return repository.UpdatePRComment(ctx, prID, commentID, body)
}

func DeletePRCommentDirect(ctx context.Context, prID, commentID bson.ObjectID) error {
    return repository.DeletePRComment(ctx, prID, commentID)
}

package service

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"devflow-backend/internal/models"
	"devflow-backend/internal/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	ErrRepoNotFound   = errors.New("repository not found")
	ErrRepoForbidden  = errors.New("access denied")
	ErrRepoDuplicate  = errors.New("repository name already taken")
	ErrAlreadyStarred = errors.New("already starred")
	ErrNotStarred     = errors.New("not starred")
)

var slugRe = regexp.MustCompile(`[^a-z0-9\-]`)

func slugify(name string) string {
	s := strings.ToLower(strings.TrimSpace(name))
	s = strings.ReplaceAll(s, " ", "-")
	return slugRe.ReplaceAllString(s, "")
}

func ListRepositories(ctx context.Context, ownerID bson.ObjectID, visibility string) ([]models.Repository, error) {
	return repository.FindReposByOwner(ctx, ownerID, visibility)
}

// ListPublicRepositories returns all public repos, optionally filtered by search term.
func ListPublicRepositories(ctx context.Context, search string) ([]models.Repository, error) {
	return repository.FindAllPublicRepos(ctx, search)
}

// ResolveRepo resolves a repository by slug for a given caller.
// - If the caller owns a repo with that slug, that repo is returned (covers both public + private).
// - Otherwise it falls back to finding any PUBLIC repo with that slug (cross-user access).
// - Returns ErrRepoNotFound when no accessible repo exists.
// - Returns ErrRepoForbidden when the repo exists but is private and owned by someone else.
func ResolveRepo(ctx context.Context, callerID bson.ObjectID, slug string) (*models.Repository, error) {
	// 1. Try caller-owned first (handles both private + public of the caller).
	owned, err := repository.FindRepoByOwnerAndSlug(ctx, callerID, slug)
	if err != nil {
		return nil, err
	}
	if owned != nil {
		return owned, nil
	}
	// 2. Fall back: look for a PUBLIC repo with this slug from any owner.
	pub, err := repository.FindPublicRepoBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if pub != nil {
		return pub, nil
	}
	// 3. Check if there is a private repo (we just can't access it).
	any, err := repository.FindRepoBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if any != nil {
		return nil, ErrRepoForbidden
	}
	return nil, ErrRepoNotFound
}

func GetRepository(ctx context.Context, callerID bson.ObjectID, slug string) (*models.Repository, error) {
	return ResolveRepo(ctx, callerID, slug)
}

func CreateRepository(ctx context.Context, ownerID bson.ObjectID, ownerUsername string, req models.CreateRepoRequest) (*models.Repository, error) {
	slug := slugify(req.Name)
	if slug == "" {
		return nil, errors.New("invalid repository name")
	}

	existing, err := repository.FindRepoByOwnerAndSlug(ctx, ownerID, slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrRepoDuplicate
	}

	defaultBranch := req.DefaultBranch
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	now := time.Now()
	repo := &models.Repository{
		Name:          req.Name,
		Slug:          slug,
		FullName:      fmt.Sprintf("%s/%s", ownerUsername, slug),
		Description:   req.Description,
		OwnerID:       ownerID,
		OwnerType:     "user",
		Visibility:    req.Visibility,
		IsArchived:    false,
		IsFork:        false,
		Stats:         models.RepoStats{},
		DefaultBranch: defaultBranch,
		Branches:      []string{defaultBranch},
		Tags:          []string{},
		Topics:        []string{},
		Collaborators: []models.Collaborator{},
		Settings: models.RepoSettings{
			HasWiki:          true,
			HasIssues:        true,
			HasProjects:      true,
			AllowMergeCommit: true,
			AllowSquashMerge: true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := repository.CreateRepo(ctx, repo); err != nil {
		return nil, err
	}
	return repo, nil
}

func UpdateRepository(ctx context.Context, ownerID bson.ObjectID, slug string, req models.UpdateRepoRequest) (*models.Repository, error) {
	repo, err := repository.FindRepoByOwnerAndSlug(ctx, ownerID, slug)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}
	if repo.OwnerID != ownerID {
		return nil, ErrRepoForbidden
	}

	update := bson.M{"updatedAt": time.Now()}
	if req.Description != nil {
		update["description"] = *req.Description
	}
	if req.Visibility != nil {
		update["visibility"] = *req.Visibility
	}
	if req.Topics != nil {
		update["topics"] = req.Topics
	}
	if req.Settings != nil {
		update["settings"] = req.Settings
	}

	if err := repository.UpdateRepo(ctx, repo.ID, update); err != nil {
		return nil, err
	}
	return repository.FindRepoByOwnerAndSlug(ctx, ownerID, slug)
}

func DeleteRepository(ctx context.Context, ownerID bson.ObjectID, slug string) error {
	repo, err := repository.FindRepoByOwnerAndSlug(ctx, ownerID, slug)
	if err != nil {
		return err
	}
	if repo == nil {
		return ErrRepoNotFound
	}
	if repo.OwnerID != ownerID {
		return ErrRepoForbidden
	}
	return repository.DeleteRepo(ctx, repo.ID)
}

// PinRepository sets isPinned = pinned on the repository owned by ownerID
func PinRepository(ctx context.Context, ownerID bson.ObjectID, slug string, pinned bool) (*models.Repository, error) {
	repo, err := repository.FindRepoByOwnerAndSlug(ctx, ownerID, slug)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}
	if repo.OwnerID != ownerID {
		return nil, ErrRepoForbidden
	}
	if err := repository.UpdateRepo(ctx, repo.ID, bson.M{"isPinned": pinned}); err != nil {
		return nil, err
	}
	return repository.FindRepoByOwnerAndSlug(ctx, ownerID, slug)
}

// GetStarStatus returns whether the callerID has starred the repo identified by slug.
func GetStarStatus(ctx context.Context, callerID bson.ObjectID, slug string) (bool, int, error) {
	repo, err := repository.FindRepoBySlug(ctx, slug)
	if err != nil {
		return false, 0, err
	}
	if repo == nil {
		return false, 0, ErrRepoNotFound
	}
	for _, id := range repo.StarredBy {
		if id == callerID {
			return true, repo.Stats.Stars, nil
		}
	}
	return false, repo.Stats.Stars, nil
}

// StarRepository adds or removes callerID from the repo's starredBy list
// (idempotent per-user — double-star is rejected with ErrAlreadyStarred).
func StarRepository(ctx context.Context, callerID bson.ObjectID, slug string, star bool) (*models.Repository, error) {
	repo, err := repository.FindRepoBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}

	// Check current starred state for this user
	alreadyStarred := false
	for _, id := range repo.StarredBy {
		if id == callerID {
			alreadyStarred = true
			break
		}
	}

	if star && alreadyStarred {
		return nil, ErrAlreadyStarred
	}
	if !star && !alreadyStarred {
		return nil, ErrNotStarred
	}

	var update bson.M
	if star {
		update = bson.M{
			"$addToSet": bson.M{"starredBy": callerID},
			"$inc":      bson.M{"stats.stars": 1},
			"$set":      bson.M{"updatedAt": time.Now()},
		}
	} else {
		update = bson.M{
			"$pull": bson.M{"starredBy": callerID},
			"$inc":  bson.M{"stats.stars": -1},
			"$set":  bson.M{"updatedAt": time.Now()},
		}
	}

	if err := repository.UpdateRepoRaw(ctx, repo.ID, update); err != nil {
		return nil, err
	}
	return repository.FindRepoBySlug(ctx, slug)
}

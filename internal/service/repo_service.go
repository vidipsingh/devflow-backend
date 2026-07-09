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
	ErrRepoNotFound  = errors.New("repository not found")
	ErrRepoForbidden = errors.New("access denied")
	ErrRepoDuplicate = errors.New("repository name already taken")
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

func GetRepository(ctx context.Context, ownerID bson.ObjectID, slug string) (*models.Repository, error) {
	repo, err := repository.FindRepoByOwnerAndSlug(ctx, ownerID, slug)
	if err != nil {
		return nil, err
	}
	if repo == nil {
		return nil, ErrRepoNotFound
	}
	return repo, nil
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


package service

import (
	"context"
	"errors"

	"devflow-backend/internal/models"
	"devflow-backend/internal/repository"

	"go.mongodb.org/mongo-driver/v2/bson"
)

var (
	ErrLabelNotFound  = errors.New("label not found")
	ErrLabelForbidden = errors.New("not authorized to modify this label")
)

// ListRepoLabels returns all labels defined for a repository.
func ListRepoLabels(ctx context.Context, callerID bson.ObjectID, repoSlug string) ([]models.RepoLabel, error) {
	repo, err := ResolveRepo(ctx, callerID, repoSlug)
	if err != nil {
		return nil, ErrRepoNotFound
	}
	return repository.FindRepoLabels(ctx, repo.ID)
}

// CreateRepoLabel creates a new label for a repository.
// Only the repo owner may create labels.
func CreateRepoLabel(ctx context.Context, callerID bson.ObjectID, repoSlug string, req models.CreateRepoLabelRequest) (*models.RepoLabel, error) {
	repo, err := ResolveRepo(ctx, callerID, repoSlug)
	if err != nil {
		return nil, ErrRepoNotFound
	}
	if repo.OwnerID != callerID {
		return nil, ErrLabelForbidden
	}

	label := &models.RepoLabel{
		RepoID:      repo.ID,
		RepoSlug:    repoSlug,
		Name:        req.Name,
		Color:       req.Color,
		Description: req.Description,
	}
	if err := repository.InsertRepoLabel(ctx, label); err != nil {
		return nil, err
	}
	return label, nil
}

// UpdateRepoLabel partially updates an existing label.
// Only the repo owner may update labels.
func UpdateRepoLabel(ctx context.Context, callerID bson.ObjectID, repoSlug, labelID string, req models.UpdateRepoLabelRequest) (*models.RepoLabel, error) {
	repo, err := ResolveRepo(ctx, callerID, repoSlug)
	if err != nil {
		return nil, ErrRepoNotFound
	}
	if repo.OwnerID != callerID {
		return nil, ErrLabelForbidden
	}

	id, err := bson.ObjectIDFromHex(labelID)
	if err != nil {
		return nil, ErrLabelNotFound
	}

	label, err := repository.FindRepoLabelByID(ctx, id)
	if err != nil || label == nil {
		return nil, ErrLabelNotFound
	}
	if label.RepoID != repo.ID {
		return nil, ErrLabelForbidden
	}

	set := bson.M{}
	if req.Name != nil {
		set["name"] = *req.Name
	}
	if req.Color != nil {
		set["color"] = *req.Color
	}
	if req.Description != nil {
		set["description"] = *req.Description
	}
	if len(set) == 0 {
		return label, nil
	}
	if err := repository.UpdateRepoLabel(ctx, id, set); err != nil {
		return nil, err
	}
	// Re-fetch updated document
	return repository.FindRepoLabelByID(ctx, id)
}

// DeleteRepoLabel removes a label from a repository.
// Only the repo owner may delete labels.
func DeleteRepoLabel(ctx context.Context, callerID bson.ObjectID, repoSlug, labelID string) error {
	repo, err := ResolveRepo(ctx, callerID, repoSlug)
	if err != nil {
		return ErrRepoNotFound
	}
	if repo.OwnerID != callerID {
		return ErrLabelForbidden
	}

	id, err := bson.ObjectIDFromHex(labelID)
	if err != nil {
		return ErrLabelNotFound
	}

	label, err := repository.FindRepoLabelByID(ctx, id)
	if err != nil || label == nil {
		return ErrLabelNotFound
	}
	if label.RepoID != repo.ID {
		return ErrLabelForbidden
	}

	return repository.DeleteRepoLabel(ctx, id)
}

package service

import (
	"context"

	"devflow-backend/internal/models"
	"devflow-backend/internal/repository"
)

func GetTopSnippets(ctx context.Context, limit int, sortBy string) ([]models.Snippet, error) {
	if limit <= 0 || limit > 50 {
		limit = 6
	}
	return repository.FindPublishedSnippets(ctx, limit, sortBy)
}
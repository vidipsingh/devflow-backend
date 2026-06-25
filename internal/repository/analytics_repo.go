package repository

import (
	"context"
	"time"

	"devflow-backend/internal/database"
	"devflow-backend/internal/models"
)

func GetPlatformStats(ctx context.Context) (*models.PlatformStats, error) {
	db := database.GetDB()
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	users, err := db.Collection("users").EstimatedDocumentCount(timeout)
	if err != nil {
		return nil, err
	}

	repos, err := db.Collection("repositories").EstimatedDocumentCount(timeout)
	if err != nil {
		return nil, err
	}

	snippets, err := db.Collection("snippets").EstimatedDocumentCount(timeout)
	if err != nil {
		return nil, err
	}

	aiReviews, err := db.Collection("aiReviews").EstimatedDocumentCount(timeout)
	if err != nil {
		return nil, err
	}

	return &models.PlatformStats{
		TotalUsers: users,
		TotalRepositories: repos,
		TotalSnippets: snippets,
		TotalAIReviews: aiReviews,
	}, nil
}

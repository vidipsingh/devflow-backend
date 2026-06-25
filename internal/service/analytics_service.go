package service

import (
	"context"

	"devflow-backend/internal/models"
	"devflow-backend/internal/repository"
)

func GetPlatformStats(ctx context.Context) (*models.PlatformStats, error) {
	return repository.GetPlatformStats(ctx)
}

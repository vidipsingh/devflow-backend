package service

import (
	"context"

	"devflow-backend/internal/models"
	"devflow-backend/internal/repository"
)

func GetPricingPlans(ctx context.Context) ([]models.PricingPlan, error) {
	return repository.GetActivePricingPlans(ctx)
}

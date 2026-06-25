package repository

import (
	"context"
	"time"

	"devflow-backend/internal/database"
	"devflow-backend/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// GetActivePricingPlans returns all active plans sorted by their display order.
func GetActivePricingPlans(ctx context.Context) ([]models.PricingPlan, error) {
	col := database.Collection("pricingPlans")
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "order", Value: 1}})
	filter := bson.M{"isActive": true}

	cursor, err := col.Find(timeout, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(timeout)

	var plans []models.PricingPlan
	if err := cursor.All(timeout, &plans); err != nil {
		return nil, err
	}
	return plans, nil
}
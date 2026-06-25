package repository

import (
	"context"
	"time"

	"devflow-backend/internal/database"
	"devflow-backend/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// FindPublishedSnippets returns the top N published snippets sorted by downloads.
func FindPublishedSnippets(ctx context.Context, limit int, sortBy string) ([]models.Snippet, error) {
	col := database.Collection("snippets")
	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	sortField := "stats.downloads"
	if sortBy == "rating" {
		sortField = "stats.rating"
	} else if sortBy == "recent" {
		sortField = "publishedAt"
	}

	opts := options.Find().
	SetSort(bson.D{{Key: sortField, Value: -1}}).
	SetLimit(int64(limit))

	filter := bson.M{"status": "published"}

	cursor, err := col.Find(timeout, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(timeout)

	var snippets []models.Snippet
	if err := cursor.All(timeout, &snippets); err != nil {
		return nil, err
	}
	return snippets, nil
}

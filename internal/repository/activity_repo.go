package repository

import (
    "context"
    "time"

    "devflow-backend/internal/database"
    "devflow-backend/internal/models"

    "go.mongodb.org/mongo-driver/v2/bson"
    "go.mongodb.org/mongo-driver/v2/mongo/options"
)

// InsertActivity writes one ActivityEvent to the activity_feed collection.
func InsertActivity(ctx context.Context, ev *models.ActivityEvent) error {
    ev.ID = bson.NewObjectID()
    col := database.Collection("activity_feed")
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()
    _, err := col.InsertOne(ctx, ev)
    return err
}

// FindActivity returns the latest limit events for the given actorID (or all users if empty)
func FindActivity(ctx context.Context, actorID string, limit int64) ([]models.ActivityEvent, error) {
	col := database.Collection("activity_feed")
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	filter := bson.M{}
	if actorID != "" {
		filter["actorId"] = actorID
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
        SetLimit(limit)
	
	cur, err := col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var events []models.ActivityEvent
	if err := cur.All(ctx, &events); err != nil {
		return nil, err
	}
	return events, nil
}

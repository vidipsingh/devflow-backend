package repository

import (
	"context"
	"time"

	"devflow-backend/internal/database"
	"devflow-backend/internal/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func col() *mongo.Collection {
	return database.Collection("notifications")
}

// InsertNotification writes a single notification document.
func InsertNotification(ctx context.Context, n *models.Notification) error {
	n.ID = bson.NewObjectID()
	if n.CreatedAt.IsZero() {
		n.CreatedAt = time.Now()
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := database.Collection("notifications").InsertOne(ctx, n)
	return err
}

// FindNotifications returns the latest `limit` notifications for a recipient,
func FindNotifications(ctx context.Context, recipientID string, limit int64) ([]models.Notification, error) {
    ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
    defer cancel()

    opts := options.Find().
        SetSort(bson.D{{Key: "createdAt", Value: -1}}).
        SetLimit(limit)

    cur, err := database.Collection("notifications").Find(ctx,
        bson.M{"recipientId": recipientID}, opts)
    if err != nil {
        return nil, err
    }
    defer cur.Close(ctx)

    var notifs []models.Notification
    if err := cur.All(ctx, &notifs); err != nil {
        return nil, err
    }
    return notifs, nil
}

// CountUnread returns the number of unread notifications for a recipient.
func CountUnreadNotifications(ctx context.Context, recipientID string) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return database.Collection("notifications").CountDocuments(ctx, bson.M{"recipientId": recipientID, "read": false})
}

// MarkAllRead marks every unread notification for a recipient as read.
func MarkAllNotificationsRead(ctx context.Context, recipientID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err := database.Collection("notifications").UpdateMany(ctx,
        bson.M{"recipientId": recipientID, "read": false},
        bson.M{"$set": bson.M{"read": true}},
    )
    return err
}

// MarkOneRead marks a single notification as read.
func MarkOneNotificationRead(ctx context.Context, notifID, recipientID string) error {
	oid, err := bson.ObjectIDFromHex(notifID)
    if err != nil {
        return err
    }
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_, err = database.Collection("notifications").UpdateOne(ctx,
        bson.M{"_id": oid, "recipientId": recipientID},
        bson.M{"$set": bson.M{"read": true}},
	)
	return err
}

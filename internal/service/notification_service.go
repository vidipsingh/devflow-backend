package service

import (
    "context"

    "devflow-backend/internal/models"
    "devflow-backend/internal/repository"
)

// GetNotifications returns the latest notifications for the authenticated user.
func GetNotifications(ctx context.Context, recipientID string, limit int64) ([]models.Notification, error) {
    if limit <= 0 {
        limit = 20
    }
    notifs, err := repository.FindNotifications(ctx, recipientID, limit)
    if err != nil {
        return nil, err
    }
    if notifs == nil {
        notifs = []models.Notification{}
    }
    return notifs, nil
}

// GetUnreadCount returns the count of unread notifications.
func GetUnreadCount(ctx context.Context, recipientID string) (int64, error) {
	return repository.CountUnreadNotifications(ctx, recipientID)
}

// MarkAllRead marks every notification for the user as read.
func MarkAllNotificationsRead(ctx context.Context, recipientID string) error {
    return repository.MarkAllNotificationsRead(ctx, recipientID)
}

// MarkOneRead marks a single notification as read.
func MarkOneNotificationRead(ctx context.Context, notifID, recipientID string) error {
    return repository.MarkOneNotificationRead(ctx, notifID, recipientID)
}

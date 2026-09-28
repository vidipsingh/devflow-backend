package handlers

import (
    "net/http"
    "strconv"

    "devflow-backend/internal/service"
    "github.com/gin-gonic/gin"
)

// GET /api/v1/notifications?limit=20
func ListNotifications(c *gin.Context) {
	userID := c.GetString("userID")
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.ParseInt(limitStr, 10, 64)
	if err != nil || limit <= 0  {
		limit = 20
	}
	notifs, err := service.GetNotifications(c.Request.Context(), userID, limit)
	if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch notifications"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"data": notifs})
}

// GET /api/v1/notifications/unread-count
func GetUnreadCount(c *gin.Context) {
    userID := c.GetString("userID")
    count, err := service.GetUnreadCount(c.Request.Context(), userID)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to count notifications"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"count": count})
}

// PATCH /api/v1/notifications/read-all
func MarkAllRead(c *gin.Context) {
    userID := c.GetString("userID")
    if err := service.MarkAllNotificationsRead(c.Request.Context(), userID); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to mark notifications read"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "all notifications marked read"})
}

// PATCH /api/v1/notifications/:notifId/read
func MarkOneRead(c *gin.Context) {
    userID := c.GetString("userID")
    notifID := c.Param("notifId")
    if err := service.MarkOneNotificationRead(c.Request.Context(), notifID, userID); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "notification not found or not yours"})
        return
    }
    c.JSON(http.StatusOK, gin.H{"message": "notification marked read"})
}
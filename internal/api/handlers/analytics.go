package handlers

import (
	"net/http"
	"strconv"

	api "devflow-backend/internal/api/response"
	"devflow-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// GET /api/v1/analytics/overview?days=30
func GetAnalyticsOverview(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 || days > 365 {
		days = 30
	}
	data, err := service.GetAnalyticsOverview(c.Request.Context(), c.GetString("userID"), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch analytics"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// GetPlatformStats handles GET /api/v1/public/stats
// Returns total counts of users, repositories, snippets, and AI reviews.
func GetPlatformStats(c *gin.Context) {
	stats, err := service.GetPlatformStats(c.Request.Context())
	if err != nil {
		api.InternalError(c, "failed to fetch platform stats")
		return
	}
	api.OK(c, stats)
}
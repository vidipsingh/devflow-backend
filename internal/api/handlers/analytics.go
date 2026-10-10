package handlers

import (
	"net/http"
	"strconv"

	api "devflow-backend/internal/api/response"
	"devflow-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// parseAnalyticsParams extracts and validates common query params: days, granularity.
func parseAnalyticsParams(c *gin.Context) (int, string) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	if days <= 0 || days > 365 {
		days = 30
	}
	granularity := c.DefaultQuery("granularity", "daily")
	switch granularity {
	case "daily", "weekly", "monthly":
	default:
		granularity = "daily"
	}
	return days, granularity
}

// GET /api/v1/analytics/overview?days=30&granularity=daily
func GetAnalyticsOverview(c *gin.Context) {
	days, granularity := parseAnalyticsParams(c)
	data, err := service.GetAnalyticsOverview(c.Request.Context(), c.GetString("userID"), days, granularity)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch analytics"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": data})
}

// GET /api/v1/analytics/repos?days=30
// Returns activity summary across all repos owned by the caller.
func GetReposAnalytics(c *gin.Context) {
	days, _ := parseAnalyticsParams(c)
	data, err := service.GetAllReposAnalytics(c.Request.Context(), c.GetString("userID"), days)
	if err != nil {
		api.InternalError(c, "failed to fetch repos analytics")
		return
	}
	total := len(data)
	api.OK(c, gin.H{"repos": data, "total": total})
}

// GET /api/v1/analytics/repos/:repoName?days=30&granularity=daily
// Returns detailed analytics for a single repository.
func GetRepoAnalytics(c *gin.Context) {
	days, granularity := parseAnalyticsParams(c)
	repoName := c.Param("repoName")
	data, err := service.GetRepoAnalytics(c.Request.Context(), c.GetString("userID"), repoName, days, granularity)
	if err != nil {
		api.NotFound(c, "repository not found or no analytics data")
		return
	}
	api.OK(c, data)
}

// GET /api/v1/analytics/teams/:slug?days=30&granularity=daily
// Returns analytics for a team (membership required is enforced at service/repo layer).
func GetTeamAnalytics(c *gin.Context) {
	days, granularity := parseAnalyticsParams(c)
	slug := c.Param("slug")
	data, err := service.GetTeamAnalytics(c.Request.Context(), slug, days, granularity)
	if err != nil {
		api.NotFound(c, "team not found or no analytics data")
		return
	}
	api.OK(c, data)
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
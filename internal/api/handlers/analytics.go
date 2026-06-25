package handlers

import (
	api "devflow-backend/internal/api/response"
	"devflow-backend/internal/service"

	"github.com/gin-gonic/gin"
)

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
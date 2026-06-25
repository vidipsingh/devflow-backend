package handlers

import (
	"strconv"

	api  "devflow-backend/internal/api/response"
	"devflow-backend/internal/models"
	"devflow-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// GetSnippets handles GET /api/v1/marketplace/snippets
// Query params: limit (default 6), sort (downloads|rating|recent)
func GetSnippets(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "6")
	sortBy := c.DefaultQuery("sort", "downloads")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 6
	}

	snippets, err := service.GetTopSnippets(c.Request.Context(), limit, sortBy)
	if err != nil {
		api.InternalError(c, "failed to fetch snippets")
		return
	}

	// Return empty array not null when no snippets yet
	if snippets == nil {
		snippets = []models.Snippet{}
	}
	api.OK(c, snippets)
}
package handlers

import (
    "devflow-backend/internal/api/response"
    "devflow-backend/internal/service"
    "strconv"

    "github.com/gin-gonic/gin"
)

func TriggerAIReview(c *gin.Context) {
	ownerID, ok := mustOwnerID(c)
	if !ok {
		return
	}
	num, _ := strconv.Atoi(c.Param("number"))
	if err := service.TriggerReview(c.Request.Context(), ownerID, c.Param("name"), num); err != nil {
		response.NotFound(c, "pull request nto found")
		return
	}
	response.OK(c, gin.H{"message": "AI Review triggered"})
}

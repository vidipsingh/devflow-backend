package handlers

import (
	api "devflow-backend/internal/api/response"
	"devflow-backend/internal/models"
	"devflow-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// GetPricingPlans handles GET /api/v1/public/pricing
func GetPricingPlans(c *gin.Context) {
	plans, err := service.GetPricingPlans(c.Request.Context())
	if err != nil {
		api.InternalError(c, "failed to fetch pricing plans")
		return
	}
	if plans == nil {
		plans = []models.PricingPlan{}
	}
	api.OK(c, plans)
}
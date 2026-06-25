package api

import (
	"devflow-backend/internal/api/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://your-devflow-domain.vercel.app"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1 := r.Group("/api/v1")
	{
		// Public
		public := v1.Group("/public")
		{
			public.GET("/stats", handlers.GetPlatformStats)
			public.GET("/pricing", handlers.GetPricingPlans)
		}

		// Marketplace
		marketplace := v1.Group("/marketplace")
		{
			marketplace.GET("/snippets", handlers.GetSnippets)
		}
	}
	return r
}
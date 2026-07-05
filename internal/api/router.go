package api

import (
	"devflow-backend/internal/api/handlers"
	"devflow-backend/internal/api/middleware"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	originsEnv := os.Getenv("ALLOWED_ORIGINS")
    origins := []string{"http://localhost:3000"}
    if originsEnv != "" {
        origins = strings.Split(originsEnv, ",")
    }

	r.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
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

		// Auth
		auth := v1.Group("/auth")
		{
			auth.POST("/register", 	handlers.Register)
			auth.POST("/login", 	handlers.Login)
		}

		// OAuth routes
		auth.GET("/github",          handlers.GitHubRedirect)
		auth.GET("/github/callback", handlers.GitHubCallback)
		auth.GET("/google",          handlers.GoogleRedirect)
		auth.GET("/google/callback", handlers.GoogleCallback)


		// Marketplace
		marketplace := v1.Group("/marketplace")
		{
			marketplace.GET("/snippets", handlers.GetSnippets)
		}

		protected := v1.Group("/")
		protected.Use(middleware.RequireAuth)
		{
            protected.GET("/me", func(c *gin.Context) {
                c.JSON(200, gin.H{"data": gin.H{
                    "userId":   c.GetString("userID"),
                    "username": c.GetString("username"),
                    "email":    c.GetString("email"),
                    "plan":     c.GetString("plan"),
                }})
            })
        }
	}
	return r
}
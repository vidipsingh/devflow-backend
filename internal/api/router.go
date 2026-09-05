package api

import (
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"devflow-backend/internal/api/handlers"
	"devflow-backend/internal/api/middleware"
	"devflow-backend/internal/database"
	"devflow-backend/internal/repository"
	ws "devflow-backend/internal/websocket"
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

	reviewRepo := repository.NewReviewRepo(database.GetDB())
	reviewHandler := handlers.NewReviewHandler(reviewRepo, ws.GlobalReviewHub)
	userRepo := repository.NewUserRepo(database.GetDB())

	v1 := r.Group("/api/v1")
	{
		// Public
		public := v1.Group("/public")
		{
			public.GET("/stats", handlers.GetPlatformStats)
			public.GET("/pricing", handlers.GetPricingPlans)
			public.GET("/repos", handlers.ListPublicRepositories)
		}

		// Auth
		auth := v1.Group("/auth")
		{
			auth.POST("/register", handlers.Register)
			auth.POST("/login", handlers.Login)
		}

		// OAuth routes
		auth.GET("/github", handlers.GitHubRedirect)
		auth.GET("/github/callback", handlers.GitHubCallback)
		auth.GET("/google", handlers.GoogleRedirect)
		auth.GET("/google/callback", handlers.GoogleCallback)

		// Marketplace
		marketplace := v1.Group("/marketplace")
		{
			marketplace.GET("/snippets", handlers.GetSnippets)
		}

		// WebSocket — auth via ?token= query param
		v1.GET("/ws/pair/:sessionId", middleware.RequireAuthWS, handlers.PairSessionWS)
		v1.GET("/ws/review/:sessionId", middleware.RequireAuthWS, reviewHandler.WebSocketSignaling)

		protected := v1.Group("/")
		protected.Use(middleware.RequireAuth)
		{
			protected.GET("/me", func(c *gin.Context) {
					userID := c.GetString("userID")
					user, err := userRepo.FindByID(c.Request.Context(), userID)
					if err != nil {
						c.JSON(500, gin.H{"error": "failed to fetch user"})
						return
					}
					c.JSON(200, gin.H{"data": gin.H{
						"userId":   user.ID.Hex(),
						"username": user.Username,
						"email":    user.Email,
						"plan":     user.Plan,
					}})
				})
			repos := protected.Group("/repositories")
			{
				repos.GET("", handlers.ListRepositories)
				repos.POST("", handlers.CreateRepository)
				repos.GET("/:name", handlers.GetRepository)
				repos.PATCH("/:name", handlers.UpdateRepository)
				repos.DELETE("/:name", handlers.DeleteRepository)
				repos.PATCH("/:name/pin", handlers.PinRepository)
				repos.GET("/:name/star", handlers.GetStarStatus)
				repos.PATCH("/:name/star", handlers.StarRepository)
				repos.POST("/:name/files", handlers.UploadFile)
				repos.GET("/:name/tree", handlers.GetTree)
				repos.GET("/:name/blob", handlers.GetBlob)
				repos.GET("/:name/commits", handlers.GetCommits)

				// Issues
				issues := repos.Group("/:name/issues")
				{
					issues.GET("", handlers.ListIssues)
					issues.POST("", handlers.CreateIssue)
					issues.GET("/:number", handlers.GetIssue)
					issues.PATCH("/:number", handlers.UpdateIssue)
					issues.DELETE("/:number", handlers.DeleteIssue)
					issues.POST("/:number/comments", handlers.AddComment)
					issues.PATCH("/:number/comments/:commentId", handlers.UpdateComment)
					issues.DELETE("/:number/comments/:commentId", handlers.DeleteComment)
					issues.POST("/:number/reactions", handlers.ReactToIssue)
				}

				// Pull Requests
				prs := repos.Group("/:name/pulls")
				{
					prs.GET("", handlers.ListPRs)
					prs.POST("", handlers.CreatePR)
					prs.GET("/:number", handlers.GetPR)
					prs.GET("/:number/diff", handlers.GetPRDiff)
					prs.PATCH("/:number", handlers.UpdatePR)
					prs.DELETE("/:number", handlers.DeletePR)
					prs.POST("/:number/merge", handlers.MergePR)
					prs.POST("/:number/comments", handlers.AddPRComment)
					prs.PATCH("/:number/comments/:commentId", handlers.UpdatePRComment)
					prs.DELETE("/:number/comments/:commentId", handlers.DeletePRComment)
					prs.POST("/:number/ai-review", handlers.TriggerAIReview)
					
				}

				// Pair Programming Sessions
				pairSessions := protected.Group("/pair-sessions")
				{
					pairSessions.POST("", handlers.CreatePairSession)
					pairSessions.GET("", handlers.ListPairSessions)
					pairSessions.GET("/:sessionId", handlers.GetPairSession)
					pairSessions.POST("/:sessionId/join", handlers.JoinPairSession)
					pairSessions.POST("/:sessionId/end", handlers.EndPairSession)
				}

				// Review Sessions (PR pair review + WebRTC signaling)
				pr := protected.Group("/repos/:repoId/pulls/:prId")
				{
					pr.POST("/review-sessions", reviewHandler.CreateReviewSession)
					pr.GET("/review-sessions", reviewHandler.ListReviewSessions)
					pr.GET("/review-sessions/:sessionId", reviewHandler.GetReviewSession)
					pr.POST("/review-sessions/:sessionId/end", reviewHandler.EndReviewSession)
				}
				
				// Payments
				payments := protected.Group("/payments")
				{
					payments.POST("/orders",  handlers.CreatePaymentOrder)
					payments.POST("/verify",  handlers.VerifyPayment)
					payments.GET("/history",  handlers.GetPaymentHistory)
				}
			}
		}
	}
	return r
}

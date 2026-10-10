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
	// Init discussion hub singleton and start its event loop.
	discHub := ws.NewDiscussionHub()
	ws.GlobalDiscussionHub = discHub
	go discHub.Run()

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
			marketplace.GET("/snippets/:snippetId", handlers.GetSnippet)
			marketplace.GET("/snippets/:snippetId/reviews", handlers.GetSnippetReviews)
		}

		// WebSocket — auth via ?token= query param
		v1.GET("/ws/pair/:sessionId", middleware.RequireAuthWS, handlers.PairSessionWS)
		v1.GET("/ws/review/:sessionId", middleware.RequireAuthWS, reviewHandler.WebSocketSignaling)
		v1.GET("/teams/:slug/discussions/ws", middleware.RequireAuthWS, handlers.WsDiscussions(discHub))

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
			// Cross-repo aggregation endpoints
			protected.GET("/issues", handlers.ListMyIssues)
			protected.GET("/pulls", handlers.ListMyPRs)

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
				repos.POST("/:name/fork", handlers.ForkRepository)
				repos.GET("/:name/forks", handlers.ListForks)

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

				// Notifications
				notifs := protected.Group("/notifications")
				{
					notifs.GET("", handlers.ListNotifications)
					notifs.GET("/unread-count", handlers.GetUnreadCount)
					notifs.PATCH("/read-all", handlers.MarkAllRead)
					notifs.PATCH("/:notifId/read", handlers.MarkOneRead)
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
					payments.POST("/orders", handlers.CreatePaymentOrder)
					payments.POST("/verify", handlers.VerifyPayment)
					payments.GET("/history", handlers.GetPaymentHistory)
				}

				// Marketplace
				mp := protected.Group("/marketplace")
				{
					mp.GET("/my-snippets", handlers.GetMySnippets)
					mp.GET("/purchases", handlers.GetMyPurchases)

					mp.POST("/snippets", handlers.CreateSnippet)
					mp.PATCH("/snippets/:snippetId", handlers.UpdateSnippet)
					mp.DELETE("/snippets/:snippetId", handlers.DeleteSnippet)
					mp.PATCH("/snippets/:snippetId/publish", handlers.PublishSnippet)
					mp.GET("/snippets/:snippetId/download", handlers.DownloadSnippet)

					mp.POST("/snippets/:snippetId/purchase/order", handlers.CreatePurchaseOrder)
					mp.POST("/snippets/:snippetId/purchase/verify", handlers.VerifyPurchase)

					mp.POST("/snippets/:snippetId/reviews", handlers.CreateReview)
				}

				// Analytics
				protected.GET("/analytics/overview", handlers.GetAnalyticsOverview)

				// Teams
				teams := protected.Group("/teams")
				{
					teams.POST("", handlers.CreateTeam)
					teams.GET("", handlers.ListMyTeams)
					teams.GET("/discover", handlers.ListPublicTeams)
					teams.GET("/invites/me", handlers.ListMyInvites)
					teams.POST("/invites/:token/accept", handlers.AcceptInvite)
					teams.POST("/invites/:token/decline", handlers.DeclineInvite)

					teams.GET("/:slug", handlers.GetTeam)
					teams.PATCH("/:slug", handlers.UpdateTeam)
					teams.DELETE("/:slug", handlers.DeleteTeam)
					teams.GET("/:slug/sub-teams", handlers.ListSubTeams)

					teams.GET("/:slug/members/me", handlers.GetMyMembership)
					teams.GET("/:slug/members", handlers.ListTeamMembers)
					teams.PATCH("/:slug/members/:username/role", handlers.UpdateMemberRole)
					teams.DELETE("/:slug/members/:username", handlers.RemoveMember)
					teams.PATCH("/:slug/members/:username/ban", handlers.BanMember)
					teams.POST("/:slug/leave", handlers.LeaveTeam)
					teams.POST("/:slug/transfer", handlers.TransferOwnership)

					teams.POST("/:slug/invites", handlers.InviteMember)
					teams.GET("/:slug/invites", handlers.ListPendingInvites)
					teams.DELETE("/:slug/invites/:inviteId", handlers.RevokeInvite)

					teams.POST("/:slug/join-requests", handlers.RequestToJoin)
					teams.GET("/:slug/join-requests/me", handlers.GetMyJoinRequest)
					teams.GET("/:slug/join-requests", handlers.ListJoinRequests)
					teams.PATCH("/:slug/join-requests/:requestId", handlers.ReviewJoinRequest)

					teams.GET("/:slug/permissions/me", handlers.GetMyPermissions)

					teams.POST("/:slug/repos", handlers.AddTeamRepo)
					teams.GET("/:slug/repos", handlers.ListTeamRepos)
					teams.DELETE("/:slug/repos/:repoSlug", handlers.RemoveTeamRepo)

					teams.GET("/:slug/activity", handlers.GetTeamActivity)
					teams.GET("/:slug/audit-log", handlers.GetTeamAuditLog)

					teams.POST("/:slug/sub-teams", handlers.CreateSubTeam)

					// Team Discussions (REST)
					teams.GET("/:slug/discussions", handlers.ListDiscussions)
					teams.POST("/:slug/discussions", handlers.CreateDiscussion(discHub))
					teams.GET("/:slug/discussions/:discussionId", handlers.GetDiscussion)
					teams.PATCH("/:slug/discussions/:discussionId", handlers.UpdateDiscussion(discHub))
					teams.PATCH("/:slug/discussions/:discussionId/pin", handlers.PinDiscussion(discHub))
					teams.PATCH("/:slug/discussions/:discussionId/resolve", handlers.ResolveDiscussion(discHub))
					teams.DELETE("/:slug/discussions/:discussionId", handlers.DeleteDiscussion(discHub))
					teams.POST("/:slug/discussions/:discussionId/replies", handlers.AddReply(discHub))
					teams.DELETE("/:slug/discussions/:discussionId/replies/:replyId", handlers.DeleteReply(discHub))
					// WebSocket route is registered at v1 level above (uses RequireAuthWS)
				}
			}
		}
	}
	return r
}

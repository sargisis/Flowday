package router

import (
	"flowday/internal/auth"
	"flowday/internal/handlers"
	"flowday/internal/middleware"
	"log"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	// ---------- AUTH ----------
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register", middleware.RegisterRateLimiter(), auth.RegisterHandler)
		authGroup.POST("/login", middleware.AuthRateLimiter(), auth.LoginHandler)
		authGroup.POST("/forgot-password", middleware.ForgotPasswordRateLimiter(), auth.ForgotPasswordHandler)
		authGroup.POST("/reset-password", auth.ResetPasswordHandler)
		authGroup.POST("/refresh", auth.RefreshHandler)
		authGroup.POST("/logout", auth.LogoutHandler)
	}

	// ---------- PROTECTED ----------
	protected := v1.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/me", auth.GetMeHandler)
		protected.PATCH("/users/profile", handlers.UpdateProfile)
		protected.POST("/users/avatar", handlers.UploadAvatar)
		protected.POST("/users/email-change/request", handlers.RequestEmailChange)
		protected.POST("/users/change-email/verify", handlers.ConfirmEmailChange)
		protected.PATCH("/users/status", handlers.UpdateStatus)
		protected.GET("/users/:id", handlers.GetUserByID)
	}

	// Serve static files for uploads
	r.Static("/api/v1/uploads", "./uploads")

	// ---------- PROJECTS ----------
	projectsGroup := v1.Group("/projects")
	projectsGroup.Use(middleware.AuthMiddleware())
	{
		projectsGroup.GET("", handlers.GetProjects)
		projectsGroup.POST("", handlers.CreateProject)
		projectsGroup.DELETE("/:id", handlers.DeleteProject)
	}

	// ---------- TASKS ----------
	tasksGroup := v1.Group("/tasks")
	tasksGroup.Use(middleware.AuthMiddleware())
	{
		tasksGroup.GET("", handlers.GetTasks) // ?project_id=
		tasksGroup.GET("/all", handlers.GetAllTasks)
		tasksGroup.POST("", handlers.CreateTask)
		tasksGroup.PATCH("/:id", handlers.UpdateTask)
		tasksGroup.POST("/bulk-delete", handlers.BulkDeleteTasks)
		tasksGroup.DELETE("/:id", handlers.DeleteTask)
		tasksGroup.GET("/ids/:id", handlers.GetTask)

		// ✅ calendar API
		tasksGroup.GET("/by-date", handlers.GetTasksByDate) // ?date=YYYY-MM-DD

		// ✅ range API
		tasksGroup.GET("/by-range", handlers.GetTasksByRange) // ?from=YYYY-MM-DD&to=YYYY-MM-DD

		// ✅ stats API
		tasksGroup.GET("/stats", handlers.GetTaskStats)

		// 🤖 AI features
		tasksGroup.POST("/:id/decompose", handlers.DecomposeTask)
		tasksGroup.POST("/:id/enrich", handlers.EnrichTask)
	}

	// ---------- AI SERVICES ----------
	aiGroup := v1.Group("/ai")
	aiGroup.Use(middleware.AuthMiddleware())
	{
		aiGroup.POST("/health-advice", handlers.GetHealthAdvice)
		aiGroup.POST("/chat", handlers.HandleChat)
		aiGroup.GET("/history", handlers.HandleGetHistory)
		aiGroup.GET("/quota", handlers.HandleGetQuota)
		aiGroup.GET("/insights", handlers.HandleGetInsights)
	}

	// ---------- NOTIFICATIONS ----------
	notificationsGroup := v1.Group("/notifications")
	notificationsGroup.Use(middleware.AuthMiddleware())
	{
		notificationsGroup.GET("", handlers.GetNotificationsHandler)
		notificationsGroup.PATCH("/:id/read", handlers.MarkNotificationReadHandler)
	}

	// ---------- FOCUS SESSIONS ----------
	focusGroup := v1.Group("/focus")
	focusGroup.Use(middleware.AuthMiddleware())
	{
		focusGroup.POST("/sessions", handlers.CreateFocusSession)
		focusGroup.GET("/sessions", handlers.GetFocusSessions)
	}

	// ---------- ACTIVITY FEED ----------
	activityGroup := v1.Group("/activity")
	activityGroup.Use(middleware.AuthMiddleware())
	{
		activityGroup.GET("", handlers.GetActivityFeed)
	}

	// ---------- ACHIEVEMENTS & STREAKS ----------
	achievementGroup := v1.Group("/achievements")
	achievementGroup.Use(middleware.AuthMiddleware())
	{
		achievementGroup.GET("", handlers.GetAchievements)
	}

	streakGroup := v1.Group("/streak")
	streakGroup.Use(middleware.AuthMiddleware())
	{
		streakGroup.GET("", handlers.GetStreak)
	}

	// ---------- MESSAGING ----------
	messagesGroup := v1.Group("/messages")
	messagesGroup.Use(middleware.AuthMiddleware())
	{
		messagesGroup.POST("/send", handlers.SendMessage)
		messagesGroup.POST("/upload", handlers.UploadAttachment)
		messagesGroup.GET("/conversations", handlers.GetConversations)
		messagesGroup.GET("/history/:partnerID", handlers.GetChatHistory)
	}

	// ---------- PROJECT MEMBERS & INVITATIONS ----------
	invitationsGroup := v1.Group("/invitations")
	invitationsGroup.Use(middleware.AuthMiddleware())
	{
		invitationsGroup.GET("", handlers.GetMyInvitations)
		invitationsGroup.POST("/:id/accept", handlers.AcceptInvitation)
		invitationsGroup.POST("/:id/reject", handlers.RejectInvitation)
	}

	projectsGroup.GET("/:id/members", handlers.GetProjectMembers)
	projectsGroup.POST("/:id/members", handlers.InviteMember)
	projectsGroup.DELETE("/:id/members/:userID", handlers.RemoveMember)
	projectsGroup.PUT("/:id/members/:userID", handlers.UpdateMemberRole)

	// Log all routes
	for _, route := range r.Routes() {
		log.Printf("Route: %s %s", route.Method, route.Path)
	}
}

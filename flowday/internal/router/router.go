package router

import (
	"flowday/internal/auth"
	"flowday/internal/handlers"
	"flowday/internal/logger"
	"flowday/internal/middleware"
	"flowday/internal/websocket"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Setup(r *gin.Engine) {
	// ✅ STABILITY: Health check endpoints (no auth required)
	r.GET("/health", handlers.HealthCheckHandler)
	r.GET("/ready", handlers.ReadinessCheckHandler)
	r.GET("/live", handlers.LivenessCheckHandler)

	// ✅ METRICS: Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// ✅ REAL-TIME: WebSocket endpoint for real-time updates
	// Protected WebSocket connection - authentication handled in handler
	// Token can be passed via query parameter (?token=...) or Authorization header
	r.GET("/ws/connect", websocket.HandleWebSocket)

	// ✅ DOCUMENTATION: Swagger/OpenAPI documentation
	// Swagger UI will be available at /swagger/index.html
	// JSON spec at /swagger/doc.json
	// Note: Requires swag init to generate docs
	// r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")

	// ---------- AUTH ----------
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register", middleware.RegisterRateLimiter(), auth.RegisterHandler)
		authGroup.POST("/login", middleware.AuthRateLimiter(), auth.LoginHandler)
		authGroup.POST("/forgot-password", middleware.ForgotPasswordRateLimiter(), auth.ForgotPasswordHandler)
		// ✅ SECURITY: Add rate limiting to prevent brute-force attacks
		authGroup.POST("/reset-password", middleware.ResetPasswordRateLimiter(), auth.ResetPasswordHandler)
		authGroup.POST("/refresh", middleware.RefreshRateLimiter(), auth.RefreshHandler)
		authGroup.POST("/logout", middleware.AuthRateLimiter(), auth.LogoutHandler)
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
		protected.PATCH("/users/notifications", handlers.UpdateNotificationSettings)
		protected.POST("/users/notifications/test-slack", handlers.TestSlackWebhook)
		protected.GET("/users/:id", handlers.GetUserByID)

		// Slack OAuth Integration
		protected.GET("/slack/oauth/initiate", handlers.InitiateSlackOAuth)
		protected.POST("/slack/disconnect", handlers.DisconnectSlack)
		protected.GET("/slack/channels", handlers.GetSlackChannels)
		protected.PATCH("/slack/channel", handlers.UpdateSlackChannel)
		protected.POST("/slack/test", handlers.TestSlackOAuth)
	}

	// Public Slack OAuth callback (no auth required)
	v1.GET("/slack/oauth/callback", handlers.HandleSlackOAuthCallback)

	// Public Slack Interactivity endpoint (no auth required - Slack sends requests directly)
	v1.POST("/slack/interactivity", handlers.HandleSlackInteractivity)

	// ✅ FILES: Serve uploaded files
	// Public access for avatars (simpler - no auth needed)
	v1.GET("/uploads/*filepath", handlers.ServePublicFile)
	// Protected files (task/message attachments) - use different path if needed
	// protected.GET("/uploads/attachments/*filepath", handlers.ServeUploadedFile)

	// ---------- ANALYTICS ----------
	protected.GET("/analytics", handlers.GetTaskAnalytics)

	// Public Slack OAuth callback (no auth required)
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
		tasksGroup.GET("/search", handlers.SearchTasks) // ?q=keyword&project_id=xxx&status=in_progress&priority=high
		tasksGroup.POST("", handlers.CreateTask)
		tasksGroup.POST("/bulk-delete", handlers.BulkDeleteTasks)
		tasksGroup.POST("/bulk-update-status", handlers.BulkUpdateTasksStatus)
		tasksGroup.POST("/bulk-update-priority", handlers.BulkUpdateTasksPriority)
		tasksGroup.POST("/batch-dependencies", handlers.GetBatchTaskDependencies)

		// ✅ calendar API
		tasksGroup.GET("/by-date", handlers.GetTasksByDate) // ?date=YYYY-MM-DD

		// ✅ range API
		tasksGroup.GET("/by-range", handlers.GetTasksByRange) // ?from=YYYY-MM-DD&to=YYYY-MM-DD

		// ✅ stats API
		tasksGroup.GET("/stats", handlers.GetTaskStats)

		// 💬 Comments (must be before generic :id routes to avoid conflict)
		tasksGroup.POST("/:id/comments", handlers.CreateCommentHandler)
		tasksGroup.GET("/:id/comments", handlers.GetTaskCommentsHandler)

		// 📎 Attachments (must be before generic :id routes to avoid conflict)
		tasksGroup.POST("/:id/attachments", handlers.UploadTaskAttachment)
		tasksGroup.GET("/:id/attachments", handlers.GetTaskAttachments)
		tasksGroup.DELETE("/:id/attachments/:attachment_id", handlers.DeleteTaskAttachment)

		// 🤖 AI features (must be before generic :id routes to avoid conflict)
		tasksGroup.POST("/:id/decompose", handlers.DecomposeTask)
		tasksGroup.POST("/:id/enrich", handlers.EnrichTask)

		// ✅ NEW: Task Dependencies (must be before generic :id routes)
		tasksGroup.POST("/:id/dependencies", handlers.AddTaskDependency)
		tasksGroup.DELETE("/:id/dependencies", handlers.RemoveTaskDependency)
		tasksGroup.GET("/:id/dependencies", handlers.GetTaskDependencies)

		// Generic task operations (keep at the end)
		tasksGroup.PATCH("/:id", handlers.UpdateTask)
		tasksGroup.DELETE("/:id", handlers.DeleteTask)
		tasksGroup.GET("/ids/:id", handlers.GetTask)
	}

	// ---------- COMMENTS ----------
	commentsGroup := v1.Group("/comments")
	commentsGroup.Use(middleware.AuthMiddleware())
	{
		commentsGroup.PUT("/:comment_id", handlers.UpdateCommentHandler)
		commentsGroup.DELETE("/:comment_id", handlers.DeleteCommentHandler)
	}

	// ---------- AI SERVICES ----------
	aiGroup := v1.Group("/ai")
	aiGroup.Use(middleware.AuthMiddleware())
	{
		aiGroup.POST("/health-advice", handlers.GetHealthAdvice)
		aiGroup.POST("/chat", handlers.HandleChat)
		aiGroup.POST("/chat/stream", handlers.HandleChatStream)
		aiGroup.GET("/history", handlers.HandleGetHistory)
		aiGroup.DELETE("/history", handlers.HandleDeleteHistory)
		aiGroup.GET("/quota", handlers.HandleGetQuota)
		aiGroup.GET("/insights", handlers.HandleGetInsights)
	}

	// ---------- NOTIFICATIONS ----------
	notificationsGroup := v1.Group("/notifications")
	notificationsGroup.Use(middleware.AuthMiddleware())
	{
		notificationsGroup.GET("", handlers.GetNotificationsHandler)
		notificationsGroup.GET("/unread-count", handlers.GetUnreadCountHandler)
		notificationsGroup.PATCH("/:id/read", handlers.MarkNotificationReadHandler)
		notificationsGroup.POST("/batch-read", handlers.MarkNotificationsReadHandler)
		notificationsGroup.POST("/mark-all-read", handlers.MarkAllNotificationsReadHandler)
		notificationsGroup.DELETE("/old", handlers.DeleteOldNotificationsHandler)
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

	// ✅ NEW FEATURES: Task Templates
	templatesGroup := v1.Group("/templates")
	templatesGroup.Use(middleware.AuthMiddleware())
	{
		templatesGroup.POST("", handlers.CreateTaskTemplate)
		templatesGroup.GET("", handlers.GetTaskTemplates)
		templatesGroup.GET("/:id", handlers.GetTaskTemplate)
		templatesGroup.PATCH("/:id", handlers.UpdateTaskTemplate)
		templatesGroup.DELETE("/:id", handlers.DeleteTaskTemplate)
		templatesGroup.POST("/:id/create-task", handlers.CreateTaskFromTemplate)
	}

	// ✅ NEW FEATURES: Saved Views
	viewsGroup := v1.Group("/views")
	viewsGroup.Use(middleware.AuthMiddleware())
	{
		viewsGroup.POST("", handlers.CreateSavedView)
		viewsGroup.GET("", handlers.GetSavedViewsHandler)
		viewsGroup.GET("/:id", handlers.GetSavedViewHandler)
		viewsGroup.PATCH("/:id", handlers.UpdateSavedViewHandler)
		viewsGroup.DELETE("/:id", handlers.DeleteSavedViewHandler)
	}

	// ---------- TIME TRACKING ----------
	timeGroup := v1.Group("/time")
	timeGroup.Use(middleware.AuthMiddleware())
	{
		timeGroup.POST("/start", handlers.StartTimeEntry)
		timeGroup.POST("/stop", handlers.StopTimeEntry)
		timeGroup.GET("/entries", handlers.GetTimeEntries)
		timeGroup.DELETE("/entries/:id", handlers.DeleteTimeEntry)
		timeGroup.GET("/report", handlers.GetTimeReport)
	}

	// ✅ NEW FEATURES: Export/Import
	exportGroup := v1.Group("/export")
	exportGroup.Use(middleware.AuthMiddleware())
	{
		exportGroup.GET("/tasks/csv", handlers.ExportTasksCSV)
		exportGroup.GET("/tasks/json", handlers.ExportTasksJSON)
	}

	importGroup := v1.Group("/import")
	importGroup.Use(middleware.AuthMiddleware())
	{
		importGroup.POST("/tasks/csv", handlers.ImportTasksCSV)
		importGroup.POST("/tasks/json", handlers.ImportTasksJSON)
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
		logger.Log.Infof("Route: %s %s", route.Method, route.Path)
	}
}

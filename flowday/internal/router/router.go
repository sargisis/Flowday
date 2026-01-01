package router

import (
	"flowday/internal/auth"
	"flowday/internal/handlers"
	"flowday/internal/middleware"

	"github.com/gin-gonic/gin"
)

func Setup(r *gin.Engine) {
	v1 := r.Group("/api/v1")

	// ---------- AUTH ----------
	authGroup := v1.Group("/auth")
	{
		authGroup.POST("/register", auth.RegisterHandler)
		authGroup.POST("/login", auth.LoginHandler)
		authGroup.POST("/forgot-password", auth.ForgotPasswordHandler)
		authGroup.POST("/reset-password", auth.ResetPasswordHandler)
	}

	// ---------- PROTECTED ----------
	protected := v1.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/me", auth.GetMeHandler)
		protected.PATCH("/users/profile", handlers.UpdateProfile)
		protected.POST("/users/avatar", handlers.UploadAvatar)
		protected.POST("/users/email-change/request", handlers.RequestEmailChange)
		protected.POST("/users/email-change/confirm", handlers.ConfirmEmailChange)
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
		tasksGroup.POST("", handlers.CreateTask)
		tasksGroup.PATCH("/:id", handlers.UpdateTask)
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
	}

	// ---------- NOTIFICATIONS ----------
	notificationsGroup := v1.Group("/notifications")
	notificationsGroup.Use(middleware.AuthMiddleware())
	{
		notificationsGroup.GET("", handlers.GetNotificationsHandler)
		notificationsGroup.PATCH("/:id/read", handlers.MarkNotificationReadHandler)
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
}

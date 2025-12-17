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
	}

	// ---------- PROTECTED ----------
	protected := v1.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/me", func(c *gin.Context) {
			c.JSON(200, gin.H{
				"user_id": c.GetUint("user_id"),
			})
		})
	}



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
		tasksGroup.GET("", handlers.GetTasks)                 // ?project_id=
		tasksGroup.POST("", handlers.CreateTask)
		tasksGroup.PATCH("/:id", handlers.UpdateTask)
		tasksGroup.DELETE("/:id", handlers.DeleteTask)

		// ✅ calendar API
		tasksGroup.GET("/by-date", handlers.GetTasksByDate)   // ?date=YYYY-MM-DD

		// ✅ range API
		tasksGroup.GET("/by-range", handlers.GetTasksByRange) // ?from=YYYY-MM-DD&to=YYYY-MM-DD

		// ✅ stats API
		tasksGroup.GET("/stats", handlers.GetTaskStats)
	}
}

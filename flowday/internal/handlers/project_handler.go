package handlers

import (
	"net/http"

	"flowday/internal/dto"
	"flowday/internal/models"
	"flowday/internal/services"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func CreateProject(c *gin.Context) {
	var req dto.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	project, err := services.CreateProject(userID.(primitive.ObjectID), req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	services.LogActivity(userID.(primitive.ObjectID), models.ActivityProjectCreated, "Created new project: "+project.Name, map[string]string{"project_id": project.ID.Hex()})

	c.JSON(http.StatusCreated, project)
}

func GetProjects(c *gin.Context) {
	// Parse pagination query
	var pagination dto.PaginationQuery
	if err := c.ShouldBindQuery(&pagination); err != nil {
		// If pagination params are not provided, use defaults
		pagination = dto.PaginationQuery{}
	}

	userID, _ := c.Get("user_id")
	projects, meta, err := services.GetProjectsPaginated(userID.(primitive.ObjectID), pagination)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": projects,
		"meta": meta,
	})
}

func DeleteProject(c *gin.Context) {
	userID, _ := c.Get("user_id")
	projectID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project id format"})
		return
	}

	if err := services.DeleteProject(userID.(primitive.ObjectID), projectID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

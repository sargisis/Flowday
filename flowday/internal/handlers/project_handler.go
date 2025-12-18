package handlers

import (
	"net/http"

	"flowday/internal/dto"
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

	c.JSON(http.StatusCreated, project)
}

func GetProjects(c *gin.Context) {
	userID, _ := c.Get("user_id")
	projects, _ := services.GetProjects(userID.(primitive.ObjectID))
	c.JSON(http.StatusOK, projects)
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

package handlers

import (
	"net/http"
	"strconv"

	"flowday/internal/dto"
	"flowday/internal/services"

	"github.com/gin-gonic/gin"
)

func CreateProject(c *gin.Context) {
	var req dto.CreateProjectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	project, err := services.CreateProject(userID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, project)
}

func GetProjects(c *gin.Context) {
	userID := c.GetUint("user_id")
	projects, _ := services.GetProjects(userID)
	c.JSON(http.StatusOK, projects)
}

func DeleteProject(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, _ := strconv.Atoi(c.Param("id"))

	if err := services.DeleteProject(userID, uint(id)); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

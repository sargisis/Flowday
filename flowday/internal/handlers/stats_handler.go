package handlers

import (
	"net/http"

	"flowday/internal/services"

	"github.com/gin-gonic/gin"
)

func GetTaskStats(c *gin.Context) {
	stats, err := services.GetTaskStats(c.GetUint("user_id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

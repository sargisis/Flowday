package handlers

import (
	"net/http"
	"time"

	"flowday/internal/db"
	"flowday/internal/dto"
	"flowday/internal/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CreateSavedView handles POST /views
func CreateSavedView(c *gin.Context) {
	var req dto.CreateSavedViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	// Map DTO filters to model filters
	filters := models.ViewFilters{
		Status:      req.Filters.Status,
		Priority:    req.Filters.Priority,
		SearchQuery: req.Filters.SearchQuery,
		DueDateFrom: req.Filters.DueDateFrom,
		DueDateTo:   req.Filters.DueDateTo,
		HasSubtasks: req.Filters.HasSubtasks,
		IsRecurring: req.Filters.IsRecurring,
	}

	if req.Filters.AssigneeID != "" {
		assigneeID, err := primitive.ObjectIDFromHex(req.Filters.AssigneeID)
		if err == nil {
			filters.AssigneeID = &assigneeID
		}
	}
	if req.Filters.ProjectID != "" {
		projectID, err := primitive.ObjectIDFromHex(req.Filters.ProjectID)
		if err == nil {
			filters.ProjectID = &projectID
		}
	}

	view := models.SavedView{
		ID:        primitive.NewObjectID(),
		Name:      req.Name,
		UserID:    userID.(primitive.ObjectID),
		Filters:   filters,
		SortBy:    req.SortBy,
		SortOrder: req.SortOrder,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if req.ProjectID != "" {
		projectID, err := primitive.ObjectIDFromHex(req.ProjectID)
		if err == nil {
			view.ProjectID = &projectID
		}
	}

	_, err := db.SavedViews.InsertOne(c.Request.Context(), view)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, view)
}

// GetSavedViewsHandler handles GET /views
func GetSavedViewsHandler(c *gin.Context) {
	userID, _ := c.Get("user_id")
	projectID := c.Query("project_id")

	filter := bson.M{"user_id": userID.(primitive.ObjectID)}
	if projectID != "" {
		pid, err := primitive.ObjectIDFromHex(projectID)
		if err == nil {
			filter["project_id"] = pid
		}
	}

	cursor, err := db.SavedViews.Find(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(c.Request.Context())

	var views []models.SavedView
	if err = cursor.All(c.Request.Context(), &views); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, views)
}

// GetSavedViewHandler handles GET /views/:id
func GetSavedViewHandler(c *gin.Context) {
	viewID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid view id format"})
		return
	}

	userID, _ := c.Get("user_id")

	var view models.SavedView
	err = db.SavedViews.FindOne(c.Request.Context(), bson.M{
		"_id":     viewID,
		"user_id": userID.(primitive.ObjectID),
	}).Decode(&view)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "view not found"})
		return
	}

	c.JSON(http.StatusOK, view)
}

// UpdateSavedViewHandler handles PATCH /views/:id
func UpdateSavedViewHandler(c *gin.Context) {
	viewID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid view id format"})
		return
	}

	var req dto.UpdateSavedViewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")

	update := bson.M{"updated_at": time.Now()}
	if req.Name != "" {
		update["name"] = req.Name
	}
	if req.Description != "" {
		update["description"] = req.Description
	}
	if req.Filters.Status != nil && len(req.Filters.Status) > 0 {
		update["filters.status"] = req.Filters.Status
	}
	if req.Filters.Priority != nil && len(req.Filters.Priority) > 0 {
		update["filters.priority"] = req.Filters.Priority
	}
	if req.SortBy != "" {
		update["sort_by"] = req.SortBy
	}
	if req.SortOrder != "" {
		update["sort_order"] = req.SortOrder
	}

	result := db.SavedViews.FindOneAndUpdate(
		c.Request.Context(),
		bson.M{"_id": viewID, "user_id": userID.(primitive.ObjectID)},
		bson.M{"$set": update},
	)

	var view models.SavedView
	if err := result.Decode(&view); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "view not found"})
		return
	}

	c.JSON(http.StatusOK, view)
}

// DeleteSavedViewHandler handles DELETE /views/:id
func DeleteSavedViewHandler(c *gin.Context) {
	viewID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid view id format"})
		return
	}

	userID, _ := c.Get("user_id")

	result, err := db.SavedViews.DeleteOne(c.Request.Context(), bson.M{
		"_id":     viewID,
		"user_id": userID.(primitive.ObjectID),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "view not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

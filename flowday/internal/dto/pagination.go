package dto

import (
	"math"
)

type PaginationQuery struct {
	Page  int    `form:"page"`  // Page number (1-indexed)
	Limit int    `form:"limit"` // Items per page
	Sort  string `form:"sort"`  // Field to sort by (e.g., "created_at", "title")
	Order string `form:"order"` // Sort order: "asc" or "desc"
}

// ValidateAndSetDefaults sets default values if not provided
func (p *PaginationQuery) ValidateAndSetDefaults() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Limit < 1 {
		p.Limit = 20 // Default to 20 items per page
	}
	if p.Limit > 100 {
		p.Limit = 100 // Max 100 items per page
	}
	if p.Sort == "" {
		p.Sort = "created_at"
	}
	if p.Order == "" {
		p.Order = "desc"
	}
	if p.Order != "asc" && p.Order != "desc" {
		p.Order = "desc"
	}
}

// GetOffset calculates the offset based on page and limit
func (p *PaginationQuery) GetOffset() int {
	return (p.Page - 1) * p.Limit
}

// PaginationMeta contains pagination metadata for responses
type PaginationMeta struct {
	Page       int   `json:"page"`        // Current page (1-indexed)
	Limit      int   `json:"limit"`       // Items per page
	Total      int64 `json:"total"`       // Total number of items
	TotalPages int   `json:"total_pages"` // Total number of pages
	HasNext    bool  `json:"has_next"`    // Whether there is a next page
	HasPrev    bool  `json:"has_prev"`    // Whether there is a previous page
}

// NewPaginationMeta creates pagination metadata from query and total count
func NewPaginationMeta(query PaginationQuery, total int64) PaginationMeta {
	totalPages := int(math.Ceil(float64(total) / float64(query.Limit)))
	if totalPages < 1 {
		totalPages = 1
	}

	return PaginationMeta{
		Page:       query.Page,
		Limit:      query.Limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    query.Page < totalPages,
		HasPrev:    query.Page > 1,
	}
}

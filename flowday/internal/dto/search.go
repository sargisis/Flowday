package dto

// SearchTasksQuery contains query parameters for task search and filtering
type SearchTasksQuery struct {
	PaginationQuery
	Query         string `form:"q"`              // Full-text search query (searches in title and description)
	ProjectID     string `form:"project_id"`      // Filter by project ID
	Status        string `form:"status"`        // Filter by status (todo, in_progress, blocked, done)
	Priority      string `form:"priority"`      // Filter by priority (low, medium, high)
	HasDueDate    *bool  `form:"has_due_date"`   // Filter by presence of due date (true/false)
	AssigneeID    string `form:"assignee_id"`   // Filter by assignee
	Tag           string `form:"tag"`           // Filter by tag name
	CreatedAfter  string `form:"created_after"`  // Filter by creation date (YYYY-MM-DD)
	CreatedBefore string `form:"created_before"` // Filter by creation date (YYYY-MM-DD)
	UpdatedAfter  string `form:"updated_after"`  // Filter by update date (YYYY-MM-DD)
	UpdatedBefore string `form:"updated_before"` // Filter by update date (YYYY-MM-DD)
	SortByRelevance bool `form:"sort_by_relevance"` // Sort by relevance (for text search)
}

// ValidateAndSetDefaults validates search query and sets defaults
func (s *SearchTasksQuery) ValidateAndSetDefaults() {
	s.PaginationQuery.ValidateAndSetDefaults()
}

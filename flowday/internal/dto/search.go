package dto

// SearchTasksQuery contains query parameters for task search and filtering
type SearchTasksQuery struct {
	PaginationQuery
	Query      string `form:"q"`        // Full-text search query (searches in title and description)
	ProjectID  string `form:"project_id"` // Filter by project ID
	Status     string `form:"status"`   // Filter by status (todo, in_progress, blocked, done)
	Priority   string `form:"priority"` // Filter by priority (low, medium, high)
	HasDueDate *bool  `form:"has_due_date"` // Filter by presence of due date (true/false)
}

// ValidateAndSetDefaults validates search query and sets defaults
func (s *SearchTasksQuery) ValidateAndSetDefaults() {
	s.PaginationQuery.ValidateAndSetDefaults()
}

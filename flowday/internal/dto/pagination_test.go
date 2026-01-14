package dto

import (
	"testing"
)

func TestPaginationQuery_ValidateAndSetDefaults(t *testing.T) {
	tests := []struct {
		name     string
		query    PaginationQuery
		expected PaginationQuery
	}{
		{
			name:  "All defaults",
			query: PaginationQuery{},
			expected: PaginationQuery{
				Page:  1,
				Limit: 20,
				Sort:  "created_at",
				Order: "desc",
			},
		},
		{
			name: "Page less than 1",
			query: PaginationQuery{
				Page:  0,
				Limit: 10,
			},
			expected: PaginationQuery{
				Page:  1,
				Limit: 10,
				Sort:  "created_at",
				Order: "desc",
			},
		},
		{
			name: "Limit less than 1",
			query: PaginationQuery{
				Page:  2,
				Limit: 0,
			},
			expected: PaginationQuery{
				Page:  2,
				Limit: 20,
				Sort:  "created_at",
				Order: "desc",
			},
		},
		{
			name: "Limit exceeds maximum",
			query: PaginationQuery{
				Page:  1,
				Limit: 150,
			},
			expected: PaginationQuery{
				Page:  1,
				Limit: 100,
				Sort:  "created_at",
				Order: "desc",
			},
		},
		{
			name: "Valid custom values",
			query: PaginationQuery{
				Page:  3,
				Limit: 50,
				Sort:  "title",
				Order: "asc",
			},
			expected: PaginationQuery{
				Page:  3,
				Limit: 50,
				Sort:  "title",
				Order: "asc",
			},
		},
		{
			name: "Invalid order",
			query: PaginationQuery{
				Page:  1,
				Limit: 20,
				Order: "invalid",
			},
			expected: PaginationQuery{
				Page:  1,
				Limit: 20,
				Sort:  "created_at",
				Order: "desc",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.query.ValidateAndSetDefaults()

			if tt.query.Page != tt.expected.Page {
				t.Errorf("Page = %d, expected %d", tt.query.Page, tt.expected.Page)
			}
			if tt.query.Limit != tt.expected.Limit {
				t.Errorf("Limit = %d, expected %d", tt.query.Limit, tt.expected.Limit)
			}
			if tt.query.Sort != tt.expected.Sort {
				t.Errorf("Sort = %s, expected %s", tt.query.Sort, tt.expected.Sort)
			}
			if tt.query.Order != tt.expected.Order {
				t.Errorf("Order = %s, expected %s", tt.query.Order, tt.expected.Order)
			}
		})
	}
}

func TestPaginationQuery_GetOffset(t *testing.T) {
	tests := []struct {
		name     string
		query    PaginationQuery
		expected int
	}{
		{
			name: "Page 1, Limit 20",
			query: PaginationQuery{
				Page:  1,
				Limit: 20,
			},
			expected: 0,
		},
		{
			name: "Page 2, Limit 20",
			query: PaginationQuery{
				Page:  2,
				Limit: 20,
			},
			expected: 20,
		},
		{
			name: "Page 3, Limit 10",
			query: PaginationQuery{
				Page:  3,
				Limit: 10,
			},
			expected: 20,
		},
		{
			name: "Page 5, Limit 50",
			query: PaginationQuery{
				Page:  5,
				Limit: 50,
			},
			expected: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			offset := tt.query.GetOffset()
			if offset != tt.expected {
				t.Errorf("GetOffset() = %d, expected %d", offset, tt.expected)
			}
		})
	}
}

func TestNewPaginationMeta(t *testing.T) {
	tests := []struct {
		name     string
		query    PaginationQuery
		total    int64
		expected PaginationMeta
	}{
		{
			name: "First page, has next",
			query: PaginationQuery{
				Page:  1,
				Limit: 20,
			},
			total: 100,
			expected: PaginationMeta{
				Page:       1,
				Limit:      20,
				Total:      100,
				TotalPages: 5,
				HasNext:    true,
				HasPrev:    false,
			},
		},
		{
			name: "Last page, has prev",
			query: PaginationQuery{
				Page:  5,
				Limit: 20,
			},
			total: 100,
			expected: PaginationMeta{
				Page:       5,
				Limit:      20,
				Total:      100,
				TotalPages: 5,
				HasNext:    false,
				HasPrev:    true,
			},
		},
		{
			name: "Middle page, has both",
			query: PaginationQuery{
				Page:  3,
				Limit: 20,
			},
			total: 100,
			expected: PaginationMeta{
				Page:       3,
				Limit:      20,
				Total:      100,
				TotalPages: 5,
				HasNext:    true,
				HasPrev:    true,
			},
		},
		{
			name: "Empty result",
			query: PaginationQuery{
				Page:  1,
				Limit: 20,
			},
			total: 0,
			expected: PaginationMeta{
				Page:       1,
				Limit:      20,
				Total:      0,
				TotalPages: 1,
				HasNext:    false,
				HasPrev:    false,
			},
		},
		{
			name: "Exact page size",
			query: PaginationQuery{
				Page:  1,
				Limit: 20,
			},
			total: 20,
			expected: PaginationMeta{
				Page:       1,
				Limit:      20,
				Total:      20,
				TotalPages: 1,
				HasNext:    false,
				HasPrev:    false,
			},
		},
		{
			name: "Partial last page",
			query: PaginationQuery{
				Page:  3,
				Limit: 20,
			},
			total: 55,
			expected: PaginationMeta{
				Page:       3,
				Limit:      20,
				Total:      55,
				TotalPages: 3,
				HasNext:    false,
				HasPrev:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			meta := NewPaginationMeta(tt.query, tt.total)

			if meta.Page != tt.expected.Page {
				t.Errorf("Page = %d, expected %d", meta.Page, tt.expected.Page)
			}
			if meta.Limit != tt.expected.Limit {
				t.Errorf("Limit = %d, expected %d", meta.Limit, tt.expected.Limit)
			}
			if meta.Total != tt.expected.Total {
				t.Errorf("Total = %d, expected %d", meta.Total, tt.expected.Total)
			}
			if meta.TotalPages != tt.expected.TotalPages {
				t.Errorf("TotalPages = %d, expected %d", meta.TotalPages, tt.expected.TotalPages)
			}
			if meta.HasNext != tt.expected.HasNext {
				t.Errorf("HasNext = %v, expected %v", meta.HasNext, tt.expected.HasNext)
			}
			if meta.HasPrev != tt.expected.HasPrev {
				t.Errorf("HasPrev = %v, expected %v", meta.HasPrev, tt.expected.HasPrev)
			}
		})
	}
}

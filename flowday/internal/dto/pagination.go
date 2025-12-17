package dto

type PaginationQuery struct {
	Limit  int    `form:"limit"`
	Offset int    `form:"offset"`
	Order  string `form:"order"`
	Dir    string `form:"dir"`
}

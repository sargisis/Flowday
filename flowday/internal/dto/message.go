package dto

type SendMessageRequest struct {
	ReceiverID     string `json:"receiver_id" binding:"required"`
	Content        string `json:"content"`
	AttachmentURL  string `json:"attachment_url,omitempty"`
	AttachmentType string `json:"attachment_type,omitempty"`
}

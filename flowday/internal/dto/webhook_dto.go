package dto

// BMACWebhookPayload represents the payload sent by Buy Me a Coffee webhook
type BMACWebhookPayload struct {
	Type           string `json:"type"`            // "donation" or "subscription"
	SupporterEmail string `json:"supporter_email"` // Email of the payer
	SupportType    string `json:"support_type"`    // "coffee" or "subscription"
	Amount         string `json:"amount"`          // Amount paid
	Message        string `json:"message"`         // Optional message
	ProjectName    string `json:"project_name"`    // Project name if applicable
}

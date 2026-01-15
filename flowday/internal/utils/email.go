package utils

// SendVerificationCode sends a verification code email (legacy function, uses new template)
func SendVerificationCode(to string, code string, subject string, bodyTitle string) error {
	// Use new templated version
	return SendVerificationCodeTemplated(to, code, bodyTitle, "Please use this code to verify your request. This code will expire in 15 minutes.")
}

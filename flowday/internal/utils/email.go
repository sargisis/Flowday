package utils

import (
	"fmt"
	"net/smtp"
	"os"
)

// SendVerificationCode sends a verification code email (legacy function, uses new template)
func SendVerificationCode(to string, code string, subject string, bodyTitle string) error {
	// Use new templated version
	return SendVerificationCodeTemplated(to, code, bodyTitle, "Please use this code to verify your request. This code will expire in 15 minutes.")
}

// SendVerificationCodeTemplated sends a templated verification code email
func SendVerificationCodeTemplated(to string, code string, bodyTitle string, bodyMessage string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	if from == "" || password == "" || host == "" || port == "" {
		return fmt.Errorf("SMTP configuration missing in .env")
	}

	addr := host + ":" + port
	subject := fmt.Sprintf("Subject: %s\n", bodyTitle)
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf(`
		<html>
		<body>
			<h3>%s</h3>
			<p>Your verification code is: <b>%s</b></p>
			<p>%s</p>
		</body>
		</html>
	`, bodyTitle, code, bodyMessage)
	msg := []byte(subject + mime + body)

	auth := smtp.PlainAuth("", from, password, host)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

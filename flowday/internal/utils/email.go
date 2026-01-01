package utils

import (
	"errors"
	"fmt"
	"net/smtp"
	"os"
)

func SendVerificationCode(to string, code string, subject string, bodyTitle string) error {
	from := os.Getenv("SMTP_EMAIL")
	password := os.Getenv("SMTP_PASSWORD")
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")

	if from == "" || password == "" || host == "" || port == "" {
		return errors.New("SMTP configuration missing in .env")
	}

	addr := host + ":" + port
	mime := "MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n"
	body := fmt.Sprintf("<html><body><h3>%s</h3><p>Your verification code is: <b>%s</b></p><p>This code expires in 15 minutes.</p></body></html>", bodyTitle, code)
	msg := []byte("Subject: " + subject + "\n" + mime + body)

	auth := smtp.PlainAuth("", from, password, host)
	return smtp.SendMail(addr, auth, from, []string{to}, msg)
}

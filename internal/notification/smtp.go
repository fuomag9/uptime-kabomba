package notification

import (
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

// SMTPProvider sends email notifications
type SMTPProvider struct{}

func init() {
	RegisterProvider(&SMTPProvider{})
}

func (s *SMTPProvider) Name() string {
	return "smtp"
}

func (s *SMTPProvider) Send(ctx context.Context, notification *Notification, message *Message) error {
	// Get SMTP configuration
	host, _ := notification.Config["smtp_host"].(string)
	port, _ := notification.Config["smtp_port"].(float64)
	username, _ := notification.Config["smtp_username"].(string)
	password, _ := notification.Config["smtp_password"].(string)
	from, _ := notification.Config["from_email"].(string)
	to, _ := notification.Config["to_email"].(string)
	useTLS, _ := notification.Config["use_tls"].(bool)

	if host == "" || from == "" || to == "" {
		return fmt.Errorf("missing required SMTP configuration")
	}

	// Default port
	if port == 0 {
		if useTLS {
			port = 587
		} else {
			port = 25
		}
	}

	// Build email message
	subject := message.Title
	body := FormatMessage(message)

	msg := fmt.Sprintf("From: %s\r\n", from)
	msg += fmt.Sprintf("To: %s\r\n", to)
	msg += fmt.Sprintf("Subject: %s\r\n", subject)
	msg += "MIME-Version: 1.0\r\n"
	msg += "Content-Type: text/plain; charset=UTF-8\r\n"
	msg += "\r\n"
	msg += body

	// Parse recipient addresses
	recipients := strings.Split(to, ",")
	for i, r := range recipients {
		recipients[i] = strings.TrimSpace(r)
	}

	// Send email
	addr := fmt.Sprintf("%s:%d", host, int(port))

	var auth smtp.Auth
	if username != "" && password != "" {
		auth = smtp.PlainAuth("", username, password, host)
	}

	err := smtp.SendMail(addr, auth, from, recipients, []byte(msg))
	if err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	return nil
}

func (s *SMTPProvider) Validate(config map[string]interface{}) error {
	host, ok := config["smtp_host"].(string)
	if !ok || host == "" {
		return fmt.Errorf("smtp_host is required")
	}

	from, ok := config["from_email"].(string)
	if !ok || from == "" {
		return fmt.Errorf("from_email is required")
	}

	to, ok := config["to_email"].(string)
	if !ok || to == "" {
		return fmt.Errorf("to_email is required")
	}

	return nil
}

package email

import (
	"fmt"
	"net/smtp"
	"strings"
)

type Config struct {
	Host     string
	Port     int
	User     string
	Password string
	From     string
}

func SendInvite(cfg Config, toEmail, inviteURL string) error {
	subject := "You've been invited"
	body := fmt.Sprintf(
		"You have been invited to join the platform.\n\nClick the link below to accept your invitation:\n\n%s\n\nThis link expires in 7 days.",
		inviteURL,
	)

	msg := buildMessage(cfg.From, toEmail, subject, body)

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	if cfg.User != "" && cfg.Password != "" {
		auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)
		return smtp.SendMail(addr, auth, cfg.From, []string{toEmail}, []byte(msg))
	}

	// No-auth relay (dev/local)
	return smtp.SendMail(addr, nil, cfg.From, []string{toEmail}, []byte(msg))
}

func buildMessage(from, to, subject, body string) string {
	var b strings.Builder
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n")
	b.WriteString("\r\n")
	b.WriteString(body)
	return b.String()
}

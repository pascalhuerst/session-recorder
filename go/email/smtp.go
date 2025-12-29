package email

import (
	"bytes"
	"fmt"
	"html/template"
	"net/smtp"
	"strings"
)

// Config holds SMTP configuration
type Config struct {
	Host     string
	Port     string
	Username string
	Password string
	From     string
	FromName string
}

// Sender sends emails via SMTP
type Sender struct {
	config Config
}

// NewSender creates a new SMTP email sender
func NewSender(config Config) *Sender {
	return &Sender{config: config}
}

// ShareEmailData contains data for the share email template
type ShareEmailData struct {
	RecipientEmail string
	SessionName    string
	DownloadURL    string
	ExpiresIn      string
}

const shareEmailTemplate = `<!DOCTYPE html>
<html>
<head>
  <meta charset="utf-8">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; line-height: 1.6; color: #333; }
    .container { max-width: 600px; margin: 0 auto; padding: 20px; }
    .header { background: #7c3aed; color: white; padding: 20px; border-radius: 8px 8px 0 0; }
    .content { background: #f9fafb; padding: 20px; border: 1px solid #e5e7eb; }
    .button { display: inline-block; background: #7c3aed; color: white; padding: 12px 24px; text-decoration: none; border-radius: 6px; font-weight: 600; }
    .footer { padding: 20px; font-size: 12px; color: #6b7280; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <h1 style="margin: 0;">Session Recording Shared</h1>
    </div>
    <div class="content">
      <p>A session recording has been shared with you:</p>
      <p><strong>{{.SessionName}}</strong></p>
      <p style="margin: 24px 0;">
        <a href="{{.DownloadURL}}" class="button">Download Recording</a>
      </p>
      <p style="font-size: 14px; color: #6b7280;">
        This link will expire in {{.ExpiresIn}}.
      </p>
    </div>
    <div class="footer">
      <p>This email was sent from Session Recorder.</p>
    </div>
  </div>
</body>
</html>`

// SendShareEmail sends an email with a download link for a shared session
func (s *Sender) SendShareEmail(data ShareEmailData) error {
	tmpl, err := template.New("share").Parse(shareEmailTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, data); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	subject := fmt.Sprintf("Session Recording Shared: %s", data.SessionName)

	return s.sendHTML(data.RecipientEmail, subject, body.String())
}

// sendHTML sends an HTML email
func (s *Sender) sendHTML(to, subject, htmlBody string) error {
	from := s.config.From
	if s.config.FromName != "" {
		from = fmt.Sprintf("%s <%s>", s.config.FromName, s.config.From)
	}

	headers := make(map[string]string)
	headers["From"] = from
	headers["To"] = to
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=utf-8"

	var msg strings.Builder
	for k, v := range headers {
		msg.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}
	msg.WriteString("\r\n")
	msg.WriteString(htmlBody)

	addr := fmt.Sprintf("%s:%s", s.config.Host, s.config.Port)

	var auth smtp.Auth
	if s.config.Username != "" && s.config.Password != "" {
		auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	}

	return smtp.SendMail(addr, auth, s.config.From, []string{to}, []byte(msg.String()))
}

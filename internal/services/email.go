package services

import (
  "context"
  "fmt"
  "os"
  
  "github.com/sendgrid/sendgrid-go"
  "github.com/sendgrid/sendgrid-go/helpers/mail"

  "github.com/yungbote/slotter/backend/internal/logger"
)

type EmailService interface {
  SendEmail(ctx context.Context, toEmail string, subject string, plainText string, htmlContent string, emailType string) error
}

type emailService struct {
  log                         *logger.Logger
  client                      *sendgrid.Client
  fromSupportEmail            string
  fromInvitationEmail         string
  fromAuthorizationEmail      string
}

func NewEmailService(log *logger.Logger) (EmailService, error) {
  serviceLog := log.With("Service", "EmailService")
  apiKey := os.Getenv("SENDGRID_API_KEY")
  if apiKey == "" {
    return nil, fmt.Errorf("Missing SENDGRID_API_KEY environment variable")
  }
  fromSupport := os.Getenv("SENDGRID_SUPPORT_EMAIL")
  if fromSupport == "" {
    serviceLog.Warn("SENDGRID_SUPPORT_EMAIL not set; using fallback no-reply@slotter.ai")
    fromSupport = "no-reply@slotter.ai"
  }
  fromInv := os.Getenv("SENDGRID_INVITATION_EMAIL")
  if fromInv == "" {
    serviceLog.Warn("SENDGRID_INVITATION_EMAIL not set; using fallback invitation@slotter.ai")
    fromInv = "invitation@slotter.ai"
  }
  fromAuth := os.Getenv("SENDGRID_AUTHORIZATION_EMAIL")
  if fromAuth == "" {
    serviceLog.Warn("SENDGRID_AUTHORIZATION_EMAIL not set; using fallback authorization@slotter.ai")
    fromAuth = "authorization@slotter.ai"
  }
  client := sendgrid.NewSendClient(apiKey)

  return &emailService{
    log:                    serviceLog,
    client:                 client,
    fromSupportEmail:       fromSupport,
    fromInvitationEmail:    fromInv,
    fromAuthorizationEmail: fromAuth,
  }, nil
}

func (es *emailService) SendEmail(ctx context.Context, toEmail string, subject string, plainText string, htmlContent string, emailType string) error {
  var fromName = "Slotter"
  var fromEmail = es.fromSupportEmail
  switch emailType {
  case "invitation":
    fromName = "Slotter Invitation"
    fromEmail = es.fromInvitationEmail
  case "authorization":
    fromName = "Slotter Invitation"
    fromEmail = es.fromAuthorizationEmail
  case "support":
    fromName = "Slotter Support"
    fromEmail = es.fromSupportEmail
  default:

  }
  from := mail.NewEmail(fromName, fromEmail)
  to := mail.NewEmail("", toEmail)
  message := mail.NewSingleEmail(from, subject, to, plainText, htmlContent)
  response, err := es.client.SendWithContext(ctx, message)
  if err != nil {
    es.log.Warn("Sendgrid email send failed", "error", err)
    return err
  }
  es.log.Info("Email sent", "to", toEmail, "statusCode", response.StatusCode)
  return nil
}








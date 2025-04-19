package services

import (
  "context"
  "fmt"
  "os"
  
  twilio "github.com/twilio/twilio-go"
  openapi "github.com/twilio/twilio-go/rest/api/v2010"
  "github.com/slotter-org/slotter-backend/internal/logger"
)

type TextService interface {
  SendText(ctx context.Context, toNumber string, body string) error
}

type textService struct {
  log         *logger.Logger
  client      *twilio.RestClient
  from        string
}

func NewTextService(log *logger.Logger) (TextService, error) {
  serviceLog := log.With("service", "TextService")
  accountSid := os.Getenv("TWILIO_ACCOUNT_SID")
  authToken := os.Getenv("TWILIO_AUTH_TOKEN")
  fromNumber := os.Getenv("TWILIO_FROM_NUMBER")

  if accountSid == "" || authToken == "" || fromNumber == "" {
    return nil, fmt.Errorf("Missing Twilio  env variables: TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN, TWILIO_FROM_NUMBER")
  }
  
  client := twilio.NewRestClientWithParams(twilio.ClientParams{
    Username: accountSid,
    Password: authToken,
  })

  ts := &textService{
    log:        serviceLog,
    client:     client,
    from:       fromNumber,
  }
  return ts, nil
}

func (ts *textService) SendText(ctx context.Context, toNumber string, body string) error {
  params := &openapi.CreateMessageParams{}
  params.SetTo(toNumber)
  params.SetFrom(ts.from)
  params.SetBody(body)

  resp, err := ts.client.Api.CreateMessage(params)
  if err != nil {
    ts.log.Warn("Failed to send Text via Twilio", "error", err)
    return err
  }
  ts.log.Info("Successfully sent Text via Twilio", "toNumber", toNumber, "sid", *resp.Sid, "status", *resp.Status)
  return nil
}

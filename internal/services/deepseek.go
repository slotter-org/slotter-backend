package services

import (
  "context"
  "encoding/json"
  "fmt"
  "io"
  "net/http"
  "os"
  "time"

  "github.com/slotter-org/slotter-backend/internal/logger"
)

type DeepseekService interface {
  QueryDeepseek(ctx context.Context, query string) (DeepseekResponse, error)
}

type deepseekService struct {
  log               *logger.Logger
  client            *http.Client
  baseURL           string
  apiKey            string
}

type DeepseekResponse struct {
  Status      string        `json:"status"`
  Message     string        `json:"message"`
}

func NewDeepseekService(log *logger.Logger) (DeepseekService, error) {
  serviceLog := log.With("service", "DeepseekService")
  baseURL := os.Getenv("DEEPSEEK_R1_API_URL")
  if baseURL == "" {
    return nil, fmt.Errorf("missing DEEPSEEK_R1_API_URL environment variable")
  }
  apiKey := os.Getenv("DEEPSEEK_R1_API_KEY")
  if apiKey == "" {
    serviceLog.Warn("DEEPSEEK_R1_API_KEY not set; calls might fail or be unauthorized")
  }
  httpClient := &http.Client{
    Timeout: 15 * time.Second,
  }
  return &deepseekService{
    log:      serviceLog,
    client:   httpClient,
    baseURL:  baseURL,
    apiKey:   apiKey,
  }, nil
}

func (ds *deepseekService) QueryDeepseek(ctx context.Context, query string) (DeepseekResponse, error) {
  var out DeepseekResponse

  reqURL := fmt.Sprintf("%s/api/r1?search=%s", ds.baseURL, query)
  req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
  if err != nil {
    ds.log.Warn("failed to build new request", "error", err)
    return out, err
  }
  if ds.apiKey != "" {
    req.Header.Set("Authorization", "Bearer"+ds.apiKey)
  }
  resp, err := ds.client.Do(req)
  if err != nil {
    ds.log.Warn("failed to call deepseek r1", "error", err)
    return out, err
  }
  defer resp.Body.Close()

  if resp.StatusCode < 200 || resp.StatusCode > 299 {
    bodyBytes, _ := io.ReadAll(resp.Body)
    ds.log.Warn("deepseek r1 responded with non-2xx", "statusCode", resp.StatusCode, "body", string(bodyBytes))
    return out, fmt.Errorf("deepseek r1 HTTP %d: %s", resp.StatusCode, string(bodyBytes))
  }
  bodyBytes, err := io.ReadAll(resp.Body)
  if err != nil {
    ds.log.Warn("failed to read deepseek r1 response body", "error", err)
    return out, err
  }
  ds.log.Info("Deepseek R1 call success", "response", out)
  return out, nil
}























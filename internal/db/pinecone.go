package db

import (
  "context"
  "fmt"
  "os"

  "github.com/pinecone-io/go-pinecone/v3/pinecone"
  "github.com/slotter-org/slotter-backend/internal/logger"
)

type PineconeService struct {
  log               *logger.Logger
  pineconeClient    *pinecone.Client
  indexName         string
  projectName       string
}

func NewPineconeService(log *logger.Logger) (*PineconeService, error) {
  serviceLog := log.With("service", "PineconeService")
  apiKey := os.Getenv("PINECONE_API_KEY")
  environment := os.Getenv("PINECONE_ENVIRONMENT")
  indexName := os.Getenv("PINECONDE_INDEX_NAME")

  if apiKey == "" || environment == "" || indexName == "" {
    return nil, fmt.Errorf("missing Pinecone environment variables: PINECONE_API_KEY, PINECONE_ENVIRONMENT, PINECONDE_INDEX_NAME")
  }

  pineClient, err := pinecone.NewClient(pinecone.Config{
    APIKey:             apiKey,

  })
}

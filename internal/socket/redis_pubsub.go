package socket

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/slotter-org/slotter-backend/internal/logger"
)

type RedisPubSub struct {
	log					*logger.Logger
	client			*redis.Client
	channel			string
	cancelFunc	context.CancelFunc
	mu					sync.Mutex
}

func NewRedisPubSub(log *logger.Logger, address, password, channel string) (*RedisPubSub, error) {
	opt := &redis.Options{
		Addr:					address,
		Password:			password,
		DB:						0,
	}
	rdb := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping failed: %w", err)
	}
	return &RedisPubSub{
		log:				log.With("component", "RedisPubSub"),
		client:			rdb,
		channel:		channel,
	}, nil
}

func (rp *RedisPubSub) StartSubscriber(hub *Hub) error {
	ctx, cancel := context.WithCancel(context.Background())
	rp.cancelFunc = cancel

	pubsub := rp.client.Subscribe(ctx, rp.channel)

	if _, err := pubsub.Receive(ctx); err != nil {
		return fmt.Errorf("failed to subscribe to redis channel: %w", err)
	}
	rp.log.Info("RedisPubSub subscribed successfully", "channel", rp.channel)

	go func() {
		ch := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				rp.log.Debug("Redis pubsub context done, stopping subscription goroutine")
				return
			case msg, ok := <-ch:
				if !ok {
					rp.log.Debug("PubSub channel closed, stopping subscription goroutine")
					return
				}
				broadcastMsg, err := decodePubSubMessage(msg.Payload)
				if err != nil {
					rp.log.Warn("Failed to decode pubsub message", "error", err)
					continue
				}
				hub.localBroadcast(broadcastMsg)
			}
		}
	}()
	return nil
}

func (rp *RedisPubSub) Publish(msg Message) error {
	payload, err := encodePubSubMessage(msg)
	if err != nil {
		rp.log.Warn("failed to encode message for redis", "error", err)
		return err
	}
	return rp.client.Publish(context.Background(), rp.channel, payload).Err()
}

func (rp *RedisPubSub) Stop() {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	if rp.cancelFunc != nil {
		rp.cancelFunc()
		rp.cancelFunc = nil
	}
}

func encodePubSubMessage(m Message) (string, error) {
	raw, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func decodePubSubMessage(payload string) (Message, error) {
	var msg Message
	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		return msg, fmt.Errorf("json unmarshal failed: %w", err)
	}
	return msg, nil
}

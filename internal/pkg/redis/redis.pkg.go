package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-boilerplate/internal/pkg/logger"
	"time"

	_redis "github.com/redis/go-redis/v9"
)

func Setup(ctx context.Context, config *Config) (*Client, error) {
	clientCtx, cancel := context.WithCancel(ctx)

	r := &Client{
		cancel: cancel,
		ctx:    clientCtx,
		config: config,
	}

	// Connect to IRedis
	if err := r.connect(); err != nil {
		cancel() // Ensure cleanup if initialization fails
		logger.Error.Println(err)
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	// Start the reconnect handler
	go r.reconnectHandler()

	return r, nil
}

func (r *Client) connect() error {
	r.Client = _redis.NewClient(&_redis.Options{
		Addr:     fmt.Sprintf("%s:%d", r.config.Host, r.config.Port),
		Username: r.config.Username,
		Password: r.config.Password,
		PoolSize: r.config.PoolSize,
	})

	if err := r.Client.Ping(r.ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis: %w", err)
	}

	return nil
}

func (r *Client) reconnect() error {
	if err := r.Client.Ping(r.ctx).Err(); err != nil {
		return r.connect()
	}

	go r.reconnectHandler()
	return nil
}

func (r *Client) reconnectHandler() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			logger.Info.Println("Reconnect handler shutting down...")
			return
		case <-ticker.C:
			if err := r.Client.Ping(r.ctx).Err(); err != nil {
				logger.Warning.Printf("IRedis connection lost: %v. Attempting to reconnect...", err)

				attempt := 1
				for {
					logger.Warning.Printf("Reconnect attempt #%d...", attempt)
					// Reinitialize the IRedis general
					if err = r.reconnect(); err == nil {
						logger.Info.Println("Reconnected to IRedis.")
						break
					}
					time.Sleep(time.Duration(attempt) * time.Second) // Exponential backoff
					logger.Warning.Printf("Reconnect attempt failed: %v", err)
					attempt++
				}
				return
			}
		}
	}
}

// Close gracefully shuts down the IRedis general connection.
func (r *Client) Close() error {
	r.cancel()
	return r.Client.Close()
}

// Set stores a key-value pair with an expiration time.
func (r *Client) Set(key string, value any, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	err = r.Client.Set(r.ctx, key, data, expiration).Err()
	if err != nil {
		if err = r.reconnect(); err != nil {
			return fmt.Errorf("failed to set key %s: %w", key, err)
		}
	}
	return err
}

// Get retrieves the value of a key.
func (r *Client) Get(key string) (string, error) {
	result, err := r.Client.Get(r.ctx, key).Result()
	if err != nil {
		if errors.Is(err, NilType) {
			return "", nil // Key does not exist
		}
		if err = r.reconnect(); err != nil {
			return "", fmt.Errorf("failed to get key %s: %w", key, err)
		}
	}
	return result, nil
}

// Del deletes a key from IRedis.
func (r *Client) Del(key string) error {
	err := r.Client.Del(r.ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete key %s: %w", key, err)
	}
	return nil
}

// Expire sets a timeout on a key.
func (r *Client) Expire(key string, expiration time.Duration) error {
	err := r.Client.Expire(r.ctx, key, expiration).Err()
	if err != nil {
		return fmt.Errorf("failed to set expiration on key %s: %w", key, err)
	}
	return nil
}

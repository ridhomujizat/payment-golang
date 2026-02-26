package rabbitmq

import (
	"context"
	"fmt"
	"go-boilerplate/internal/pkg/logger"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type ConnectionManager struct {
	conn          *amqp.Connection
	mu            sync.Mutex
	url           string
	isConnected   bool
	retryInterval time.Duration
	ctx           context.Context
	cancel        context.CancelFunc
}

type QueueConfig struct {
	Durable    bool
	AutoDelete bool
	Exclusive  bool
	NoWait     bool
	Args       amqp.Table
}

func DefaultQueueConfig() *QueueConfig {
	config := &QueueConfig{
		Durable:    true,
		AutoDelete: false,
		Exclusive:  false,
		NoWait:     false,
		Args:       nil,
	}

	return config
}

type Config struct {
	Username string
	Password string
	Host     string
	Port     int
	URI      string
}

func NewConnectionManager(ctx context.Context, config *Config) (*ConnectionManager, error) {
	ctx, cancel := context.WithCancel(ctx)

	var url string
	if config.URI != "" {
		url = config.URI
	} else {
		url = fmt.Sprintf("amqp://%s:%s@%s:%d/", config.Username, config.Password, config.Host, config.Port)
	}

	cm := &ConnectionManager{
		url:           url,
		retryInterval: time.Second * 2,
		ctx:           ctx,
		cancel:        cancel,
	}

	if err := cm.connect(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create connection: %w", err)
	}

	return cm, nil
}

func (cm *ConnectionManager) connect() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.isConnected {
		return nil
	}

	if err := cm.ctx.Err(); err != nil {
		return fmt.Errorf("context canceled: %w", err)
	}

	conn, err := amqp.Dial(cm.url)
	if err != nil {
		return fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	cm.conn = conn
	cm.isConnected = true

	go cm.connectionMonitor()

	return nil
}

func (cm *ConnectionManager) connectionMonitor() {
	connErr := make(chan *amqp.Error)
	cm.conn.NotifyClose(connErr)

	for {
		select {
		case err := <-connErr:
			if err != nil {
				cm.mu.Lock()
				cm.isConnected = false
				cm.mu.Unlock()
				logger.Warning.Printf("Connection lost: %v. Attempting to reconnect...\n", err)

				for {
					select {
					case <-cm.ctx.Done():
						return
					default:
					}

					// Always sleep before attempting reconnect to avoid CPU spike
					time.Sleep(cm.retryInterval)

					if err := cm.connect(); err != nil {
						logger.Warning.Printf("Failed to reconnect: %v. Retrying in %v...\n",
							err, cm.retryInterval)
						continue
					}

					logger.Info.Println("Reconnected successfully")
					return // Exit monitor, new goroutine will be started by connect()
				}
			}
		case <-cm.ctx.Done():
			return
		}
	}
}

func (cm *ConnectionManager) GetConnection() *amqp.Connection {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.ctx.Err() != nil {
		return nil
	}

	return cm.conn
}

func (cm *ConnectionManager) Close() error {
	cm.cancel()

	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.conn != nil {
		if err := cm.conn.Close(); err != nil {
			return fmt.Errorf("failed to close connection: %w", err)
		}
		cm.conn = nil
	}

	cm.isConnected = false
	return nil
}

func (cm *ConnectionManager) IsClosed() bool {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	return cm.ctx.Err() != nil || !cm.isConnected
}

package rabbitmq

import (
	"context"
	"fmt"
	"go-boilerplate/internal/pkg/logger"
	"sync"
	"sync/atomic"
	"time"

	"github.com/panjf2000/ants/v2"

	amqp "github.com/rabbitmq/amqp091-go"
)

type MessageHandler func(msg *amqp.Delivery) (interface{}, error)

type RetryStrategy string

const (
	FixedRetry       RetryStrategy = "fixed"
	ExponentialRetry RetryStrategy = "exponential"
	LinearRetry      RetryStrategy = "linear"
)

type SubscribeOptions struct {
	QueueOpts        *QueueConfig
	QueueName        string
	ConsumerName     string
	AutoAck          bool
	Exclusive        bool
	NoLocal          bool
	NoWait           bool
	Args             amqp.Table
	WorkerCount      int
	PrefetchCount    int
	MessageBuffer    int
	IsRPC            bool
	MaxRetryAttempts int           // Maximum number of retry attempts
	EnableDeadLetter bool          // Enable dead letter queue
	DeadLetterName   string        // Dead letter queue name
	RetryStrategy    RetryStrategy // Strategy for retry delays
	BaseRetryDelay   time.Duration // Base delay for retries (depends on strategy)
	MaxRetryDelay    time.Duration // Maximum delay for retries
}

func DefaultSubscribeOptions(queueName string, isRPC bool) *SubscribeOptions {
	opts := &SubscribeOptions{
		QueueOpts:        nil,
		QueueName:        queueName,
		ConsumerName:     queueName,
		AutoAck:          false,
		Exclusive:        false,
		NoLocal:          false,
		NoWait:           false,
		Args:             nil,
		WorkerCount:      3,
		PrefetchCount:    10,
		MessageBuffer:    100,
		IsRPC:            isRPC,
		MaxRetryAttempts: 5,
		EnableDeadLetter: true,
		DeadLetterName:   "fail:" + queueName,
		RetryStrategy:    FixedRetry,
		BaseRetryDelay:   time.Second * 5,
		MaxRetryDelay:    time.Minute * 10,
	}

	if isRPC {
		opts.WorkerCount = 5
		opts.PrefetchCount = 1
	}

	return opts
}

type Subscriber struct {
	connManager     *ConnectionManager
	channelManagers []*ChannelManager
	handler         MessageHandler
	opts            *SubscribeOptions
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	isRunning       atomic.Bool
	pool            *ants.Pool
	mu              sync.RWMutex
	msgChan         chan *amqp.Delivery
}

func NewSubscriber(ctx context.Context, connManager *ConnectionManager, handler MessageHandler, opts *SubscribeOptions) (*Subscriber, error) {
	ctx, cancel := context.WithCancel(ctx)

	poolOpts := ants.Options{
		ExpiryDuration: time.Hour,
		PreAlloc:       true,
		Nonblocking:    true,
		PanicHandler: func(i interface{}) {
			logger.Error.Printf("Worker panic: %v\n", i)
		},
	}

	pool, err := ants.NewPool(opts.WorkerCount, ants.WithOptions(poolOpts))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create example-rabbit-worker pool: %w", err)
	}

	sub := &Subscriber{
		connManager:     connManager,
		handler:         handler,
		opts:            opts,
		ctx:             ctx,
		cancel:          cancel,
		channelManagers: make([]*ChannelManager, opts.WorkerCount),
		pool:            pool,
		msgChan:         make(chan *amqp.Delivery, opts.MessageBuffer),
	}

	for i := 0; i < opts.WorkerCount; i++ {
		sub.channelManagers[i] = NewChannelManager(ctx, connManager)
	}

	return sub, nil
}

func (s *Subscriber) declareQueue(name string, workerID int, config *QueueConfig) (*amqp.Queue, error) {
	ch, err := s.channelManagers[workerID].GetChannel()
	queueName := name

	if err != nil || ch == nil {
		return nil, fmt.Errorf("failed to get channel: %w", err)
	}

	// Set consumer timeout to 25 minutes (less than RabbitMQ default 30 minutes)
	// This gives us a buffer to handle messages before RabbitMQ times out
	err = ch.Qos(
		s.opts.PrefetchCount,
		0,
		false,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to set QoS: %w", err)
	}

	if config == nil {
		config = DefaultQueueConfig()
	}

	if config.Args == nil {
		config.Args = make(amqp.Table)
	}

	reply, err := ch.QueueDeclare(
		queueName,
		config.Durable,
		config.AutoDelete,
		config.Exclusive,
		config.NoWait,
		config.Args,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return &reply, nil
}

func (s *Subscriber) Start() error {
	if s.isRunning.Swap(true) {
		return fmt.Errorf("subscriber is already running")
	}
	for i := 0; i < s.opts.WorkerCount; i++ {
		s.wg.Add(1)
		workerID := i
		go func(workerID int) {
			err := s.pool.Submit(func() {
				s.runWorker(workerID)
			})
			if err != nil {
				return
			}
		}(workerID)
	}

	return nil
}

func (s *Subscriber) runWorker(workerID int) {
	defer s.wg.Done()

	backoff := &exponentialBackoff{
		min:    1 * time.Second,
		max:    30 * time.Second,
		factor: 2,
	}

	consecutiveErrors := 0
	maxConsecutiveErrors := 10

	for s.isRunning.Load() {
		select {
		case <-s.ctx.Done():
			_ = s.Stop()
			return
		default:
			if err := s.consume(workerID); err != nil {
				consecutiveErrors++
				logger.Warning.Printf("Worker %d consume error (%d consecutive): %v\n", workerID, consecutiveErrors, err)

				// If too many consecutive errors, increase backoff significantly
				if consecutiveErrors >= maxConsecutiveErrors {
					logger.Error.Printf("Worker %d exceeded max consecutive errors, backing off significantly\n", workerID)
					time.Sleep(backoff.max)
				} else {
					backoff.sleep()
				}
				continue
			}
			consecutiveErrors = 0
			backoff.reset()
		}
	}
}

type exponentialBackoff struct {
	min    time.Duration
	max    time.Duration
	factor float64
	curr   time.Duration
}

func (b *exponentialBackoff) sleep() {
	if b.curr == 0 {
		b.curr = b.min
	} else {
		b.curr = time.Duration(float64(b.curr) * b.factor)
		if b.curr > b.max {
			b.curr = b.max
		}
	}
	time.Sleep(b.curr)
}

func (b *exponentialBackoff) reset() {
	b.curr = 0
}

func (s *Subscriber) consume(workerID int) error {
	ch, err := s.channelManagers[workerID].GetChannel()
	if err != nil || ch == nil {
		// Add delay when channel is not available to prevent CPU spike
		time.Sleep(time.Second)
		return fmt.Errorf("failed to get channel: %w", err)
	}

	// Use blocking pool to ensure messages are processed
	messagePool, err := ants.NewPool(s.opts.PrefetchCount, ants.WithOptions(ants.Options{
		ExpiryDuration: time.Hour,
		PreAlloc:       true,
		Nonblocking:    false, // Changed to blocking to prevent message loss
		PanicHandler: func(i interface{}) {
			logger.Error.Printf("Message processor panic: %v\n", i)
		},
	}))
	if err != nil {
		return fmt.Errorf("failed to create message processor pool: %w", err)
	}
	defer messagePool.Release()

	q, err := s.declareQueue(s.opts.QueueName, workerID, s.opts.QueueOpts)
	if err != nil {
		return fmt.Errorf("failed to declare queue: %w", err)
	}

	consumerName := fmt.Sprintf("%s-%d-%d", s.opts.ConsumerName, workerID, time.Now().Unix())
	msgs, err := ch.ConsumeWithContext(
		s.ctx,
		q.Name,
		consumerName,
		s.opts.AutoAck,
		s.opts.Exclusive,
		s.opts.NoLocal,
		s.opts.NoWait,
		s.opts.Args,
	)
	if err != nil {
		return fmt.Errorf("failed to start consuming %d: %w", workerID, err)
	}

	for msg := range msgs {
		select {
		case <-s.ctx.Done():
			logger.Error.Printf("Worker %d stopping\n", workerID)
			_ = s.Stop()
			return nil
		default:
			// Copy message to avoid closure issues
			msgCopy := msg

			// Submit to pool (blocking, will wait if pool is full)
			err := messagePool.Submit(func() {
				// Add timeout for message processing
				ctx, cancel := context.WithTimeout(s.ctx, 25*time.Minute) // Less than RabbitMQ timeout (30 min)
				defer cancel()

				done := make(chan error, 1)
				go func() {
					done <- s.processMessage(workerID, &msgCopy)
				}()

				select {
				case err := <-done:
					if err != nil {
						logger.Error.Printf("Worker %d failed to process message: %v\n", workerID, err)
						// Message already handled in processMessage (ack/nack/retry)
					}
				case <-ctx.Done():
					logger.Error.Printf("Worker %d message processing timeout (25min), nacking message\n", workerID)
					if !s.opts.AutoAck {
						// Nack without requeue - message took too long
						if nackErr := msgCopy.Nack(false, false); nackErr != nil {
							logger.Error.Printf("Failed to nack timed out message: %v\n", nackErr)
						}
					}
				}
			})

			if err != nil {
				logger.Error.Printf("Worker %d failed to submit to pool: %v\n", workerID, err)
			}
		}
	}

	return nil
}

func (s *Subscriber) processMessage(workerID int, msg *amqp.Delivery) error {
	deliveryCount := s.getDeliveryCount(msg)

	response, err := s.handler(msg)

	if err != nil {
		return s.handleProcessingError(workerID, msg, err, deliveryCount)
	}

	return s.handleSuccessfulProcessing(workerID, msg, response)
}

func (s *Subscriber) getDeliveryCount(msg *amqp.Delivery) int {
	deliveryCount := 0
	if msg.Headers != nil {
		if count, exists := msg.Headers["x-retry-count"]; exists {
			switch v := count.(type) {
			case int:
				deliveryCount = v
			case int32:
				deliveryCount = int(v)
			case int64:
				deliveryCount = int(v)
			default:
				logger.Warning.Printf("Unexpected type for x-retry-count: %T", v)
			}
		}
	}

	if msg.Redelivered && deliveryCount == 0 {
		deliveryCount = 1
	}

	return deliveryCount
}

func (s *Subscriber) handleProcessingError(workerID int, msg *amqp.Delivery, err error, deliveryCount int) error {
	// Check if this is an RPC message (has CorrelationId, regardless of ReplyTo)
	if msg.CorrelationId != "" {
		return s.handleRPCProcessingError(workerID, msg, err)
	}

	// For regular messages, apply retry logic
	if deliveryCount >= s.opts.MaxRetryAttempts {
		return s.handleMaxRetryExceeded(workerID, msg, err)
	}

	if retryErr := s.republishWithDelay(workerID, msg, deliveryCount+1); retryErr != nil {
		return fmt.Errorf("failed to republish message with delay: %w", retryErr)
	}

	return fmt.Errorf("handler error on attempt %d: %w", deliveryCount+1, err)
}

func (s *Subscriber) handleRPCProcessingError(workerID int, msg *amqp.Delivery, err error) error {
	// Only send RPC error response if ReplyTo is present
	if msg.ReplyTo != "" {
		if rpcErr := s.handleRPCError(workerID, msg, err); rpcErr != nil {
			logger.Error.Printf("Failed to handle RPC error: %v", rpcErr)
		}
	} else {
		logger.Warning.Printf("Worker %d: RPC message missing ReplyTo, cannot send error response\n", workerID)
	}

	if !s.opts.AutoAck {
		if ackErr := msg.Ack(false); ackErr != nil {
			return fmt.Errorf("failed to acknowledge message: %w", ackErr)
		}
	}

	return nil
}

func (s *Subscriber) handleMaxRetryExceeded(workerID int, msg *amqp.Delivery, err error) error {
	if s.opts.EnableDeadLetter {
		return s.moveToDeadLetter(workerID, msg, err)
	}

	if rejectErr := msg.Reject(false); rejectErr != nil {
		return fmt.Errorf("failed to reject message: %w", rejectErr)
	}

	return nil
}

func (s *Subscriber) moveToDeadLetter(workerID int, msg *amqp.Delivery, err error) error {
	if !s.opts.AutoAck {
		if ackErr := msg.Ack(false); ackErr != nil {
			return fmt.Errorf("failed to acknowledge message: %w", ackErr)
		}
	}

	if dlErr := s.publishToDeadLetter(workerID, msg, err); dlErr != nil {
		return fmt.Errorf("failed to publish to dead letter queue: %w", dlErr)
	}
	return nil
}

func (s *Subscriber) handleSuccessfulProcessing(workerID int, msg *amqp.Delivery, response interface{}) error {
	// Check if this is an RPC message (has both CorrelationId and ReplyTo)
	if msg.CorrelationId != "" && msg.ReplyTo != "" {
		if err := s.handleSuccessfulRPC(workerID, msg, response); err != nil {
			return fmt.Errorf("failed to handle successful RPC: %w", err)
		}
	} else if msg.CorrelationId != "" && msg.ReplyTo == "" {
		// Message has CorrelationId but no ReplyTo - likely a malformed RPC or regular message
		logger.Warning.Printf("Worker %d: message has CorrelationId but no ReplyTo, treating as regular message\n", workerID)
	}

	if !s.opts.AutoAck {
		if err := msg.Ack(false); err != nil {
			return fmt.Errorf("failed to acknowledge message: %w", err)
		}
	}

	return nil
}

func (s *Subscriber) republishWithDelay(workerID int, msg *amqp.Delivery, retryCount int) error {
	msg.Headers["x-retry-count"] = retryCount

	delay := s.calculateRetryDelay(retryCount)
	delayMs := int64(delay / time.Millisecond)

	publishing := amqp.Publishing{
		Headers:         msg.Headers,
		ContentType:     msg.ContentType,
		ContentEncoding: msg.ContentEncoding,
		DeliveryMode:    msg.DeliveryMode,
		Priority:        msg.Priority,
		CorrelationId:   msg.CorrelationId,
		ReplyTo:         msg.ReplyTo,
		Expiration:      msg.Expiration,
		MessageId:       msg.MessageId,
		Timestamp:       msg.Timestamp,
		Type:            msg.Type,
		UserId:          msg.UserId,
		AppId:           msg.AppId,
		Body:            msg.Body,
	}
	logger.Info.Printf("Scheduling retry with %dms delay using time-based retry", delayMs)

	if !s.opts.AutoAck {
		if ackErr := msg.Ack(false); ackErr != nil {
			return fmt.Errorf("failed to acknowledge original message: %w", ackErr)
		}
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		timer := time.NewTimer(delay)
		select {
		case <-timer.C:
		case <-s.ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		}

		ch, err := s.channelManagers[workerID].GetChannel()
		if err != nil {
			logger.Error.Printf("Failed to get channel after delay: %v", err)
			return
		}

		err = ch.PublishWithContext(
			s.ctx,
			"",
			s.opts.QueueName,
			true,
			false,
			publishing,
		)
		if err != nil {
			logger.Error.Printf("Failed to republish message after delay: %v", err)
		}
	}()

	return nil
}

func (s *Subscriber) publishToDeadLetter(workerID int, msg *amqp.Delivery, err error) error {
	ch, err2 := s.channelManagers[workerID].GetChannel()

	if s.opts.EnableDeadLetter {
		err2 = ch.ExchangeDeclare(
			s.opts.DeadLetterName,
			"direct",
			true,
			false,
			false,
			false,
			nil,
		)
		if err2 != nil {
			return fmt.Errorf("failed to declare dead letter exchange: %w", err2)
		}

		_, err2 = ch.QueueDeclare(
			s.opts.DeadLetterName,
			true,
			false,
			false,
			false,
			nil,
		)
		if err2 != nil {
			return fmt.Errorf("failed to declare dead letter queue: %w", err2)
		}

		err2 = ch.QueueBind(
			s.opts.DeadLetterName,
			s.opts.QueueName,
			s.opts.DeadLetterName,
			false,
			nil,
		)
		if err2 != nil {
			return fmt.Errorf("failed to bind dead letter queue: %w", err2)
		}
	}

	if err2 != nil {
		return fmt.Errorf("failed to get channel for dead letter: %w", err2)
	}

	msg.Headers["x-death-reason"] = err.Error()
	msg.Headers["x-death-time"] = time.Now().Format(time.RFC3339)
	msg.Headers["x-death-queue"] = s.opts.QueueName
	msg.Headers["x-death-max-retries"] = s.opts.MaxRetryAttempts

	retryCount := 0
	if count, exists := msg.Headers["x-retry-count"]; exists {
		switch v := count.(type) {
		case int:
			retryCount = v
		case int32:
			retryCount = int(v)
		case int64:
			retryCount = int(v)
		}
	}

	// Create publishing with all original message properties
	publishing := amqp.Publishing{
		Headers:         msg.Headers,
		ContentType:     msg.ContentType,
		ContentEncoding: msg.ContentEncoding,
		DeliveryMode:    msg.DeliveryMode,
		Priority:        msg.Priority,
		CorrelationId:   msg.CorrelationId,
		ReplyTo:         msg.ReplyTo,
		Expiration:      msg.Expiration,
		MessageId:       msg.MessageId,
		Timestamp:       msg.Timestamp,
		Type:            msg.Type,
		UserId:          msg.UserId,
		AppId:           msg.AppId,
		Body:            msg.Body,
	}

	err2 = ch.PublishWithContext(
		s.ctx,
		s.opts.DeadLetterName,
		s.opts.QueueName,
		true,
		false,
		publishing,
	)
	if err2 != nil {
		return fmt.Errorf("failed to publish to dead letter exchange: %w", err2)
	}

	logger.Info.Printf("Successfully published failed message to dead letter queue after %d retries", retryCount)
	return nil
}

func (s *Subscriber) handleRPCError(workerID int, msg *amqp.Delivery, handlerErr error) error {
	payload, err := NewMessage(&RPCResponse{
		Status: "error",
		Err:    handlerErr.Error(),
	}, &msg.Headers)
	if err != nil {
		return fmt.Errorf("failed to create error payload: %w", err)
	}
	if err := s.sendReply(workerID, msg, payload); err != nil {
		return fmt.Errorf("failed to send error reply: %w", err)
	}
	return nil
}

func (s *Subscriber) handleSuccessfulRPC(workerID int, msg *amqp.Delivery, response interface{}) error {
	payload, err := NewMessage(&RPCResponse{
		IsDisposed: true,
		Response:   response,
	}, &msg.Headers)
	if err != nil {
		return fmt.Errorf("failed to create response payload: %w", err)
	}
	if err := s.sendReply(workerID, msg, payload); err != nil {
		return fmt.Errorf("failed to send reply: %w", err)
	}
	return nil
}

func (s *Subscriber) sendReply(workerID int, delivery *amqp.Delivery, msg *Message) error {
	ch, err := s.channelManagers[workerID].GetChannel()

	if err != nil || ch == nil {
		return fmt.Errorf("channel not available")
	}

	payload := msg.GenerateRPCReplyPayload(delivery.CorrelationId)

	if delivery.ReplyTo == "" {
		return fmt.Errorf("reply_to property not found in delivery")
	}

	err = ch.PublishWithContext(
		s.ctx,
		"",
		delivery.ReplyTo,
		false,
		false,
		*payload)
	if err != nil {
		return fmt.Errorf("failed to send reply: %w", err)
	}
	return nil
}

func (s *Subscriber) calculateRetryDelay(retryCount int) time.Duration {
	var delay time.Duration

	switch s.opts.RetryStrategy {
	case FixedRetry:
		delay = s.opts.BaseRetryDelay
	case LinearRetry:
		delay = s.opts.BaseRetryDelay * time.Duration(retryCount)
	case ExponentialRetry:
		multiplier := 1
		for i := 0; i < retryCount; i++ {
			multiplier *= 2
		}
		delay = s.opts.BaseRetryDelay * time.Duration(multiplier)
	default:
		multiplier := 1
		for i := 0; i < retryCount; i++ {
			multiplier *= 2
		}
		delay = s.opts.BaseRetryDelay * time.Duration(multiplier)
	}

	if delay > s.opts.MaxRetryDelay {
		delay = s.opts.MaxRetryDelay
	}

	return delay
}

func (s *Subscriber) Stop() error {
	if !s.isRunning.Swap(false) {
		return nil
	}

	s.cancel()

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second * 60):
		return fmt.Errorf("timeout waiting for workers to stop")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i, ch := range s.channelManagers {
		if ch != nil {
			if err := ch.Close(); err != nil {
				logger.Error.Printf("Error closing channel for example-rabbit-worker %d: %v\n", i, err)
			}
			s.channelManagers[i] = nil
		}
	}

	close(s.msgChan)
	s.pool.Release()
	return nil
}

func (s *Subscriber) GetRunningWorkers() int {
	return s.pool.Running()
}

func (s *Subscriber) GetWorkerCapacity() int {
	return s.pool.Cap()
}

func (s *Subscriber) IsHealthy() bool {
	return s.isRunning.Load() && s.pool.Running() > 0
}

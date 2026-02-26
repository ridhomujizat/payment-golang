package serverApp

import (
	"context"
	"fmt"
	database "go-boilerplate/internal/pkg/db"
	"go-boilerplate/internal/pkg/logger"
	"go-boilerplate/internal/pkg/rabbitmq"
	"go-boilerplate/internal/pkg/redis"
	s3aws "go-boilerplate/internal/pkg/storage/s3"
	"time"

	"github.com/panjf2000/ants"
)

// InitWorker initializes background workers
// Add your worker initialization here following the example:
//
//	err = pool.Submit(func() {
//	    // Your worker logic here
//	    if err := myWorker.Subscribe(); err != nil {
//	        logger.Error.Printf("Failed to initialize worker: %v\n", err)
//	    }
//	})
func InitWorker(
	ctx context.Context,
	redisClient redis.IRedis,
	db *database.Database,
	rb *rabbitmq.ConnectionManager,
	publisher *rabbitmq.Publisher,
	s3 *s3aws.Is3,
) {
	poolOpts := ants.Options{
		ExpiryDuration: time.Hour,
		PreAlloc:       true,
		Nonblocking:    true,
		PanicHandler: func(i interface{}) {
			logger.Error.Printf("Worker panic: %v\n", i)
		},
	}

	pool, err := ants.NewPool(100, ants.WithOptions(poolOpts))
	if err != nil {
		panic(fmt.Errorf("failed to create worker pool: %w", err))
	}
	defer pool.Release()

	// TODO: Add your workers here
	// Example:
	// err = pool.Submit(func() {
	//     if err := myWorker.Subscribe(); err != nil {
	//         logger.Error.Printf("Failed to initialize worker: %v\n", err)
	//     }
	// })
	if err != nil {
		panic(fmt.Errorf("failed to submit task to pool: %w", err))
	}
}

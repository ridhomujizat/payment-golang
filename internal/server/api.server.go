package serverApp

import (
	"context"

	ai "go-boilerplate/internal/pkg/ai-connector"
	database "go-boilerplate/internal/pkg/db"
	"go-boilerplate/internal/pkg/middleware"
	"go-boilerplate/internal/pkg/rabbitmq"
	"go-boilerplate/internal/pkg/redis"
	s3aws "go-boilerplate/internal/pkg/storage/s3"
	"go-boilerplate/internal/repository"
	"sync"

	xampleHandler "go-boilerplate/internal/handler/example"
	xampleService "go-boilerplate/internal/service/example"

	"github.com/gin-gonic/gin"
)

// Setup initializes the HTTP server with middleware and routes
func Setup(
	engine *gin.Engine,
	ctx context.Context,
	wg *sync.WaitGroup,
	db *database.Database,
	redisClient redis.IRedis,
	rb *rabbitmq.ConnectionManager,
	publisher *rabbitmq.Publisher,
	s3 *s3aws.Is3,
	ai *ai.AiClient,
) {
	InitMiddleware(engine, publisher)

	// Health check endpoint
	engine.GET("/health", func(c *gin.Context) {
		rabbitmqHealth := "unhealthy"
		redisHealth := "unhealthy"
		databaseHealth := "unhealthy"
		rbCon := rb.GetConnection()

		if db != nil && !db.IsCloseConnection() {
			databaseHealth = "healthy"
		}

		if rbCon != nil && !rbCon.IsClosed() {
			rabbitmqHealth = "healthy"
		}
		if redisClient != nil && redisClient.Close() == nil {
			redisHealth = "healthy"
		}
		c.JSON(200, gin.H{
			"status": 200,
			"service": gin.H{
				"rabbitmq": gin.H{
					"status": rabbitmqHealth,
				},
				"redis": gin.H{
					"status": redisHealth,
				},
				"database": gin.H{
					"status": databaseHealth,
				},
			},
		})
	})

	e := engine.Group(BasePath())
	InitRoutes(e, ctx, wg, db, redisClient, rb, publisher, s3, ai)
}

// BasePath returns the base API path
func BasePath() string {
	return "/api"
}

// InitMiddleware initializes global middleware
func InitMiddleware(e *gin.Engine, publisher *rabbitmq.Publisher) {
	e.Use(middleware.CorsMiddleware())
	e.Use(middleware.RequestInit())
	e.Use(middleware.ResponseInit())
}

func InitRoutes(
	e *gin.RouterGroup,
	ctx context.Context,
	wg *sync.WaitGroup,
	db *database.Database,
	redisClient redis.IRedis,
	rb *rabbitmq.ConnectionManager,
	publisher *rabbitmq.Publisher,
	s3 *s3aws.Is3,
	ai *ai.AiClient,
) {

	// setup repo
	rp := repository.IRepository{}
	// setup service
	XampleService := xampleService.NewService(ctx, redisClient, rb, publisher, rp)
	// setup handler
	XampleHandler := xampleHandler.NewHandler(ctx, rb, XampleService)
	// init route
	XampleHandler.NewRoutes(e)

}

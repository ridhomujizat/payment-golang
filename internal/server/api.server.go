package serverApp

import (
	"context"
	"crypto/rsa"
	"net/url"
	"strings"

	ai "go-boilerplate/internal/pkg/ai-connector"
	database "go-boilerplate/internal/pkg/db"
	"go-boilerplate/internal/pkg/logger"
	"go-boilerplate/internal/pkg/middleware"
	midtransPkg "go-boilerplate/internal/pkg/midtrans"
	"go-boilerplate/internal/pkg/rabbitmq"
	"go-boilerplate/internal/pkg/redis"
	s3aws "go-boilerplate/internal/pkg/storage/s3"
	"go-boilerplate/internal/pkg/waflow"
	"go-boilerplate/internal/repository"
	paymentRepo "go-boilerplate/internal/repository/payment"
	"sync"

	xampleHandler "go-boilerplate/internal/handler/example"
	paymentHandler "go-boilerplate/internal/handler/payment"
	xampleService "go-boilerplate/internal/service/example"
	paymentService "go-boilerplate/internal/service/payment"

	"go-boilerplate/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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
	mt *midtransPkg.MidtransClient,
	baseURL string,
	waPrivateKeyPath string,
) {
	InitMiddleware(engine, publisher)

	// Set swagger host dynamically from APP_BASE_URL
	if parsed, err := url.Parse(baseURL); err == nil {
		docs.SwaggerInfo.Host = parsed.Host
		if strings.HasPrefix(baseURL, "https") {
			docs.SwaggerInfo.Schemes = []string{"https"}
		} else {
			docs.SwaggerInfo.Schemes = []string{"http"}
		}
	}

	// Swagger UI
	engine.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Load HTML templates for payment pages
	engine.LoadHTMLGlob("frontend/templates/*")

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
	InitRoutes(e, engine, ctx, wg, db, redisClient, rb, publisher, s3, ai, mt, baseURL, waPrivateKeyPath)
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
	engine *gin.Engine,
	ctx context.Context,
	wg *sync.WaitGroup,
	db *database.Database,
	redisClient redis.IRedis,
	rb *rabbitmq.ConnectionManager,
	publisher *rabbitmq.Publisher,
	s3 *s3aws.Is3,
	ai *ai.AiClient,
	mt *midtransPkg.MidtransClient,
	baseURL string,
	waPrivateKeyPath string,
) {

	// setup repo
	rp := repository.IRepository{
		Payment: paymentRepo.NewRepo(db),
	}

	// === Example ===
	XampleService := xampleService.NewService(ctx, redisClient, rb, publisher, rp)
	XampleHandler := xampleHandler.NewHandler(ctx, rb, XampleService)
	XampleHandler.NewRoutes(e)

	// === Load WA Flows private key (optional) ===
	var waPrivateKey *rsa.PrivateKey
	if waPrivateKeyPath != "" {
		var err error
		waPrivateKey, err = waflow.LoadPrivateKey(waPrivateKeyPath)
		if err != nil {
			logger.Error.Printf("Failed to load WA Flows private key: %v", err)
		} else {
			logger.Info.Printf("WA Flows private key loaded from %s", waPrivateKeyPath)
		}
	}

	// === Payment ===
	PaymentService := paymentService.NewService(ctx, rp, mt, baseURL)
	PaymentHandler := paymentHandler.NewHandler(ctx, PaymentService, mt, baseURL, waPrivateKey)
	PaymentHandler.NewRoutes(e)
	PaymentHandler.NewPageRoutes(engine)
}

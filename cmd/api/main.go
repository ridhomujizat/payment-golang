package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	config "go-boilerplate/configs"
	ai "go-boilerplate/internal/pkg/ai-connector"
	database "go-boilerplate/internal/pkg/db"
	"go-boilerplate/internal/pkg/helper"
	"go-boilerplate/internal/pkg/logger"
	midtransPkg "go-boilerplate/internal/pkg/midtrans"
	"go-boilerplate/internal/pkg/rabbitmq"
	"go-boilerplate/internal/pkg/redis"
	"go-boilerplate/internal/pkg/validation"
	serverApp "go-boilerplate/internal/server"
	"sync"
	"syscall"

	"github.com/gin-gonic/gin"
)

// @title           Go Boilerplate API
// @version         1.0
// @description     A production-ready Golang boilerplate API with common features

// @contact.name    API Support
// @contact.url     http://www.swagger.io/support
// @contact.email   support@swagger.io

// @license.name    Apache 2.0
// @license.url     http://www.apache.org/licenses/LICENSE-2.0.html

// @BasePath        /api
func main() {
	logger.Setup()

	env, err := config.GetEnv()
	if err != nil {
		logger.Error.Println("Error getting environment", err)
		panic(err)
	}

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	// Setup Redis
	redisClient, err := setupRedis(ctx, env)
	if err != nil {
		logger.Error.Println("Error setting up Redis", err)
		cancel()
		return
	}

	// Setup RabbitMQ
	rabbit, err := setupRabbitMQ(ctx, env)
	if err != nil {
		logger.Error.Println("Error setting up RabbitMQ", err)
		cancel()
		return
	}

	// Setup Database
	db, err := setupDB(env)
	if err != nil {
		logger.Error.Println("Error setting up Database", err)
		cancel()
		return
	}

	// Setup AI Client (optional)
	aiClient := setupAI(ctx)

	// Setup Midtrans Client
	mtClient := setupMidtrans(env)

	// Setup Server
	setupServer(&config.SetupServerDto{
		Rds:    redisClient,
		Env:    env,
		Ctx:    &ctx,
		Cancel: cancel,
		Db:     db,
		Wg:     &wg,
		Rb:     rabbit,
		Ai:     aiClient,
		Mt:     mtClient,
	})
}

func setupRedis(ctx context.Context, env *config.Config) (redis.IRedis, error) {
	return redis.Setup(ctx, &redis.Config{
		Host:     env.RedisHost,
		Username: env.RedisUser,
		Port:     env.RedisPort,
		Password: env.RedisPass,
		PoolSize: env.RedisPoolSize,
	})
}

func setupRabbitMQ(ctx context.Context, env *config.Config) (*rabbitmq.ConnectionManager, error) {
	return rabbitmq.NewConnectionManager(ctx, &rabbitmq.Config{
		Username: env.RabbitUser,
		Password: env.RabbitPass,
		Host:     env.RabbitHost,
		Port:     env.RabbitPort,
	})
}

func setupDB(env *config.Config) (*database.Database, error) {
	return database.Setup(&database.Config{
		Host:     env.DBHost,
		Port:     env.DBPort,
		User:     env.DBUser,
		Password: env.DBPass,
		Database: env.DBName,
		SSLMode:  "disable",
		Driver:   "postgres",
	})
}

func setupAI(ctx context.Context) *ai.AiClient {
	apiKey := helper.GetEnv("GEMINI_API_KEY")
	model := helper.GetEnv("GEMINI_MODEL", "gemini-pro")

	fmt.Printf("Gemini API Key configured: %t\n", apiKey != "")
	fmt.Printf("Gemini Model: %s\n", model)

	return ai.NewAiClient(
		ctx,
		&ai.Config{
			GeminiAPIKey: apiKey,
			GeminiModel:  model,
		},
	)
}

func setupMidtrans(env *config.Config) *midtransPkg.MidtransClient {
	return midtransPkg.Setup(&midtransPkg.Config{
		ServerKey:   env.MidtransServerKey,
		ClientKey:   env.MidtransClientKey,
		Environment: env.MidtransEnvironment,
	})
}

func setupServer(payload *config.SetupServerDto) {
	rds := payload.Rds
	env := payload.Env
	ctx := payload.Ctx
	cancel := payload.Cancel
	wg := payload.Wg
	rb := payload.Rb
	db := payload.Db
	s3 := payload.S3
	ai := payload.Ai
	mt := payload.Mt

	defer func() {
		if rds != nil {
			_ = rds.Close()
		}
		cancel()
		wg.Wait()
	}()

	err := validation.Setup()
	if err != nil {
		logger.Error.Println("Failed to setup validation")
		panic(err)
	}

	e := gin.Default()

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", env.AppPort),
		Handler: e,
	}

	publisher, err := rabbitmq.NewPublisher(*ctx, rb)
	if err != nil {
		panic(err)
	}

	serverApp.Setup(e, *ctx, wg, db, rds, rb, publisher, s3, ai, mt, env.AppBaseURL, env.WAPrivateKeyPath)
	if payload.Env.AppEnv != "development" {
		serverApp.InitWorker(*ctx, rds, db, rb, publisher, s3)
	}

	go func() {
		logger.HTTP.Println("========= Server Started =========")
		logger.HTTP.Println("=========", env.AppPort, "=========")
		if err := server.ListenAndServe(); err != nil {
			logger.Error.Println("Server error:", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-sigChan
	logger.HTTP.Println("========= Server Shutting Down =========")
	_ = server.Shutdown(*ctx)
}

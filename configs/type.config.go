package config

import (
	"context"
	"go-boilerplate/internal/common/enum"
	"go-boilerplate/internal/pkg/ai-connector"
	database "go-boilerplate/internal/pkg/db"
	"go-boilerplate/internal/pkg/rabbitmq"
	"go-boilerplate/internal/pkg/redis"
	s3aws "go-boilerplate/internal/pkg/storage/s3"
	"sync"
)

// Config holds all application configuration loaded from environment variables
type Config struct {
	AppEnv        enum.EnvEnum `env:"APP_ENV" envDefault:"development"`
	AppPort       int          `env:"APP_PORT" envDefault:"8080"`
	RedisHost     string       `env:"REDIS_HOST" envDefault:"localhost"`
	RedisPort     int          `env:"REDIS_PORT" envDefault:"6379"`
	RedisUser     string       `env:"REDIS_USER" envDefault:"default"`
	RedisPass     string       `env:"REDIS_PASS" envDefault:""`
	RedisPoolSize int          `env:"REDIS_POOL_SIZE" envDefault:"10"`
	RabbitHost    string       `env:"RABBIT_HOST" envDefault:"localhost"`
	RabbitPort    int          `env:"RABBIT_PORT" envDefault:"5672"`
	RabbitUser    string       `env:"RABBIT_USER" envDefault:"guest"`
	RabbitPass    string       `env:"RABBIT_PASS" envDefault:"guest"`
	DBHost        string       `env:"DB_HOST" envDefault:"localhost"`
	DBPort        int          `env:"DB_PORT" envDefault:"5432"`
	DBUser        string       `env:"DB_USER" envDefault:"postgres"`
	DBPass        string       `env:"DB_PASS" envDefault:""`
	DBName        string       `env:"DB_NAME" envDefault:"postgres"`
	GeminiAPIKey  string       `env:"GEMINI_API_KEY" envDefault:""`
	GeminiModel   string       `env:"GEMINI_MODEL" envDefault:"gemini-2.0-flash-exp"`
	// AWS S3 Configuration (optional, uncomment if needed)
	// AWSACCESSKEYID     string       `env:"AWS_ACCESS_KEY_ID" envDefault:""`
	// AWSSECRETACCESSKEY string       `env:"AWS_SECRET_ACCESS_KEY" envDefault:""`
	// AWSREGION          string       `env:"AWS_REGION" envDefault:"us-east-1"`
	// AWSBUCKETNAME      string       `env:"AWS_BUCKET_NAME" envDefault:""`
}

// SetupServerDto contains dependencies for server setup
type SetupServerDto struct {
	Ctx    *context.Context
	Cancel context.CancelFunc
	Wg     *sync.WaitGroup
	Env    *Config
	Db     *database.Database
	Rds    redis.IRedis
	Rb     *rabbitmq.ConnectionManager
	S3     *s3aws.Is3
	Ai     *ai.AiClient
}

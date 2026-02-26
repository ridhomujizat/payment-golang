package redis

import (
	"context"
	"time"

	_redis "github.com/redis/go-redis/v9"
)

type Config struct {
	Host     string
	Username string
	Port     int
	Password string
	PoolSize int
}

type Client struct {
	Client *_redis.Client
	config *Config
	cancel context.CancelFunc
	ctx    context.Context
}

type IRedis interface {
	Close() error
	Set(key string, value interface{}, expiration time.Duration) error
	Get(key string) (string, error)
	Del(key string) error
	Expire(key string, expiration time.Duration) error
}

type ClientType = _redis.Client

const NilType = _redis.Nil

package activitylog

import (
	"context"
	types "go-boilerplate/internal/common/type"
	"go-boilerplate/internal/pkg/rabbitmq"
	"go-boilerplate/internal/pkg/redis"
	"go-boilerplate/internal/repository"
)

type Service struct {
	ctx       context.Context
	redis     redis.IRedis
	rp        repository.IRepository
	rabbitmq  *rabbitmq.ConnectionManager
	publisher *rabbitmq.Publisher
}

type IService interface {
	XampleService() *types.Response
}

func NewService(ctx context.Context, redis redis.IRedis, rabbitmq *rabbitmq.ConnectionManager, publisher *rabbitmq.Publisher, repository repository.IRepository) IService {
	return &Service{
		ctx:       ctx,
		redis:     redis,
		rp:        repository,
		rabbitmq:  rabbitmq,
		publisher: publisher,
	}
}

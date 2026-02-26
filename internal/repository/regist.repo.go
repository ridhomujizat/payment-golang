package repository

import (
	paymentRepo "go-boilerplate/internal/repository/payment"
)

// IRepository is a container for all repository interfaces
type IRepository struct {
	Payment paymentRepo.IRepository
}

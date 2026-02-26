package payment

import (
	"context"
	"go-boilerplate/internal/common/models"
	database "go-boilerplate/internal/pkg/db"
)

type IRepository interface {
	Create(ctx context.Context, trx *models.Transaction) error
	FindByOrderID(ctx context.Context, orderID string) (*models.Transaction, error)
	FindBySnapToken(ctx context.Context, snapToken string) (*models.Transaction, error)
	UpdateStatus(ctx context.Context, orderID string, updates map[string]any) error
}

type Repository struct {
	db *database.Database
}

func NewRepo(db *database.Database) IRepository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, trx *models.Transaction) error {
	return r.db.WithContext(ctx).Create(trx).Error
}

func (r *Repository) FindByOrderID(ctx context.Context, orderID string) (*models.Transaction, error) {
	var trx models.Transaction
	err := r.db.WithContext(ctx).Where("order_id = ?", orderID).First(&trx).Error
	if err != nil {
		return nil, err
	}
	return &trx, nil
}

func (r *Repository) FindBySnapToken(ctx context.Context, snapToken string) (*models.Transaction, error) {
	var trx models.Transaction
	err := r.db.WithContext(ctx).Where("snap_token = ?", snapToken).First(&trx).Error
	if err != nil {
		return nil, err
	}
	return &trx, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, orderID string, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&models.Transaction{}).Where("order_id = ?", orderID).Updates(updates).Error
}

package payment

import (
	"context"
	"encoding/json"
	types "go-boilerplate/internal/common/type"
	midtransPkg "go-boilerplate/internal/pkg/midtrans"
	"go-boilerplate/internal/repository"
)

type Service struct {
	ctx      context.Context
	rp       repository.IRepository
	midtrans *midtransPkg.MidtransClient
	baseURL  string
}

type IService interface {
	CreatePayment(req *CreatePaymentRequest) *types.Response
	CheckPaymentStatus(orderID string) *types.Response
	HandlePayment(req *PaymentResultRequest) *types.Response
	MidtransCallback(payload map[string]any) *types.Response
	GetTransactionByToken(snapToken string) *types.Response
}

func NewService(ctx context.Context, rp repository.IRepository, midtrans *midtransPkg.MidtransClient, baseURL string) IService {
	return &Service{
		ctx:      ctx,
		rp:       rp,
		midtrans: midtrans,
		baseURL:  baseURL,
	}
}

// Request/Response DTOs

type CustomerInfo struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Email string `json:"email"`
}

type ItemDetail struct {
	ID    string `json:"id" binding:"required"`
	Name  string `json:"name" binding:"required"`
	Price int64  `json:"price" binding:"required"`
	Qty   int    `json:"qty" binding:"required"`
}

type CreatePaymentRequest struct {
	OrderID  string         `json:"order_id" binding:"required"`
	Customer CustomerInfo   `json:"customer" binding:"required"`
	Items    []ItemDetail   `json:"items" binding:"required,min=1"`
	Metadata map[string]any `json:"metadata"`
}

type CreatePaymentResponse struct {
	OrderID    string `json:"order_id"`
	PaymentURL string `json:"payment_url"`
	SnapToken  string `json:"snap_token"`
	SnapURL    string `json:"snap_url"`
	Amount     int64  `json:"amount"`
}

type PaymentStatusResponse struct {
	OrderID       string `json:"order_id"`
	Status        string `json:"status"`
	PaymentType   string `json:"payment_type"`
	Amount        int64  `json:"amount"`
	TransactionID string `json:"transaction_id"`
}

type PaymentResultRequest struct {
	OrderID           string `json:"order_id"`
	TransactionID     string `json:"transaction_id"`
	TransactionStatus string `json:"transaction_status"`
	PaymentType       string `json:"payment_type"`
	GrossAmount       string `json:"gross_amount"`
	StatusCode        string `json:"status_code"`
	SignatureKey      string `json:"signature_key"`
}

type PaymentPageData struct {
	SnapToken     string `json:"snap_token"`
	OrderID       string `json:"order_id"`
	CustomerName  string `json:"customer_name"`
	CustomerPhone string `json:"customer_phone"`
	GrossAmount   int64  `json:"gross_amount"`
	Items         []ItemDetail `json:"items"`
}

func itemsToJSON(items []ItemDetail) json.RawMessage {
	b, _ := json.Marshal(items)
	return b
}

func metadataToJSON(metadata map[string]any) json.RawMessage {
	if metadata == nil {
		return json.RawMessage("null")
	}
	b, _ := json.Marshal(metadata)
	return b
}

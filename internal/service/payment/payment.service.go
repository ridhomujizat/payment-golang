package payment

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"go-boilerplate/internal/common/models"
	types "go-boilerplate/internal/common/type"
	"go-boilerplate/internal/pkg/helper"
	"go-boilerplate/internal/pkg/logger"
	"net/http"
	"time"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
	"github.com/midtrans/midtrans-go/snap"
)

func (s *Service) CreatePayment(req *CreatePaymentRequest) *types.Response {
	// Calculate gross amount from items
	var grossAmount int64
	var midtransItems []midtrans.ItemDetails
	for _, item := range req.Items {
		grossAmount += item.Price * int64(item.Qty)
		midtransItems = append(midtransItems, midtrans.ItemDetails{
			ID:    item.ID,
			Name:  item.Name,
			Price: item.Price,
			Qty:   int32(item.Qty),
		})
	}

	// Create Snap Request
	snapReq := &snap.Request{
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  req.OrderID,
			GrossAmt: grossAmount,
		},
		CustomerDetail: &midtrans.CustomerDetails{
			FName: req.Customer.Name,
			Email: req.Customer.Email,
			Phone: req.Customer.Phone,
		},
		Items: &midtransItems,
	}

	// Request to Midtrans Snap API
	snapResp, midErr := s.midtrans.Snap.CreateTransaction(snapReq)
	if midErr != nil {
		logger.Error.Printf("Failed to create Midtrans transaction: %v", midErr.GetMessage())
		return helper.ParseResponse(&types.Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create payment",
			Error:   fmt.Errorf("midtrans error: %s", midErr.GetMessage()),
		})
	}

	// Save to database
	trx := &models.Transaction{
		OrderID:       req.OrderID,
		CustomerName:  req.Customer.Name,
		CustomerPhone: req.Customer.Phone,
		CustomerEmail: req.Customer.Email,
		GrossAmount:   grossAmount,
		Items:         models.JSONB(itemsToJSON(req.Items)),
		Metadata:      models.JSONB(metadataToJSON(req.Metadata)),
		SnapToken:     snapResp.Token,
		SnapURL:       snapResp.RedirectURL,
		Status:        "pending",
	}

	if err := s.rp.Payment.Create(s.ctx, trx); err != nil {
		logger.Error.Printf("Failed to save transaction: %v", err)
		return helper.ParseResponse(&types.Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to save transaction",
			Error:   err,
		})
	}

	return helper.ParseResponse(&types.Response{
		Code:    http.StatusCreated,
		Message: "Payment created successfully",
		Data: CreatePaymentResponse{
			OrderID:    req.OrderID,
			PaymentURL: fmt.Sprintf("%s/pay/%s", s.baseURL, snapResp.Token),
			SnapToken:  snapResp.Token,
			SnapURL:    snapResp.RedirectURL,
			Amount:     grossAmount,
		},
	})
}

func (s *Service) CheckPaymentStatus(orderID string) *types.Response {
	// Check from Midtrans directly (real-time)
	transactionStatusResp, midErr := s.midtrans.CoreAPI.CheckTransaction(orderID)
	if midErr != nil {
		// Fallback to database
		trx, err := s.rp.Payment.FindByOrderID(s.ctx, orderID)
		if err != nil {
			return helper.ParseResponse(&types.Response{
				Code:    http.StatusNotFound,
				Message: "Transaction not found",
				Error:   err,
			})
		}
		return helper.ParseResponse(&types.Response{
			Code: http.StatusOK,
			Data: PaymentStatusResponse{
				OrderID:     trx.OrderID,
				Status:      trx.Status,
				Amount:      trx.GrossAmount,
				PaymentType: trx.PaymentType,
			},
		})
	}

	// Update database if status changed
	s.updateTransactionStatus(orderID, transactionStatusResp)

	// Get amount from DB (Midtrans returns string)
	trx, _ := s.rp.Payment.FindByOrderID(s.ctx, orderID)
	var amount int64
	if trx != nil {
		amount = trx.GrossAmount
	}

	return helper.ParseResponse(&types.Response{
		Code: http.StatusOK,
		Data: PaymentStatusResponse{
			OrderID:       orderID,
			Status:        transactionStatusResp.TransactionStatus,
			PaymentType:   transactionStatusResp.PaymentType,
			Amount:        amount,
			TransactionID: transactionStatusResp.TransactionID,
		},
	})
}

func (s *Service) HandlePayment(req *PaymentResultRequest) *types.Response {
	// Data from frontend CANNOT be trusted — always verify with Midtrans API
	transactionStatusResp, midErr := s.midtrans.CoreAPI.CheckTransaction(req.OrderID)
	if midErr != nil {
		return helper.ParseResponse(&types.Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to verify payment",
			Error:   fmt.Errorf("midtrans check error: %s", midErr.GetMessage()),
		})
	}

	s.updateTransactionStatus(req.OrderID, transactionStatusResp)

	trx, _ := s.rp.Payment.FindByOrderID(s.ctx, req.OrderID)
	var amount int64
	if trx != nil {
		amount = trx.GrossAmount
	}

	return helper.ParseResponse(&types.Response{
		Code:    http.StatusOK,
		Message: "Payment processed",
		Data: PaymentStatusResponse{
			OrderID:       req.OrderID,
			Status:        transactionStatusResp.TransactionStatus,
			PaymentType:   transactionStatusResp.PaymentType,
			Amount:        amount,
			TransactionID: transactionStatusResp.TransactionID,
		},
	})
}

func (s *Service) MidtransCallback(payload map[string]any) *types.Response {
	orderID, ok := payload["order_id"].(string)
	if !ok || orderID == "" {
		return helper.ParseResponse(&types.Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid notification payload: missing order_id",
		})
	}

	// Verify with Midtrans API (mandatory)
	transactionStatusResp, midErr := s.midtrans.CoreAPI.CheckTransaction(orderID)
	if midErr != nil {
		logger.Error.Printf("Failed to verify callback for order %s: %s", orderID, midErr.GetMessage())
		return helper.ParseResponse(&types.Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to verify notification",
			Error:   fmt.Errorf("midtrans check error: %s", midErr.GetMessage()),
		})
	}

	// Verify signature key
	if signatureKey, exists := payload["signature_key"].(string); exists {
		serverKey := s.midtrans.Snap.ServerKey
		statusCode, _ := payload["status_code"].(string)
		grossAmount, _ := payload["gross_amount"].(string)
		if !verifySignatureKey(orderID, statusCode, grossAmount, serverKey, signatureKey) {
			logger.Error.Printf("Invalid signature key for order %s", orderID)
			return helper.ParseResponse(&types.Response{
				Code:    http.StatusForbidden,
				Message: "Invalid signature key",
			})
		}
	}

	s.updateTransactionStatus(orderID, transactionStatusResp)

	logger.Info.Printf("Callback processed for order %s: status=%s", orderID, transactionStatusResp.TransactionStatus)

	return helper.ParseResponse(&types.Response{
		Code:    http.StatusOK,
		Message: "ok",
	})
}

func (s *Service) GetTransactionByToken(snapToken string) *types.Response {
	trx, err := s.rp.Payment.FindBySnapToken(s.ctx, snapToken)
	if err != nil {
		return helper.ParseResponse(&types.Response{
			Code:    http.StatusNotFound,
			Message: "Transaction not found",
			Error:   err,
		})
	}

	var items []ItemDetail
	_ = json.Unmarshal(trx.Items, &items)

	return helper.ParseResponse(&types.Response{
		Code: http.StatusOK,
		Data: PaymentPageData{
			SnapToken:     trx.SnapToken,
			OrderID:       trx.OrderID,
			CustomerName:  trx.CustomerName,
			CustomerPhone: trx.CustomerPhone,
			GrossAmount:   trx.GrossAmount,
			Items:         items,
		},
	})
}

func (s *Service) updateTransactionStatus(orderID string, resp *coreapi.TransactionStatusResponse) {
	if resp == nil {
		return
	}

	updates := map[string]any{
		"status":         resp.TransactionStatus,
		"payment_type":   resp.PaymentType,
		"transaction_id": resp.TransactionID,
		"fraud_status":   resp.FraudStatus,
		"status_code":    resp.StatusCode,
		"signature_key":  resp.SignatureKey,
	}

	if resp.TransactionStatus == "settlement" || resp.TransactionStatus == "capture" {
		now := time.Now()
		updates["paid_at"] = &now
	}

	if err := s.rp.Payment.UpdateStatus(s.ctx, orderID, updates); err != nil {
		logger.Error.Printf("Failed to update transaction status for order %s: %v", orderID, err)
	}
}

func verifySignatureKey(orderID, statusCode, grossAmount, serverKey, signatureKey string) bool {
	input := orderID + statusCode + grossAmount + serverKey
	hash := sha512.Sum512([]byte(input))
	return hex.EncodeToString(hash[:]) == signatureKey
}

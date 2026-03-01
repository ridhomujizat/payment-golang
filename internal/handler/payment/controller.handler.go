package payment

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	types "go-boilerplate/internal/common/type"
	"go-boilerplate/internal/pkg/helper"
	"go-boilerplate/internal/pkg/logger"
	midtransPkg "go-boilerplate/internal/pkg/midtrans"
	"go-boilerplate/internal/pkg/waflow"
	paymentService "go-boilerplate/internal/service/payment"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	ctx            context.Context
	paymentService paymentService.IService
	midtrans       *midtransPkg.MidtransClient
	baseURL        string
	waPrivateKey   *rsa.PrivateKey
}

type IHandler interface {
	NewRoutes(e *gin.RouterGroup)
	NewPageRoutes(e *gin.Engine)
}

func NewHandler(ctx context.Context, paymentService paymentService.IService, midtrans *midtransPkg.MidtransClient, baseURL string, waPrivateKey *rsa.PrivateKey) IHandler {
	return &Handler{
		ctx:            ctx,
		paymentService: paymentService,
		midtrans:       midtrans,
		baseURL:        baseURL,
		waPrivateKey:   waPrivateKey,
	}
}

// CreatePayment godoc
// @Summary      Create a new payment
// @Description  Receives order data from WhatsApp Bot, generates a Midtrans Snap transaction, saves to DB, and returns a payment URL
// @Tags         Payments
// @Accept       json
// @Produce      json
// @Param        request  body      paymentService.CreatePaymentRequest  true  "Payment creation request"
// @Success      201      {object}  types.ResponseAPI{data=paymentService.CreatePaymentResponse}
// @Failure      400      {object}  types.ResponseAPI
// @Failure      500      {object}  types.ResponseAPI
// @Router       /v1/payments/create [post]
func (h *Handler) CreatePayment(c *gin.Context) {
	send := c.MustGet("send").(func(r *types.Response))

	var req paymentService.CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		send(helper.ParseResponse(&types.Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request body",
			Error:   err,
		}))
		return
	}

	send(h.paymentService.CreatePayment(&req))
}

// CheckStatus godoc
// @Summary      Check payment status
// @Description  Checks real-time payment status from Midtrans API with database fallback
// @Tags         Payments
// @Accept       json
// @Produce      json
// @Param        order_id  path      string  true  "Order ID"
// @Success      200       {object}  types.ResponseAPI{data=paymentService.PaymentStatusResponse}
// @Failure      400       {object}  types.ResponseAPI
// @Failure      404       {object}  types.ResponseAPI
// @Router       /v1/payments/status/{order_id} [get]
func (h *Handler) CheckStatus(c *gin.Context) {
	send := c.MustGet("send").(func(r *types.Response))

	orderID := c.Param("order_id")
	if orderID == "" {
		send(helper.ParseResponse(&types.Response{
			Code:    http.StatusBadRequest,
			Message: "order_id is required",
		}))
		return
	}

	send(h.paymentService.CheckPaymentStatus(orderID))
}

// HandlePaymentResult godoc
// @Summary      Process payment result from frontend
// @Description  Receives snap.js callback result from frontend, verifies with Midtrans API, and updates transaction status
// @Tags         Payments
// @Accept       json
// @Produce      json
// @Param        request  body      paymentService.PaymentResultRequest  true  "Payment result from snap.js"
// @Success      200      {object}  types.ResponseAPI{data=paymentService.PaymentStatusResponse}
// @Failure      400      {object}  types.ResponseAPI
// @Failure      500      {object}  types.ResponseAPI
// @Router       /v1/payments/process [post]
func (h *Handler) HandlePaymentResult(c *gin.Context) {
	send := c.MustGet("send").(func(r *types.Response))

	var req paymentService.PaymentResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		send(helper.ParseResponse(&types.Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request body",
			Error:   err,
		}))
		return
	}

	send(h.paymentService.HandlePayment(&req))
}

// MidtransCallback godoc
// @Summary      Midtrans payment notification webhook
// @Description  Receives HTTP POST notification from Midtrans when transaction status changes. This URL must be registered in Midtrans Dashboard > Settings > Payment Notification URL.
// @Tags         Payments
// @Accept       json
// @Produce      json
// @Param        request  body      map[string]interface{}  true  "Midtrans notification payload"
// @Success      200      {object}  map[string]string
// @Failure      400      {object}  map[string]string
// @Failure      403      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /v1/payments/callback [post]
func (h *Handler) MidtransCallback(c *gin.Context) {
	var payload map[string]any
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "error", "message": "invalid payload"})
		return
	}

	result := h.paymentService.MidtransCallback(payload)
	c.JSON(result.Code, gin.H{"status": "ok"})
}

// WAFlowEndpoint godoc
// @Summary      WhatsApp Flow encrypted endpoint
// @Description  Receives encrypted request from WhatsApp Flows, decrypts, processes action, and returns encrypted response
// @Tags         WhatsApp Flow
// @Accept       json
// @Produce      plain
// @Param        request  body      waflow.EncryptedRequest  true  "Encrypted WhatsApp Flow request"
// @Success      200      {string}  string  "Base64 encrypted response"
// @Failure      400      {object}  map[string]string
// @Failure      500      {object}  map[string]string
// @Router       /v1/payments/wa-flow-endpoint [post]
func (h *Handler) WAFlowEndpoint(c *gin.Context) {
	if h.waPrivateKey == nil {
		logger.Error.Printf("WhatsApp Flow private key not configured")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "endpoint not configured"})
		return
	}

	var encReq waflow.EncryptedRequest
	if err := c.ShouldBindJSON(&encReq); err != nil {
		logger.Error.Printf("Failed to bind WA Flow request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	// Decrypt the request
	decrypted, aesKey, iv, err := waflow.DecryptRequest(h.waPrivateKey, encReq)
	if err != nil {
		logger.Error.Printf("Failed to decrypt WA Flow request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "decryption failed"})
		return
	}

	logger.Info.Printf("WA Flow action=%s screen=%s data=%v", decrypted.Action, decrypted.Screen, decrypted.Data)

	// Process action
	var response waflow.FlowResponse

	switch decrypted.Action {
	case "ping":
		response = waflow.FlowResponse{
			Data: map[string]interface{}{
				"status": "active",
			},
		}

	case "INIT":
		// Return ORDER_FORM screen with items data from flow message
		response = waflow.FlowResponse{
			Screen: "ORDER_FORM",
			Data:   decrypted.Data,
		}

	case "data_exchange":
		response = h.handleDataExchange(decrypted)

	case "BACK":
		response = waflow.FlowResponse{
			Screen: decrypted.Screen,
			Data:   decrypted.Data,
		}

	default:
		logger.Error.Printf("Unsupported WA Flow action: %s", decrypted.Action)
		response = waflow.FlowResponse{
			Data: map[string]interface{}{
				"error": "unsupported action",
			},
		}
	}

	// Log response before encrypting
	respJSON, _ := json.Marshal(response)
	logger.Info.Printf("WA Flow response: %s", string(respJSON))

	// Encrypt the response
	encrypted, err := waflow.EncryptResponse(aesKey, iv, response)
	if err != nil {
		logger.Error.Printf("Failed to encrypt WA Flow response: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}

	c.String(http.StatusOK, encrypted)
}

// handleDataExchange processes data_exchange action based on screen
func (h *Handler) handleDataExchange(req *waflow.DecryptedRequest) waflow.FlowResponse {
	getData := func(key string) string {
		if v, ok := req.Data[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	return waflow.FlowResponse{
		Screen: "SUMMARY_ORDER",
		Data: map[string]interface{}{
			"nama_penerima":    getData("nama_penerima"),
			"nomor_handphone":  getData("nomor_handphone"),
			"alamat_lengkap":   getData("alamat_lengkap"),
			"provinsi":         getData("provinsi"),
			"kota_kecamatan":   getData("kota_kecamatan"),
			"kode_pos":         getData("kode_pos"),
			"items_text":       getData("items_text"),
			"total_barang":     getData("total_barang"),
			"total_pengiriman": getData("total_pengiriman"),
			"total_biaya":      getData("total_biaya"),
		},
	}
}

// PaymentPage handles GET /pay/:token — serves the payment HTML page
func (h *Handler) PaymentPage(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"message": "Invalid payment token"})
		return
	}

	result := h.paymentService.GetTransactionByToken(token)
	if result.Code != http.StatusOK {
		c.HTML(http.StatusNotFound, "status.html", gin.H{
			"OrderID":     "",
			"StatusTitle": "Transaksi Tidak Ditemukan",
			"StatusMsg":   "Link pembayaran tidak valid atau sudah kadaluarsa.",
			"StatusIcon":  "❌",
			"StatusColor": "red",
		})
		return
	}

	pageData := result.Data.(paymentService.PaymentPageData)

	c.HTML(http.StatusOK, "payment.html", gin.H{
		"SnapToken":     pageData.SnapToken,
		"OrderID":       pageData.OrderID,
		"CustomerName":  pageData.CustomerName,
		"CustomerPhone": pageData.CustomerPhone,
		"GrossAmount":   pageData.GrossAmount,
		"Items":         pageData.Items,
		"SnapJSURL":     h.midtrans.SnapBaseURL(),
		"ClientKey":     h.midtrans.ClientKey,
		"BaseURL":       h.baseURL,
	})
}

// StatusPage handles GET /status/:order_id — serves the payment status HTML page
func (h *Handler) StatusPage(c *gin.Context) {
	orderID := c.Param("order_id")
	if orderID == "" {
		c.HTML(http.StatusBadRequest, "status.html", gin.H{
			"OrderID":     "",
			"StatusTitle": "Order ID Tidak Valid",
			"StatusMsg":   "Silakan periksa kembali link Anda.",
			"StatusIcon":  "❌",
			"StatusColor": "red",
		})
		return
	}

	c.HTML(http.StatusOK, "status.html", gin.H{
		"OrderID": orderID,
		"BaseURL": h.baseURL,
	})
}

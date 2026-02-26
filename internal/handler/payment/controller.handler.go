package payment

import (
	"context"
	types "go-boilerplate/internal/common/type"
	"go-boilerplate/internal/pkg/helper"
	midtransPkg "go-boilerplate/internal/pkg/midtrans"
	paymentService "go-boilerplate/internal/service/payment"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	ctx            context.Context
	paymentService paymentService.IService
	midtrans       *midtransPkg.MidtransClient
	baseURL        string
}

type IHandler interface {
	NewRoutes(e *gin.RouterGroup)
	NewPageRoutes(e *gin.Engine)
}

func NewHandler(ctx context.Context, paymentService paymentService.IService, midtrans *midtransPkg.MidtransClient, baseURL string) IHandler {
	return &Handler{
		ctx:            ctx,
		paymentService: paymentService,
		midtrans:       midtrans,
		baseURL:        baseURL,
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
// @Summary      WhatsApp Flow data exchange endpoint
// @Description  Handles data_exchange action from WhatsApp Flow, forwards order form data to SUMMARY_ORDER screen
// @Tags         WhatsApp Flow
// @Accept       json
// @Produce      json
// @Param        request  body      WAFlowRequest  true  "WhatsApp Flow request"
// @Success      200      {object}  WAFlowResponse
// @Failure      400      {object}  map[string]string
// @Router       /v1/payments/wa-flow-endpoint [post]
func (h *Handler) WAFlowEndpoint(c *gin.Context) {
	var req WAFlowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}

	if req.Action == "data_exchange" {
		c.JSON(http.StatusOK, WAFlowResponse{
			Screen: "SUMMARY_ORDER",
			Data: WAFlowOrderData{
				NamaPenerima:   req.Data.NamaPenerima,
				NomorHandphone: req.Data.NomorHandphone,
				AlamatLengkap:  req.Data.AlamatLengkap,
				Provinsi:       req.Data.Provinsi,
				KotaKecamatan:  req.Data.KotaKecamatan,
				KodePos:        req.Data.KodePos,
				ItemsText:      req.Data.ItemsText,
				TotalBarang:    req.Data.TotalBarang,
				TotalPengiriman: req.Data.TotalPengiriman,
				TotalBiaya:     req.Data.TotalBiaya,
			},
		})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "unsupported action"})
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

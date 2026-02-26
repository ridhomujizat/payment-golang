package payment

import (
	"github.com/gin-gonic/gin"
)

func (h *Handler) NewRoutes(e *gin.RouterGroup) {
	payments := e.Group("/v1/payments")

	payments.POST("/create", h.CreatePayment)
	payments.GET("/status/:order_id", h.CheckStatus)
	payments.POST("/process", h.HandlePaymentResult)
	payments.POST("/callback", h.MidtransCallback)
	payments.POST("/wa-flow-endpoint", h.WAFlowEndpoint)
}

func (h *Handler) NewPageRoutes(e *gin.Engine) {
	e.GET("/pay/:token", h.PaymentPage)
	e.GET("/status/:order_id", h.StatusPage)
}

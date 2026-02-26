package dataroom

import (
	"context"
	types "go-boilerplate/internal/common/type"
	"go-boilerplate/internal/pkg/rabbitmq"
	xampleService "go-boilerplate/internal/service/example"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	ctx           context.Context
	rabbitmq      *rabbitmq.ConnectionManager
	xampleService xampleService.IService
}

type IHandler interface {
	NewRoutes(e *gin.RouterGroup)
}

func NewHandler(ctx context.Context, rabbitmq *rabbitmq.ConnectionManager, xampleService xampleService.IService) IHandler {
	return &Handler{
		ctx:           ctx,
		rabbitmq:      rabbitmq,
		xampleService: xampleService,
	}
}

// GetAllUploadedDucment godoc
// @Summary      Get example data
// @Description  Returns example data for testing purposes
// @Tags         Example
// @Produce      json
// @Success      200  {object}  types.ResponseAPI{data=string}
// @Router       /xample [get]
func (h *Handler) GetAllUploadedDucment(c *gin.Context) {
	send := c.MustGet("send").(func(r *types.Response))

	send(h.xampleService.XampleService())

}

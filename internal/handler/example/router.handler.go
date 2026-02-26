package dataroom

import (
	"github.com/gin-gonic/gin"
)

func (h *Handler) NewRoutes(e *gin.RouterGroup) {
	gorup := e.Group("/xample")

	gorup.
		GET("", h.GetAllUploadedDucment)

}

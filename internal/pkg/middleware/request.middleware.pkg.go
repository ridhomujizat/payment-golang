package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequestInit() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("requestId", uuid.New().String())
		version := c.Request.Header.Get("version")
		if version == "" {
			version = "1.0.0"
		}
		c.Set("version", version)
		c.Set("start-time", time.Now())
		c.Next()
	}
}

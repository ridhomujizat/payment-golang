package middleware

import (
	"net/http"
	_type "go-boilerplate/internal/common/type"
	"go-boilerplate/internal/pkg/helper"
	"time"

	"github.com/gin-gonic/gin"
)

func ResponseInit() gin.HandlerFunc {
	return func(c *gin.Context) {
		shouldDebug := gin.Mode() == gin.DebugMode
		c.Set("send", func(r *_type.Response) {
			if r.Message == "" {
				r.Message = "Success"
			}
			if r.Code == 0 {
				r.Code = http.StatusOK
			}

			response := _type.ResponseAPI{
				Message: r.Message,
				Data:    r.Data,
			}

			if shouldDebug {
				startTime := func() time.Time {
					if value, exists := c.Get("start-time"); exists || value != nil {
						if t, ok := value.(time.Time); ok {
							return t
						}
					}
					return time.Now()
				}()
				endTime := time.Now()

				response.Debug = &_type.ResponseAPIDebug{
					RequestID: c.GetString("requestId"),
					Version:   c.GetString("version"),
					StartTime: startTime,
					EndTime:   endTime,
					RuntimeMs: endTime.Sub(startTime).Milliseconds(),

					Error: func() *string {
						if r.Error != nil {
							return helper.StringPtr(r.Error.Error())
						}
						return nil
					}(),
				}
			}

			c.Abort()
			c.JSON(r.Code, response)
		})

		c.Next()
	}
}

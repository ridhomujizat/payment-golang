package middleware

import (
	"net/http"
	_type "go-boilerplate/internal/common/type"
	types "go-boilerplate/internal/common/type"
	"go-boilerplate/internal/pkg/helper"
	"go-boilerplate/internal/pkg/jwt"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		send := c.MustGet("send").(func(r *_type.Response))
		if token == "" {
			send(helper.ParseResponse(&_type.Response{Code: http.StatusUnauthorized, Message: "token not found"}))
			return
		}

		claims, err := jwt.ValidateToken(token)
		if err != nil {
			send(helper.ParseResponse(&_type.Response{Code: http.StatusUnauthorized, Message: "invalid token", Error: err}))
			return
		}

		c.Set("claims", claims)
		c.Set("auth", types.UserWithAuth{
			ID:     claims.ID,
			Email:  claims.Email,
			RoleId: claims.RoleId,
			TeamId: claims.TeamId,
		})
		c.Next()
	}
}

func AuthMiddlewareWithDynamicRole(permisionCode string) gin.HandlerFunc {
	return func(c *gin.Context) {

	}
}

package middleware

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	_type "go-boilerplate/internal/common/type"
	"go-boilerplate/internal/pkg/helper"
	"go-boilerplate/internal/pkg/redis"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type JwtUser struct {
	ID    int  `json:"id"`
	Is2FA bool `json:"is_2fa"`
}

type data struct {
	Data string `json:"data"`
}

var exemptedPaths = []string{
	"/service/account/socioConnectCallback",
	"/admin/generateMasterData",
	"/admin/generateMasterDataNonTransaction",
}

var whitelistedPaths = map[string][]string{
	http.MethodPost: {
		"/api/v1/workflow-studio/duplicate/:id",
		"/api/v1/form-studio/duplicate/:id",
		// "/api/v1/auth/login",
		// add more paths here
	},
	http.MethodPut:   {},
	http.MethodPatch: {},
}

func EncryptMiddleware(rds redis.IRedis) gin.HandlerFunc {
	return func(c *gin.Context) {
		send := c.MustGet("send").(func(r *_type.Response))
		if err := validateHeaders(c, send); err != nil {
			return
		}
		if err := validateJwt(c, rds, send); err != nil {
			return
		}
		if err := validateRequestBody(c, send); err != nil {
			return
		}
		c.Next()
	}
}

func validateHeaders(c *gin.Context, send func(r *_type.Response)) error {
	timeHeader := c.GetHeader("x-time")
	encryptHeader := c.GetHeader("x-encrypt")
	tenantHeader := c.GetHeader("x-tenant")
	host := c.Request.Header.Get("Origin")
	if host == "" {
		host = c.Request.Host
	}

	if tenantHeader != os.Getenv("APP_TENANT") && os.Getenv("DEV") != "1" {
		send(helper.ParseResponse(&_type.Response{
			Code:    http.StatusForbidden,
			Message: "Invalid Tenant",
			Error:   errors.New("invalid Tenant"),
		}))
		return errors.New("invalid Tenant")
	}

	if host != "" && os.Getenv("DEV_HOST") != "" && os.Getenv("DEV_HOST") != "1" {
		appURL := os.Getenv("APP_URL")
		if !strings.Contains(host, appURL) {
			send(helper.ParseResponse(&_type.Response{
				Code:    http.StatusForbidden,
				Message: "Invalid Host",
				Error:   errors.New("invalid Host"),
			}))
			return errors.New("invalid Host")
		}
	}

	if os.Getenv("DEV") != "1" && !isPathExempted(c.Request.URL.Path) {
		intTimeHeader, err := strconv.Atoi(timeHeader)
		if err != nil {
			send(helper.ParseResponse(&_type.Response{
				Code:    http.StatusForbidden,
				Message: "Invalid Headers",
				Error:   errors.New("invalid Headers"),
			}))
			return err
		}
		if err := validateTime(intTimeHeader, encryptHeader); err != nil {
			send(helper.ParseResponse(&_type.Response{
				Code:    http.StatusForbidden,
				Message: "Invalid Headers",
				Error:   err,
			}))
			return err
		}
	}
	return nil
}

func validateJwt(c *gin.Context, rds redis.IRedis, send func(r *_type.Response)) error {
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		user, err := parseJwt(authHeader)
		if err != nil {
			send(helper.ParseResponse(&_type.Response{
				Code:    http.StatusForbidden,
				Message: "Invalid Token",
				Error:   err,
			}))
			return err
		}

		if user.Is2FA {
			keyCache := os.Getenv("APP_TENANT") + ":" + strconv.Itoa(user.ID) + ":2fa"
			token, err := rds.Get(keyCache)
			if err != nil {
				send(helper.ParseResponse(&_type.Response{
					Code:    http.StatusForbidden,
					Message: "not authorized",
					Error:   err,
				}))
				return err
			}
			if token == "" && c.Request.URL.Path != "/auth/2fa-authenticate" {
				send(helper.ParseResponse(&_type.Response{
					Code:    http.StatusForbidden,
					Message: "not authorized #2",
					Error:   errors.New("not authorized #2"),
				}))
				return errors.New("not authorized #2")
			}
		}
	}
	return nil
}

func validateRequestBody(c *gin.Context, send func(r *_type.Response)) error {
	if c.Request.Method == http.MethodPost || c.Request.Method == http.MethodPut || c.Request.Method == http.MethodPatch {
		var payload data

		if isPathWhitelisted(c.FullPath(), c.Request.Method) {
			c.Next()
			return nil
		}

		if err := c.ShouldBind(&payload); err != nil {
			send(helper.ParseResponse(&_type.Response{
				Code:    http.StatusForbidden,
				Message: "Failed to read request body",
				Error:   err,
			}))
			return err
		}

		decryptedData, err := helper.DecryptAESCBC(payload.Data)
		if err != nil {
			send(helper.ParseResponse(&_type.Response{
				Code:    http.StatusForbidden,
				Message: "Failed to decrypt data",
				Error:   err,
			}))
			return err
		}

		var bodyData map[string]interface{}
		if err := json.Unmarshal([]byte(decryptedData), &bodyData); err != nil {
			send(helper.ParseResponse(&_type.Response{
				Code:    http.StatusForbidden,
				Message: "Failed to parse data",
				Error:   err,
			}))
			return err
		}

		c.Set("body", bodyData)
		c.Request.Body = io.NopCloser(bytes.NewBuffer([]byte(decryptedData)))
	}
	return nil
}

func isPathExempted(path string) bool {
	for _, exemptedPath := range exemptedPaths {
		if path == exemptedPath {
			return true
		}
	}
	return false
}

func isPathWhitelisted(path, method string) bool {
	if paths, exists := whitelistedPaths[method]; exists {
		for _, whitelistedPath := range paths {
			if whitelistedPath == path {
				return true
			}
		}
	}
	return false
}

func validateTime(timeHeader int, encryptHeader string) error {
	if timeHeader <= 0 || encryptHeader == "" {
		return errors.New("invalid headers")
	}

	timeValue := time.Unix(int64(timeHeader), 0)

	delta := time.Since(timeValue).Seconds()
	if delta > float64(helper.GetEnvAsInt("HEADER_TIME")) {
		return errors.New("invalid time")
	}

	message := os.Getenv("HEADER_CODE") + ":" + strconv.Itoa(timeHeader)
	computed, err := helper.HMACSHA256(message)
	if err != nil {
		return err
	}

	if computed != encryptHeader {
		return errors.New("invalid encryption")
	}
	return nil
}

func parseJwt(token string) (*JwtUser, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return nil, errors.New("invalid token format")
	}

	payload, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	var user JwtUser
	if err := json.Unmarshal(payload, &user); err != nil {
		return nil, err
	}

	return &user, nil
}

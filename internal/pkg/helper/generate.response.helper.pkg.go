package helper

import (
	"net/http"
	_type "go-boilerplate/internal/common/type"
)

func ParseResponse(r *_type.Response) *_type.Response {
	if r.Code < 200 || r.Code >= 599 {
		r.Code = http.StatusInternalServerError
	}
	if r.Message == "" {
		generateMessage(r)
	}
	return r
}

func generateMessage(r *_type.Response) {
	switch {
	case r.Code == http.StatusOK:
		r.Message = "Success"
	case r.Code == http.StatusCreated:
		r.Message = "Created"
	case r.Code == http.StatusBadRequest:
		r.Message = "Bad Request"
	case r.Code == http.StatusUnauthorized:
		r.Message = "Unauthorized"
	case r.Code == http.StatusForbidden:
		r.Message = "Forbidden"
	case r.Code == http.StatusNotFound:
		r.Message = "Not Found"
	case r.Code == http.StatusMethodNotAllowed:
		r.Message = "Method Not Allowed"
	case r.Code == http.StatusInternalServerError:
		r.Message = "Internal Server Error"
	case r.Code == http.StatusServiceUnavailable:
		r.Message = "Service Unavailable"
	default:
		r.Message = http.StatusText(r.Code)
	}
}

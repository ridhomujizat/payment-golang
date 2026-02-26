package activitylog

import (
	types "go-boilerplate/internal/common/type"
	"go-boilerplate/internal/pkg/helper"
	"net/http"
)

func (s *Service) XampleService() *types.Response {
	data := "This Xample data"

	return helper.ParseResponse(&types.Response{
		Code:    http.StatusOK,
		Message: "DataRoom deleted successfully",
		Data:    data,
		Error:   nil,
	})
}

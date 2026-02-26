package enum

import (
	"github.com/go-playground/validator/v10"
)

type Enum interface {
	ToString() string
	IsValid() bool
}

func ValidateEnum(fl validator.FieldLevel) bool {
	value := fl.Field().Interface().(Enum)
	return value.IsValid()
}

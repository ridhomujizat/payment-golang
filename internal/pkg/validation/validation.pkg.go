package validation

import (
	"errors"
	"fmt"
	"go-boilerplate/internal/common/enum"
	types "go-boilerplate/internal/common/type"
	"reflect"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin/binding"

	"github.com/go-playground/validator/v10"
)

var val *validator.Validate

var validationMessages = map[string]string{
	"e164":         "must be a e164 formatted phone number",
	"required":     "is required",
	"url":          "must be a valid URL",
	"datetime":     "must be a valid date-time format (2006-01-02T15:04:05Z07:00)",
	"number":       "must be a number",
	"oneof":        "must be one of the allowed values: %s",
	"email":        "must be a valid email address",
	"min":          "must be greater than or equal to %s",
	"max":          "must be less than or equal to %s",
	"len":          "must have the exact length of %s",
	"alpha":        "must contain only alphabetic characters",
	"alphanum":     "must contain only alphanumeric characters",
	"eqfield":      "must be equal to the value of the %s field",
	"nefield":      "must not be equal to the value of the %s field",
	"gt":           "must be greater than %s",
	"gte":          "must be greater than or equal to %s",
	"lt":           "must be less than %s",
	"lte":          "must be less than or equal to %s",
	"excludes":     "must not contain the value %s",
	"excludesall":  "must not contain any of the values: %s",
	"enum":         "must be one of the allowed enum values: %s",
	"stringToBool": "must be a boolean value",
	"password":     "must be at least 8 characters long and contain uppercase, lowercase, number, and special character",
	"phone":        "must be a valid phone number",
}

func Setup() error {
	val = validator.New(validator.WithRequiredStructEnabled())

	if err := registerValidations(val); err != nil {
		return fmt.Errorf("failed to register custom validations: %w", err)
	}

	val.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := registerValidations(v); err != nil {
			return fmt.Errorf("failed to register custom validations in Gin engine: %w", err)
		}
	} else {
		return fmt.Errorf("failed to get validation engine")
	}

	return nil
}

func registerValidations(v *validator.Validate) error {
	if err := v.RegisterValidation("enum", enum.ValidateEnum); err != nil {
		return fmt.Errorf("failed to register enum validation: %w", err)
	}
	if err := v.RegisterValidation("stringToBool", types.ValidateStringToBool); err != nil {
		return fmt.Errorf("failed to register stringToBool validation: %w", err)
	}
	if err := v.RegisterValidation("mapStringString", validateMapStringString); err != nil {
		return fmt.Errorf("failed to register map string validation: %w", err)
	}
	if err := v.RegisterValidation("mapStringInterface", validateNestedMap); err != nil {
		return fmt.Errorf("failed to register nested map validation: %w", err)
	}
	if err := v.RegisterValidation("password", validatePassword); err != nil {
		return fmt.Errorf("failed to register password validation: %w", err)
	}
	return nil
}

// validatePassword checks if the password meets the following criteria:
// - At least 8 characters long
// - Contains at least one uppercase letter
// - Contains at least one lowercase letter
// - Contains at least one number
// - Contains at least one special character
func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	// Check minimum length
	if len(password) < 8 {
		return false
	}

	// Check for uppercase
	if !regexp.MustCompile(`[A-Z]`).MatchString(password) {
		return false
	}

	// Check for lowercase
	if !regexp.MustCompile(`[a-z]`).MatchString(password) {
		return false
	}

	// Check for number
	if !regexp.MustCompile(`[0-9]`).MatchString(password) {
		return false
	}

	// Check for special character
	if !regexp.MustCompile(`[^a-zA-Z0-9]`).MatchString(password) {
		return false
	}

	return true
}

func Validate(payload interface{}) error {
	if err := val.Struct(payload); err != nil {
		var errorMessages []string

		validationErrors := parsingErrorValidate(err)
		if validationErrors != "" {
			errorMessages = append(errorMessages, validationErrors)
		}
		message := "Validation failed: " + strings.Join(errorMessages, ", ")
		return errors.New(message)
	}

	return nil
}

func parsingErrorValidate(err error) string {
	var errs validator.ValidationErrors
	if errors.As(err, &errs) {
		var sb strings.Builder
		for _, e := range errs {
			name := e.Namespace()
			field := e.Field()
			tag := e.Tag()
			param := e.Param()
			tp := e.Type()

			msg := validationMessages[tag]
			switch tag {
			case "enum":
				msg = fmt.Sprintf(msg, tp)
			default:
				if strings.Contains(msg, "%s") {
					msg = fmt.Sprintf(msg, param)
				}
			}
			sb.WriteString(fmt.Sprintf("%s: %s %s", name, field, msg))
			sb.WriteString(", ")
		}
		return strings.TrimSuffix(sb.String(), ", ")
	}
	return err.Error()
}

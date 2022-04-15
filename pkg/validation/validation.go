// package validation defines some custom validations and provides
// methods for creating instances of validator.Validation with
// said custom validations mapped
package validation

import (
	"regexp"

	"github.com/go-playground/validator/v10"
)

func New() *validator.Validate {
	v := validator.New()

	allowedRunesRegexp := regexp.MustCompile(`^[\p{L}\p{N}\-_ ]*$`)
	v.RegisterValidation("allowed-runes", func(fl validator.FieldLevel) bool {
		return allowedRunesRegexp.MatchString(fl.Field().String())
	})
	return v
}

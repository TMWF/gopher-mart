package validation

import (
	"fmt"
	"log/slog"

	"github.com/go-playground/validator"
)

func IsValidRequest(req any, validate *validator.Validate, logger *slog.Logger) bool {
	err := validate.Struct(req)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			logger.Error(fmt.Sprintf("Поле '%s' не прошло валидацию '%s' (значение: %v)\n",
				err.Field(), err.Tag(), err.Value()))
		}
		return false
	}
	return true
}

package handler

import (
	"errors"

	"shop_project_be/internal/domain"
	"shop_project_be/pkg/response"
	appvalidator "shop_project_be/pkg/validator"

	"github.com/gofiber/fiber/v3"
)

// validate is shared by handlers to validate the DTO AFTER fields
// derived from the token (e.g. user_id) are injected. Binding itself is done
// with SkipValidation so it does not fail on fields populated later.
var validate = appvalidator.New()

// bindBody binds the JSON body into out without auto-validation.
func bindBody(c fiber.Ctx, out any) error {
	return c.Bind().SkipValidation(true).Body(out)
}

// bindQuery binds the query string into out without auto-validation.
func bindQuery(c fiber.Ctx, out any) error {
	return c.Bind().SkipValidation(true).Query(out)
}

// writeError maps a usecase error to an HTTP response. Sentinels from the
// domain layer get their proper status code; anything else falls back to the
// caller-supplied status so business/validation messages stay visible.
func writeError(c fiber.Ctx, fallback int, err error) error {
	switch {
	case errors.Is(err, domain.ErrInternal):
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	case errors.Is(err, domain.ErrNotFound):
		return response.Error(c, fiber.StatusNotFound, err.Error(), err)
	case errors.Is(err, domain.ErrInvalidID):
		return response.Error(c, fiber.StatusBadRequest, err.Error(), err)
	case errors.Is(err, domain.ErrDuplicateInvoice):
		return response.Error(c, fiber.StatusConflict, err.Error(), err)
	case errors.Is(err, domain.ErrInvalidSignature):
		return response.Error(c, fiber.StatusForbidden, err.Error(), err)
	case errors.Is(err, domain.ErrPaymentAccessDenied):
		return response.Error(c, fiber.StatusForbidden, err.Error(), err)
	default:
		return response.Error(c, fallback, err.Error(), err)
	}
}

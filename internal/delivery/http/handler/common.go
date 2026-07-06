package handler

import (
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

package handler

import (
	appvalidator "shop_project_be/pkg/validator"

	"github.com/gofiber/fiber/v3"
)

// validate dipakai bersama oleh handler untuk memvalidasi DTO SETELAH field
// yang berasal dari token (mis. user_id) di-inject. Binding sendiri dilakukan
// dengan SkipValidation agar tidak gagal pada field yang baru diisi belakangan.
var validate = appvalidator.New()

// bindBody mengikat JSON body ke out tanpa auto-validation.
func bindBody(c fiber.Ctx, out any) error {
	return c.Bind().SkipValidation(true).Body(out)
}

// bindQuery mengikat query string ke out tanpa auto-validation.
func bindQuery(c fiber.Ctx, out any) error {
	return c.Bind().SkipValidation(true).Query(out)
}

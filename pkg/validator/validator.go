// Package validator provides a fiber.StructValidator implementation (Fiber v3)
// based on go-playground/validator. By registering it in fiber.Config,
// the `validate:"..."` tag on DTOs is evaluated automatically whenever a handler
// memanggil c.Bind().Body()/.Query()/.dst.
package validator

import "github.com/go-playground/validator/v10"

// StructValidator wraps *validator.Validate to satisfy the
// fiber.StructValidator: Validate(out any) error.
type StructValidator struct {
	validate *validator.Validate
}

// New builds a ready-to-use StructValidator. The validator instance is safe for
// concurrent use (concurrency-safe) and caches struct info, so create it
// once at startup.
func New() *StructValidator {
	return &StructValidator{
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
}

// Validate runs struct validation according to the `validate` tag on each field.
// out is a pointer to the struct produced by Fiber binding.
func (v *StructValidator) Validate(out any) error {
	return v.validate.Struct(out)
}

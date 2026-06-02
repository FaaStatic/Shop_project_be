// Package validator menyediakan implementasi fiber.StructValidator (Fiber v3)
// berbasis go-playground/validator. Dengan mendaftarkannya pada fiber.Config,
// tag `validate:"..."` pada DTO akan otomatis dievaluasi setiap kali handler
// memanggil c.Bind().Body()/.Query()/.dst.
package validator

import "github.com/go-playground/validator/v10"

// StructValidator membungkus *validator.Validate agar memenuhi interface
// fiber.StructValidator: Validate(out any) error.
type StructValidator struct {
	validate *validator.Validate
}

// New membuat StructValidator siap pakai. Instance validator aman dipakai
// bersamaan (concurrency-safe) dan meng-cache info struct, jadi cukup dibuat
// sekali saat startup.
func New() *StructValidator {
	return &StructValidator{
		validate: validator.New(validator.WithRequiredStructEnabled()),
	}
}

// Validate menjalankan validasi struct sesuai tag `validate` pada field.
// out adalah pointer ke struct hasil binding Fiber.
func (v *StructValidator) Validate(out any) error {
	return v.validate.Struct(out)
}

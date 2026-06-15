package requestdto

type UserLoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=100"`
	Password string `json:"password" validate:"required,min=6"`
}

// UserRegisterRequest hanya untuk pendaftaran staff. Role tidak diterima dari
// client dan selalu dipaksa ke "staff"; admin/superadmin dibuat langsung di DB.
type UserRegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=100"`
	Password string `json:"password" validate:"required,min=6"`
}

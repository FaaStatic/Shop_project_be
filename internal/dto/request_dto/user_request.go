package requestdto

type UserLoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=100"`
	Password string `json:"password" validate:"required,min=6"`
}

// UserRegisterRequest is only for staff registration. The role is not accepted from
// the client and is always forced to "staff"; admin/superadmin are created directly in the DB.
type UserRegisterRequest struct {
	Username string `json:"username" validate:"required,min=3,max=100"`
	Password string `json:"password" validate:"required,min=6"`
}

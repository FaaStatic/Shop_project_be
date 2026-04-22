package responsedto

type UserLoginResponse struct {
	ID           uint   `json:"id"`
	Username     string `json:"username"`
	Role         string `json:"role"`
	Token        string `json:"token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	ExpiredTime  int64  `json:"token_valid,omitempty"`
}

type UserRegisterResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

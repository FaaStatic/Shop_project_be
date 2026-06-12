package handler

import (
	"shop_project_be/internal/delivery/http/middleware"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	"shop_project_be/pkg/response"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type UserHandler struct {
	usecase domain.UserUsecase
	log     *zap.Logger
}

func NewUserHandler(usecase domain.UserUsecase, log *zap.Logger) *UserHandler {
	return &UserHandler{usecase: usecase, log: log}
}

// Register godoc
//
//	@Summary		Register user baru
//	@Description	Membuat akun user baru. Hanya superadmin yang dapat mengakses endpoint ini.
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		requestdto.UserRegisterRequest	true	"Data user baru"
//	@Success		201		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Router			/api/users [post]
func (h *UserHandler) Register(c fiber.Ctx) error {
	var req requestdto.UserRegisterRequest
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}

	result, err := h.usecase.RegisterUser(c.Context(), &req)
	if err != nil {
		status := fiber.StatusInternalServerError
		if result != nil && result.Status != 0 {
			status = result.Status
		}
		return response.Error(c, status, err.Error(), err)
	}
	return response.Success(c, fiber.StatusCreated, "register success", result)
}

// Login godoc
//
//	@Summary		Login user
//	@Description	Autentikasi user dan menghasilkan access & refresh token.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requestdto.UserLoginRequest	true	"Kredensial login"
//	@Success		200		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		401		{object}	response.APIResponse
//	@Router			/auth/login [post]
func (h *UserHandler) Login(c fiber.Ctx) error {
	var req requestdto.UserLoginRequest
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}

	result, err := h.usecase.UserLogin(c.Context(), &req)
	if err != nil {
		return response.Error(c, fiber.StatusUnauthorized, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "login success", result)
}

// Logout godoc
//
//	@Summary		Logout user
//	@Description	Menghapus session user yang sedang login (token jadi tidak valid).
//	@Tags			Auth
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse
//	@Failure		401	{object}	response.APIResponse
//	@Router			/api/users/logout [post]
func (h *UserHandler) Logout(c fiber.Ctx) error {
	token, _ := c.Locals("access_token").(string)
	if token == "" {
		return response.Error(c, fiber.StatusUnauthorized, "token not found", nil)
	}
	userID := middleware.GetUserID(c)
	if err := h.usecase.Logout(c.Context(), token, userID); err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "logout failed", err)
	}
	return response.Success(c, fiber.StatusOK, "logout success", nil)
}

// OnlineUsers godoc
//
//	@Summary		Daftar kasir online
//	@Description	Mengembalikan daftar user (kasir) yang sedang online saat ini.
//	@Tags			Users
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.APIResponse
//	@Failure		500	{object}	response.APIResponse
//	@Router			/api/users/online [get]
func (h *UserHandler) OnlineUsers(c fiber.Ctx) error {
	users, err := h.usecase.ListOnlineUsers(c.Context())
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, "failed to get online users", err)
	}
	return response.Success(c, fiber.StatusOK, "online users fetched", users)
}

package handler

import (
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
//	@Summary		Register staff baru
//	@Description	Pendaftaran akun staff (publik). Role selalu dipaksa ke "staff"; admin/superadmin dibuat langsung lewat DB.
//	@Tags			Auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requestdto.UserRegisterRequest	true	"Data staff baru"
//	@Success		201		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		409		{object}	response.APIResponse
//	@Router			/auth/register [post]
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

package handler

import (
	"errors"

	"shop_project_be/internal/delivery/http/middleware"
	"shop_project_be/internal/domain"
	requestdto "shop_project_be/internal/dto/request_dto"
	"shop_project_be/pkg/response"

	"github.com/gofiber/fiber/v3"
	"go.uber.org/zap"
)

type PaymentHandler struct {
	usecase domain.PaymentUsecase
	log     *zap.Logger
}

func NewPaymentHandler(usecase domain.PaymentUsecase, log *zap.Logger) *PaymentHandler {
	return &PaymentHandler{usecase: usecase, log: log}
}

// ChargeQris godoc
//
//	@Summary		Create QRIS payment
//	@Description	Creates a QRIS charge via Midtrans. The response contains qr_url/qr_string to display as a QR code.
//	@Tags			Payments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		requestdto.ChargeQrisRequest	true	"QRIS payment cart"
//	@Success		201		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/api/payments/qris [post]
func (h *PaymentHandler) ChargeQris(c fiber.Ctx) error {
	var req requestdto.ChargeQrisRequest
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	req.UserId = middleware.GetUserID(c)
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	res, err := h.usecase.ChargeQris(c.Context(), &req)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusCreated, "qris payment created", res)
}

// ChargeCard godoc
//
//	@Summary		Create debit/credit card payment
//	@Description	Creates a card charge via Midtrans using the token_id from client-side tokenization. If 3DS is needed, the response contains redirect_url.
//	@Tags			Payments
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			request	body		requestdto.ChargeCardRequest	true	"Cart + card token"
//	@Success		201		{object}	response.APIResponse
//	@Failure		400		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/api/payments/card [post]
func (h *PaymentHandler) ChargeCard(c fiber.Ctx) error {
	var req requestdto.ChargeCardRequest
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid request body", err)
	}
	req.UserId = middleware.GetUserID(c)
	if err := validate.Validate(&req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "validation failed", err)
	}
	res, err := h.usecase.ChargeCard(c.Context(), &req)
	if err != nil {
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusCreated, "card payment created", res)
}

// Status godoc
//
//	@Summary		Check payment status
//	@Description	Returns the current payment status from the DB. Used by Flutter to poll after showing the QR / completing 3DS.
//	@Tags			Payments
//	@Produce		json
//	@Security		BearerAuth
//	@Param			order_id	path		string	true	"Order ID (= no_invoice)"
//	@Success		200			{object}	response.APIResponse
//	@Failure		403			{object}	response.APIResponse
//	@Failure		404			{object}	response.APIResponse
//	@Router			/api/payments/{order_id}/status [get]
func (h *PaymentHandler) Status(c fiber.Ctx) error {
	orderID := c.Params("order_id")
	role, _ := c.Locals("role").(string)
	res, err := h.usecase.GetStatus(c.Context(), orderID, middleware.GetUserID(c), role)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentAccessDenied) {
			return response.Error(c, fiber.StatusForbidden, "akses ditolak", err)
		}
		return response.Error(c, fiber.StatusNotFound, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "payment status fetched", res)
}

// Notification godoc
//
//	@Summary		Midtrans notification webhook
//	@Description	Midtrans HTTP(S) Notification endpoint. PUBLIC (no JWT) — authenticity is validated via signature_key. Register this URL in the Midtrans dashboard.
//	@Tags			Payments
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requestdto.MidtransNotificationRequest	true	"Midtrans notification payload"
//	@Success		200		{object}	response.APIResponse
//	@Failure		403		{object}	response.APIResponse
//	@Failure		500		{object}	response.APIResponse
//	@Router			/payments/notification [post]
func (h *PaymentHandler) Notification(c fiber.Ctx) error {
	var req requestdto.MidtransNotificationRequest
	if err := bindBody(c, &req); err != nil {
		return response.Error(c, fiber.StatusBadRequest, "invalid notification body", err)
	}
	if err := h.usecase.HandleNotification(c.Context(), &req); err != nil {
		// Invalid signature → 403 (do not retry). Other errors → 500 so
		// Midtrans resends the notification.
		if err.Error() == "invalid signature" {
			return response.Error(c, fiber.StatusForbidden, "invalid signature", err)
		}
		return response.Error(c, fiber.StatusInternalServerError, err.Error(), err)
	}
	return response.Success(c, fiber.StatusOK, "notification processed", nil)
}

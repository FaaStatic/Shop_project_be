package usecase

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"shop_project_be/infrastructure/fcm"
	"shop_project_be/internal/domain"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type fcmUsecase struct {
	fcm     *fcm.Sender
	fcmRepo domain.DeviceTokenRepository
	log     *zap.Logger
}

func NewFcmUsecase(client *fcm.Sender, repo domain.DeviceTokenRepository, logger *zap.Logger) domain.DeviceTokenUsecase {
	return &fcmUsecase{
		fcm:     client,
		fcmRepo: repo,
		log:     logger,
	}
}

// HandleLogout implements [domain.DeviceTokenUsecase].
func (f *fcmUsecase) HandleLogout(ctx context.Context, token string) error {
	return f.fcmRepo.DetachDeviceTokenFromUser(ctx, token)
}

// NotifyPaymentResult implements [domain.DeviceTokenUsecase].
func (f *fcmUsecase) NotifyPaymentResult(ctx context.Context, userID string, orderID string, success bool, amount int64) error {
	userid, err := uuid.Parse(userID)
	if err != nil {
		return fmt.Errorf("invalid userID: %w", err)
	}
	tokens, err := f.fcmRepo.GetDeviceTokensByUserID(ctx, userid)
	if err != nil {
		return fmt.Errorf("failed to get device tokens: %w", err)
	}

	if len(tokens) == 0 {
		f.log.Info("No device tokens found for user", zap.String("userID", userID))
		return nil
	}
	var payload domain.Payload
	if success {
		payload = domain.Payload{
			Title: "Pembayaran Berhasil",
			Body:  fmt.Sprintf("Pembayaran %s untuk pesanan %s berhasil.", formatRupiah(amount), orderID),
			Data: map[string]string{
				"order_id": orderID,
				"status":   "success",
				"amount":   fmt.Sprintf("%d", amount),
			},
		}
	} else {
		payload = domain.Payload{
			Title: "Pembayaran Gagal",
			Body:  fmt.Sprintf("Pembayaran %s untuk pesanan %s gagal. Silakan coba lagi.", formatRupiah(amount), orderID),
			Data: map[string]string{
				"order_id": orderID,
				"status":   "failed",
				"amount":   fmt.Sprintf("%d", amount),
			},
		}
	}

	invalidTokens, err := f.fcm.SendToToken(ctx, tokens, payload)
	if err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	if len(invalidTokens) > 0 {
		f.log.Info("Removing invalid device tokens", zap.Strings("tokens", invalidTokens))
		// The notification was already sent; a token-cleanup failure is only logged
		// so the caller (e.g. the webhook flow) does not treat it as a failure.
		if err := f.fcmRepo.DeleteDeviceToken(ctx, invalidTokens); err != nil {
			f.log.Error("failed to delete invalid device tokens", zap.Error(err))
		}
	}

	return nil
}

// formatRupiah memformat nominal ke bentuk "Rp15.000" (pemisah ribuan titik).
func formatRupiah(amount int64) string {
	s := strconv.FormatInt(amount, 10)
	var b strings.Builder
	b.WriteString("Rp")
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// RegisterDevice implements [domain.DeviceTokenUsecase].
func (f *fcmUsecase) RegisterDevice(ctx context.Context, userID string, token string, platform string, deviceID string) error {
	dt := &domain.DeviceToken{
		Token:    token,
		DeviceID: deviceID,
		Platform: domain.Platform(platform),
	}

	if userID != "" {
		userUUID, err := uuid.Parse(userID)
		if err != nil {
			return fmt.Errorf("invalid userID: %w", err)
		}
		dt.UserID = &userUUID
	}

	if err := f.fcmRepo.RegisterDeviceToken(ctx, dt); err != nil {
		return fmt.Errorf("failed to register device token: %w", err)
	}

	return nil
}

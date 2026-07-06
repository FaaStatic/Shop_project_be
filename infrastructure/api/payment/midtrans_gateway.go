// Package payment contains the adapter to the external payment gateway (Midtrans).
// This adapter implements the domain.PaymentGateway port so the
// usecase layer does not depend directly on the Midtrans SDK (dependency inversion).
package payment

import (
	"context"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"

	"shop_project_be/internal/domain"

	"github.com/midtrans/midtrans-go"
	"github.com/midtrans/midtrans-go/coreapi"
)

type midtransGateway struct {
	client    coreapi.Client
	serverKey string
}

// NewMidtransGateway builds the Midtrans Core API adapter. environment: "production"
// or anything else is treated as sandbox.
func NewMidtransGateway(serverKey, environment string) domain.PaymentGateway {
	env := midtrans.Sandbox
	if strings.EqualFold(environment, "production") {
		env = midtrans.Production
	}
	var client coreapi.Client
	client.New(serverKey, env)
	return &midtransGateway{client: client, serverKey: serverKey}
}

// ChargeQris creates a QRIS transaction. The "gopay" acquirer produces a QR that can
// be paid via any app that supports QRIS.
func (g *midtransGateway) ChargeQris(_ context.Context, in domain.GatewayChargeInput) (*domain.GatewayChargeResult, error) {
	req := &coreapi.ChargeReq{
		PaymentType: coreapi.PaymentTypeQris,
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  in.OrderID,
			GrossAmt: in.GrossAmount,
		},
		CustomerDetails: toCustomerDetails(in.Customer),
		Qris:            &coreapi.QrisDetails{Acquirer: "gopay"},
	}
	if items := toItemDetails(in.Items); len(items) > 0 {
		req.Items = &items
	}
	res, mErr := g.client.ChargeTransaction(req)
	if mErr != nil {
		return nil, errors.New(mErr.GetMessage())
	}
	return mapChargeResponse(res), nil
}

// ChargeCard charges the card using a single-use token from the client. 3DS
// is enabled (Authentication) for security; if needed, RedirectURL holds the
// 3DS URL the client must open.
func (g *midtransGateway) ChargeCard(_ context.Context, in domain.GatewayChargeInput) (*domain.GatewayChargeResult, error) {
	req := &coreapi.ChargeReq{
		PaymentType: coreapi.PaymentTypeCreditCard,
		TransactionDetails: midtrans.TransactionDetails{
			OrderID:  in.OrderID,
			GrossAmt: in.GrossAmount,
		},
		CustomerDetails: toCustomerDetails(in.Customer),
		CreditCard: &coreapi.CreditCardDetails{
			TokenID:        in.CardTokenID,
			Authentication: in.Authentication,
		},
	}
	if items := toItemDetails(in.Items); len(items) > 0 {
		req.Items = &items
	}
	res, mErr := g.client.ChargeTransaction(req)
	if mErr != nil {
		return nil, errors.New(mErr.GetMessage())
	}
	return mapChargeResponse(res), nil
}

// CheckStatus fetches the authoritative transaction status from Midtrans. Used when
// processing a notification so it does not rely on a replayable payload.
func (g *midtransGateway) CheckStatus(_ context.Context, orderID string) (*domain.GatewayChargeResult, error) {
	res, mErr := g.client.CheckTransaction(orderID)
	if mErr != nil {
		return nil, errors.New(mErr.GetMessage())
	}
	return &domain.GatewayChargeResult{
		TransactionID:     res.TransactionID,
		OrderID:           res.OrderID,
		PaymentType:       res.PaymentType,
		TransactionStatus: res.TransactionStatus,
		FraudStatus:       res.FraudStatus,
		StatusCode:        res.StatusCode,
		ExpiryTime:        res.ExpiryTime,
	}, nil
}

// VerifySignature matches the notification signature_key against the hash computed
// from the ServerKey: SHA512(order_id + status_code + gross_amount + serverKey).
func (g *midtransGateway) VerifySignature(orderID, statusCode, grossAmount, signatureKey string) bool {
	raw := orderID + statusCode + grossAmount + g.serverKey
	sum := sha512.Sum512([]byte(raw))
	expected := hex.EncodeToString(sum[:])
	// Constant-time so the comparison does not leak the position of the first
	// differing byte via timing. Hex is lowercased first (Midtrans sends lowercase).
	return subtle.ConstantTimeCompare([]byte(expected), []byte(strings.ToLower(signatureKey))) == 1
}

func mapChargeResponse(res *coreapi.ChargeResponse) *domain.GatewayChargeResult {
	out := &domain.GatewayChargeResult{
		TransactionID:     res.TransactionID,
		OrderID:           res.OrderID,
		PaymentType:       res.PaymentType,
		TransactionStatus: res.TransactionStatus,
		FraudStatus:       res.FraudStatus,
		StatusCode:        res.StatusCode,
		QRString:          res.QRString,
		RedirectURL:       res.RedirectURL,
		ExpiryTime:        res.ExpiryTime,
	}
	// The QR image URL is in actions[name=generate-qr-code].
	for _, a := range res.Actions {
		if a.Name == "generate-qr-code" {
			out.QRURL = a.URL
			break
		}
	}
	return out
}

func toItemDetails(items []domain.GatewayItem) []midtrans.ItemDetails {
	out := make([]midtrans.ItemDetails, 0, len(items))
	for _, it := range items {
		out = append(out, midtrans.ItemDetails{
			ID:    it.ID,
			Name:  it.Name,
			Price: it.Price,
			Qty:   it.Qty,
		})
	}
	return out
}

func toCustomerDetails(c domain.GatewayCustomer) *midtrans.CustomerDetails {
	if c.FirstName == "" && c.Email == "" && c.Phone == "" {
		return nil
	}
	return &midtrans.CustomerDetails{
		FName: c.FirstName,
		Email: c.Email,
		Phone: c.Phone,
	}
}

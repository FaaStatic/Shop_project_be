package handler

import (
	"shop_project_be/internal/domain"

	"go.uber.org/zap"
)

type TransactionHandler struct {
	trxUsecase domain.TransactionUsecase
	log        *zap.Logger
}

func NewTransactionHandler(trxUsecase domain.TransactionUsecase, log *zap.Logger) *TransactionHandler {
	return &TransactionHandler{
		trxUsecase: trxUsecase,
		log:        log,
	}
}

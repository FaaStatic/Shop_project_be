package usecase

import (
	"strings"
	"time"

	"shop_project_be/internal/constant/paginated"
	"shop_project_be/internal/domain"

	"github.com/google/uuid"
)

func parseCursor(afterID, afterTime *string) (*paginated.CursorMeta, error) {
	var id, at string
	if afterID != nil {
		id = strings.TrimSpace(*afterID)
	}
	if afterTime != nil {
		at = strings.TrimSpace(*afterTime)
	}
	if id == "" || at == "" {
		return nil, nil
	}
	parsedID, err := uuid.Parse(id)
	if err != nil {
		return nil, domain.InvalidID("invalid cursor id format")
	}
	parsedTime, err := time.Parse(paginated.TimeLayout, at)
	if err != nil {
		return nil, domain.Validation("invalid after_time format")
	}
	return &paginated.CursorMeta{AfterTime: parsedTime, AfterID: parsedID}, nil
}

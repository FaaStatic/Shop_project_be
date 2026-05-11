package paginated

import (
	"time"

	"github.com/google/uuid"
)

type CursorMeta struct {
	AfterTime time.Time `json:"after_time"`
	AfterID   uuid.UUID `json:"after_id"`
}

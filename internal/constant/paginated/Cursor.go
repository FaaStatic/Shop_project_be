package paginated

import (
	"time"

	"github.com/google/uuid"
)

type CursorMeta struct {
	AfterTime time.Time `json:"after_time"`
	AfterID   uuid.UUID `json:"after_id"`
}

// TimeLayout is the serialization format for cursor time to/from the client. RFC3339Nano
// is used (not RFC3339) so sub-second precision of created_at is not lost; if
// truncated to seconds, the keyset could skip/repeat rows whose created_at
// differs only in milliseconds.
const TimeLayout = time.RFC3339Nano

// Encode turns the cursor into an (after_id, after_time) pair for the response.
// Safe to call on a nil receiver: on the last page the cursor is nil,
// and calling this returns empty strings instead of a nil-pointer panic.
func (c *CursorMeta) Encode() (afterID string, afterTime string) {
	if c == nil {
		return "", ""
	}
	return c.AfterID.String(), c.AfterTime.Format(TimeLayout)
}

package notifications

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// idempotencyBucket is the time granularity used when hashing. All calls
// landing in the same UTC-aligned 5-minute bucket produce identical keys for
// the same (user, type, title) triple.
const idempotencyBucket = 5 * time.Minute

// IdempotencyKey returns a deterministic idempotency key for a notification
// destined to userID, emitted around now, carrying the given type and title.
// Duplicate notifications fired within the same 5-minute UTC bucket produce
// the same key and are deduplicated by the notificator.
func IdempotencyKey(userID string, now time.Time, typ, title string) string {
	bucket := now.UTC().Truncate(idempotencyBucket).Unix()
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "%s|%d|%s|%s", userID, bucket, typ, title)
	return hex.EncodeToString(h.Sum(nil))
}

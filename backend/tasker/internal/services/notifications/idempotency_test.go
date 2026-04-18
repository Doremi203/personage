package notifications

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIdempotencyKey(t *testing.T) {
	userA := "11111111-1111-1111-1111-111111111111"
	userB := "22222222-2222-2222-2222-222222222222"

	base := time.Date(2026, 4, 18, 10, 2, 30, 0, time.UTC) // bucket 10:00
	sameBucket := base.Add(2 * time.Minute)                // 10:04:30 — same 5m bucket
	nextBucket := base.Add(5 * time.Minute)                // 10:07:30 — next bucket

	t.Run("deterministic for identical inputs", func(t *testing.T) {
		a := IdempotencyKey(userA, base, "upcoming_event", "hello")
		b := IdempotencyKey(userA, base, "upcoming_event", "hello")
		assert.Equal(t, a, b)
	})

	t.Run("stable within the same 5-minute bucket", func(t *testing.T) {
		a := IdempotencyKey(userA, base, "upcoming_event", "hello")
		b := IdempotencyKey(userA, sameBucket, "upcoming_event", "hello")
		assert.Equal(t, a, b)
	})

	t.Run("changes across buckets", func(t *testing.T) {
		a := IdempotencyKey(userA, base, "upcoming_event", "hello")
		b := IdempotencyKey(userA, nextBucket, "upcoming_event", "hello")
		assert.NotEqual(t, a, b)
	})

	t.Run("different users differ", func(t *testing.T) {
		a := IdempotencyKey(userA, base, "upcoming_event", "hello")
		b := IdempotencyKey(userB, base, "upcoming_event", "hello")
		assert.NotEqual(t, a, b)
	})

	t.Run("different types differ", func(t *testing.T) {
		a := IdempotencyKey(userA, base, "upcoming_event", "hello")
		b := IdempotencyKey(userA, base, "schedule_change", "hello")
		assert.NotEqual(t, a, b)
	})

	t.Run("different titles differ", func(t *testing.T) {
		a := IdempotencyKey(userA, base, "upcoming_event", "hello")
		b := IdempotencyKey(userA, base, "upcoming_event", "world")
		assert.NotEqual(t, a, b)
	})

	t.Run("timezone-independent (UTC bucketing)", func(t *testing.T) {
		moscow := time.FixedZone("MSK", 3*3600)
		utc := IdempotencyKey(userA, base, "upcoming_event", "hello")
		local := IdempotencyKey(userA, base.In(moscow), "upcoming_event", "hello")
		assert.Equal(t, utc, local)
	})
}

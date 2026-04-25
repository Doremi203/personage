package domain_test

import (
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/tasker/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestCluster_AddEvent(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	earlier := now.Add(-time.Hour)

	t.Run("first event averages with weight zero", func(t *testing.T) {
		c := domain.Cluster{
			Centroid:   []float32{0, 0, 0},
			EventCount: 0,
			UpdatedAt:  earlier,
		}
		c.AddEvent(domain.EventWithEmbedding{Embedding: []float32{1, 2, 3}}, now)

		assert.Equal(t, []float32{1, 2, 3}, c.Centroid)
		assert.Equal(t, 1, c.EventCount)
		assert.Equal(t, now, c.UpdatedAt)
	})

	t.Run("second event averages with existing centroid", func(t *testing.T) {
		c := domain.Cluster{
			Centroid:   []float32{2, 4, 6},
			EventCount: 2,
			UpdatedAt:  earlier,
		}
		c.AddEvent(domain.EventWithEmbedding{Embedding: []float32{8, 4, 0}}, now)

		assert.InDeltaSlice(t, []float32{4, 4, 4}, c.Centroid, 1e-6)
		assert.Equal(t, 3, c.EventCount)
		assert.Equal(t, now, c.UpdatedAt)
	})

	t.Run("centroid length governs the loop", func(t *testing.T) {
		c := domain.Cluster{
			Centroid:   []float32{1, 2},
			EventCount: 1,
		}
		c.AddEvent(domain.EventWithEmbedding{Embedding: []float32{3, 4, 99, 99}}, now)

		assert.Equal(t, []float32{2, 3}, c.Centroid)
		assert.Equal(t, 2, c.EventCount)
	})
}

func TestClusterID_String(t *testing.T) {
	assert.Equal(t, "abc", domain.ClusterID("abc").String())
}

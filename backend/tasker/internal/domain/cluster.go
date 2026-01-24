package domain

import "time"

type ClusterID string

func (id ClusterID) String() string {
	return string(id)
}

type ClusterStatus string

const (
	ClusterStatusOpen       ClusterStatus = "open"
	ClusterStatusProcessing ClusterStatus = "processing"
	ClusterStatusClosed     ClusterStatus = "closed"
)

type Cluster struct {
	ID         ClusterID
	UserID     UserID
	Centroid   []float32
	EventCount int
	Status     ClusterStatus
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (c *Cluster) AddEvent(e EventWithEmbedding) {
	dim := len(c.Centroid)

	n := float32(c.EventCount)
	for i := 0; i < dim; i++ {
		c.Centroid[i] = (c.Centroid[i]*n + e.Embedding[i]) / (n + 1)
	}
	c.EventCount++
	c.UpdatedAt = time.Now()
}

type ClusterWithSimilarity struct {
	Cluster
	Similarity float64
}

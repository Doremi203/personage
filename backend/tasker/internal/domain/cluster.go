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

type ClusterGenerationOutcome string

const (
	ClusterGenerationOutcomeTaskGenerated ClusterGenerationOutcome = "task_generated"
	ClusterGenerationOutcomeNonActionable ClusterGenerationOutcome = "non_actionable"
	ClusterGenerationOutcomeEmpty         ClusterGenerationOutcome = "empty"
)

type Cluster struct {
	ID                ClusterID
	UserID            UserID
	Centroid          []float32
	EventCount        int
	Status            ClusterStatus
	GenerationOutcome *ClusterGenerationOutcome
	GenerationReason  *string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (c *Cluster) AddEvent(e EventWithEmbedding, now time.Time) {
	dim := len(c.Centroid)

	n := float32(c.EventCount)
	for i := range dim {
		c.Centroid[i] = (c.Centroid[i]*n + e.Embedding[i]) / (n + 1)
	}
	c.EventCount++
	c.UpdatedAt = now
}

type ClusterWithSimilarity struct {
	Cluster
	Similarity float64
}

type ClusterGenerationDiagnostic struct {
	ClusterID          ClusterID
	UserID             UserID
	Status             ClusterStatus
	EventCount         int
	GenerationOutcome  *ClusterGenerationOutcome
	GenerationReason   *string
	GeneratedTaskCount int
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type AdminClusterListItem struct {
	ClusterID         ClusterID
	UserID            UserID
	Status            ClusterStatus
	EventCount        int
	GenerationOutcome *ClusterGenerationOutcome
	GenerationReason  *string
	TaskID            *TaskID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

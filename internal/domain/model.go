package domain

type PipelineStatus string

const (
	StatusSuccess   PipelineStatus = "success"
	StatusFailed    PipelineStatus = "failed"
	StatusRunning   PipelineStatus = "running"
	StatusCancelled PipelineStatus = "cancelled"
	StatusOther     PipelineStatus = "other"
)

type Pipeline struct {
	ID     int64
	Ref    string
	Status PipelineStatus
	WebURL string
}

type ProjectRef struct {
	ProjectID int64
	Ref       string
}

type Snapshot struct {
	Project   ProjectRef
	Pipeline  Pipeline
	Retrieved int64
}

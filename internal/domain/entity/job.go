package entity

import "time"

type JobStatus string
type JobAction string

const (
	JobStatusPending JobStatus = "pending"
	JobStatusRunning JobStatus = "running"
	JobStatusSuccess JobStatus = "success"
	JobStatusFailed  JobStatus = "failed"

	JobActionCreate JobAction = "create"
	JobActionStart  JobAction = "start"
	JobActionStop   JobAction = "stop"
	JobActionRemove JobAction = "remove"
)

type Job struct {
	ID           string
	UserID       string
	ContainerID  string
	ErrorMessage string
	Status       JobStatus
	Action       JobAction
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

package entity

import "time"

type ContainerStatus string

const (
	ContainerStatusCreated    ContainerStatus = "created"
	ContainerStatusRunning    ContainerStatus = "running"
	ContainerStatusPaused     ContainerStatus = "paused"
	ContainerStatusRestarting ContainerStatus = "restarting"
	ContainerStatusExited     ContainerStatus = "exited"
	ContainerStatusDead       ContainerStatus = "dead"
)

type Container struct {
	ID        string
	DockerID  string
	Name      string
	Image     string
	Status    ContainerStatus
	UserID    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

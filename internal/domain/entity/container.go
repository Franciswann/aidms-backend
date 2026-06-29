// Package entity holds the Domain layer's data types - plain structs with
// no third-party dependencies, shared by every other layer.
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
	UserID    string
	DockerID  string
	Name      string
	Image     string
	Status    ContainerStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

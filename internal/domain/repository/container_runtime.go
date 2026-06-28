package repository

// ContainerRuntime defines the container lifecycle operations the system
// needs from whatever engine actually runs containers (Docker today,
// potentially Kubernetes later). Use Case code depends only on this
// interface, never on a specific engine's SDK.
type ContainerRuntime interface {
	Create(image, name string) (dockerID string, err error)
	Start(dockerID string) error
	Stop(dockerID string) error
	Remove(dockerID string) error
}

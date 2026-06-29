// Package docker implements domainrepo.ContainerRuntime against a real
// Docker daemon via the Docker SDK for Go.
package docker

import (
	"context"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"

	domainrepo "github.com/Franciswann/aidms-backend/internal/domain/repository"
)

var _ domainrepo.ContainerRuntime = (*DockerRuntime)(nil)

type DockerRuntime struct {
	cli *client.Client
}

func NewDockerRuntime() (*DockerRuntime, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &DockerRuntime{cli: cli}, nil
}

func (d *DockerRuntime) Create(imageName, name string) (string, error) {
	ctx := context.Background()

	// ImagePull is asynchronous; the response body streams pull progress and
	// must be drained to completion before the image is guaranteed to exist
	// locally for ContainerCreate to use.
	reader, err := d.cli.ImagePull(ctx, imageName, types.ImagePullOptions{})
	if err != nil {
		return "", err
	}
	defer reader.Close()
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return "", err
	}

	resp, err := d.cli.ContainerCreate(ctx, &container.Config{Image: imageName}, nil, nil, nil, name)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

func (d *DockerRuntime) Start(dockerID string) error {
	return d.cli.ContainerStart(context.Background(), dockerID, types.ContainerStartOptions{})
}

func (d *DockerRuntime) Stop(dockerID string) error {
	return d.cli.ContainerStop(context.Background(), dockerID, container.StopOptions{})
}

func (d *DockerRuntime) Remove(dockerID string) error {
	return d.cli.ContainerRemove(context.Background(), dockerID, types.ContainerRemoveOptions{})
}

package main

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"
)

func TestWithRedis(t *testing.T) {
	ctx := context.Background()
	t.Log("creating container requests")
	rabbitmqRequest := testcontainers.ContainerRequest{
		Image:    "rabbitmq:3.8",
		Networks: []string{"testnetwork"},
	}
	fakedataRequest := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context: ".",
		},
		Networks: []string{"testnetwork"},
	}

	t.Log("about to create rabbitmq container")
	rabbitmqContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: rabbitmqRequest,
		Started:          true,
	})
	if err != nil {
		t.Error(err)
	}

	t.Log("about to create fakedata container")
	fakedataContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: fakedataRequest,
		Started:          true,
	})
	if err != nil {
		t.Error(err)
	}
	defer fakedataContainer.Terminate(ctx)
	defer rabbitmqContainer.Terminate(ctx)
}

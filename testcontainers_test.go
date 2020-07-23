package main

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/testcontainers/testcontainers-go"
)

type TestLogConsumer struct {
	Msgs []string
}

func (g *TestLogConsumer) Accept(l testcontainers.Log) {
	fmt.Print(string(l.Content))
}

func TestWithFakedataAndRabbitmq(t *testing.T) {
	ctx := context.Background()

	absPath, err := filepath.Abs("./test-data/data.json")
	if err != nil {
		t.Fatal(err)
	}

	t.Log("creating container requests")
	rabbitmqRequest := testcontainers.ContainerRequest{
		Image:    "rabbitmq:3-management",
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
	ip, err := rabbitmqContainer.ContainerIP(ctx)
	if err != nil {
		t.Error(err)
	}

	fakedataRequest := testcontainers.ContainerRequest{
		FromDockerfile: testcontainers.FromDockerfile{
			Context: ".",
		},
		Networks: []string{"testnetwork"},
		Env: map[string]string{
			"FAKEDATA_RABBITMQ_HOSTNAME":           ip,
			"FAKEDATA_RABBITMQ_QUERIES_ROUTINGKEY": "test",
			"FAKEDATA_FILENAME":                    "/data.json",
		},
		BindMounts: map[string]string{
			absPath: "/data.json",
		},
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

package main

import (
	"context"
	"log"

	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
)

func main() {
	config := &zbc.ClientConfig{
		GatewayAddress:         "localhost:26500",
		UsePlaintextConnection: true,
	}
	client, err := zbc.NewClient(config)
	if err != nil {
		log.Fatal("Failed to connect to " + config.GatewayAddress)
	}
	_, err = client.NewCreateInstanceCommand().BPMNProcessId("doSomethingProcess").LatestVersion().Send(context.Background())
	if err != nil {
		log.Fatalf("error creating process instance: $%s\n", err)
	}
}

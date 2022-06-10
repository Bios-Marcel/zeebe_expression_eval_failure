package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/camunda/zeebe/clients/go/v8/pkg/entities"
	"github.com/camunda/zeebe/clients/go/v8/pkg/worker"
	"github.com/camunda/zeebe/clients/go/v8/pkg/zbc"
)

type handler struct {
	client zbc.Client
}

func main() {
	config := &zbc.ClientConfig{
		GatewayAddress:         "localhost:26500",
		UsePlaintextConnection: true,
	}
	client, err := zbc.NewClient(config)
	if err != nil {
		log.Fatal("Failed to connect to " + config.GatewayAddress)
	}
	handler := handler{client}

	deployWorkflow(client, "./workflow.bpmn")

	client.NewJobWorker().JobType("doSomething").Handler(handler.doSomethingHandler).Open()

	log.Println("Ready")
	shutdownChannel := make(chan os.Signal, 1)
	signal.Notify(shutdownChannel, syscall.SIGINT, syscall.SIGTERM)
	<-shutdownChannel
}

func (h *handler) doSomethingHandler(client worker.JobClient, job entities.Job) {
	variables, err := job.GetVariablesAsMap()
	if err != nil {
		fail(client, job, err)
	}

	outputVariables := make(map[string]any)
	var answer map[string]any
	succeedNextTry, isBool := variables["succeedNextTry"].(bool)
	if !isBool || !succeedNextTry {
		answer = map[string]any{
			"type": map[string]any{
				"id": "",
			},
		}
		outputVariables["succeedNextTry"] = true
	} else {
		answer = map[string]any{
			"answer": map[string]any{
				"type": map[string]any{
					"id": "yes",
				},
			},
		}
		outputVariables["succeedNextTry"] = false
	}

	completionDispatcher, err := client.NewCompleteJobCommand().JobKey(job.GetKey()).VariablesFromMap(outputVariables)
	if err != nil {
		log.Printf("error creating complete completionDispatcher: %s\n", err)
		fail(client, job, err)
		return
	}

	if _, err = completionDispatcher.Send(context.Background()); err != nil {
		fail(client, job, err)
	}

	go func() {
		time.Sleep(1 * time.Second)
		log.Println("Sending message event")
		messageDispatcher, err := h.client.
			NewPublishMessageCommand().
			MessageName("message").
			CorrelationKey(variables["someId"].(string)).
			VariablesFromMap(answer)
		if err != nil {
			panic(err)
		}

		if _, err = messageDispatcher.Send(context.Background()); err != nil {
			panic(err)
		}
		log.Println("Message event sent")
	}()
}

func fail(client worker.JobClient, job entities.Job, err error) {
	if _, err = client.NewFailJobCommand().JobKey(job.GetKey()).Retries(job.GetRetries() - 1).Send(context.Background()); err != nil {
		log.Printf("failed to fail ... %s\n", err)
	}
}

func deployWorkflow(client zbc.Client, workflowPath string) {
	response, err := client.
		NewDeployResourceCommand().
		AddResourceFile(workflowPath).
		Send(context.Background())
	if err != nil {
		log.Fatal("Failed to deploy workflow: ", err.Error())
	}
	log.Println(response.String())
	log.Print("Deployed workflow ")
}

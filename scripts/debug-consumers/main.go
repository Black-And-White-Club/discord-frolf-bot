package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

func main() {
	natsURL := "nats://auth-service:auth-service-local-dev@localhost:4222"
	if len(os.Args) > 1 {
		natsURL = os.Args[1]
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	js, err := jetstream.New(nc)
	if err != nil {
		log.Fatalf("Failed to create JetStream context: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	streamName := "round"
	stream, err := js.Stream(ctx, streamName)
	if err != nil {
		log.Fatalf("Failed to get stream %s: %v", streamName, err)
	}

	fmt.Printf("Consumers for stream '%s':\n", streamName)
	consumers := stream.ListConsumers(ctx)
	for cons := range consumers.Info() {
		fmt.Printf("--------------------------------------------------\n")
		fmt.Printf("Name: %s\n", cons.Name)
		fmt.Printf("Durable: %s\n", cons.Config.Durable)
		fmt.Printf("FilterSubject: %s\n", cons.Config.FilterSubject)
		fmt.Printf("AckWait: %v\n", cons.Config.AckWait)
		fmt.Printf("AckPolicy: %v\n", cons.Config.AckPolicy)
		fmt.Printf("DeliverPolicy: %v\n", cons.Config.DeliverPolicy)

		// Print full config JSON for detail
		configJSON, _ := json.MarshalIndent(cons.Config, "", "  ")
		fmt.Printf("Config: %s\n", string(configJSON))
	}
	if consumers.Err() != nil {
		log.Printf("Error listing consumers: %v", consumers.Err())
	}
}

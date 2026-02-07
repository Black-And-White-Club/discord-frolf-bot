package main

import (
	"context"
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
	consumerName := "discord-round-reminder-sent-v1"

	fmt.Printf("Attempting to delete consumer '%s' from stream '%s'...\n", consumerName, streamName)

	err = js.DeleteConsumer(ctx, streamName, consumerName)
	if err != nil {
		log.Fatalf("Failed to delete consumer: %v", err)
	}

	fmt.Printf("Successfully deleted consumer '%s'.\n", consumerName)
}

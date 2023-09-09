// ./consumer/main.go

package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/pmorelli92/bunnify/bunnify"
)

func main() {
	// Define RabbitMQ server URL.
	amqpServerURL := os.Getenv("AMQP_SERVER_URL")

	// Setup
	queueName := "vega-queue"
	exchangeName := "vega-exchange"
	routingKey := "vega.rpcCreated"

	type RPC struct {
		Method string                 `json:"method"`
		Params map[string]interface{} `json:"params"`
	}

	exitCh := make(chan bool)
	notificationChannel := make(chan bunnify.Notification)
	go func() {
		for {
			select {
			case n := <-notificationChannel:
				fmt.Println(n)
			case <-exitCh:
				return
			}
		}
	}()

	// Create a new RabbitMQ connection.
	connection := bunnify.NewConnection(
		bunnify.WithURI(amqpServerURL),
		bunnify.WithReconnectInterval(1*time.Second),
		bunnify.WithNotificationChannel(notificationChannel))

	connection.Start()

	var rpc bunnify.ConsumableEvent[RPC]
	eventHandler := func(ctx context.Context, event bunnify.ConsumableEvent[RPC]) error {
		rpc = event
		fmt.Println(rpc)
		return nil
	}

	consumer := connection.NewConsumer(
		queueName,
		bunnify.WithQuorumQueue(),
		bunnify.WithBindingToExchange(exchangeName),
		bunnify.WithHandler(routingKey, eventHandler))

	if err := consumer.Consume(); err != nil {
		fmt.Println(err)
	}
	<-exitCh
}

// ./consumer/main.go

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
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
	responseRoutingKey := "vega.response"

	type RPC struct {
		Method string                 `json:"method"`
		Params map[string]interface{} `json:"params"`
	}

	type Response struct {
		Message string `json:"message"`
	}

	exitCh := make(chan bool)
	notificationChannel := make(chan bunnify.Notification)
	go func() {
		for {
			select {
			case n := <-notificationChannel:
				log.Println(n)
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

	publisher := connection.NewPublisher()

	counter := 0
	processRPC := func(ctx context.Context, event bunnify.ConsumableEvent[RPC]) error {
		counter++
		log.Println("Starting processRPC: " + strconv.Itoa(counter))
		log.Println(event)
		rpc := event.Payload
		timeParam, ok := rpc.Params["time"].(float64)

		if ok {
			seconds := int(timeParam)
			time.Sleep(time.Duration(seconds) * time.Second)
			response := bunnify.NewPublishableEvent(Response{
				Message: strconv.Itoa(counter),
			})

			err := publisher.Publish(context.TODO(), exchangeName, responseRoutingKey, response)
			if err != nil {
				log.Println(err)
				return nil
			}
			return nil
		} else {
			log.Println(fmt.Sprintf("RPC %s could not be processed", event.Payload))
			return nil
		}

	}

	consumer := connection.NewConsumer(
		queueName,
		bunnify.WithQuorumQueue(),
		bunnify.WithBindingToExchange(exchangeName),
		bunnify.WithHandler(routingKey, processRPC))

	if err := consumer.Consume(); err != nil {
		log.Println(err)
	}
	<-exitCh
}

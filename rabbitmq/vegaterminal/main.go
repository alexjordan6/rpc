// ./sender/main.go

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pmorelli92/bunnify/bunnify"
)

func main() {

	// Define RabbitMQ server URL.
	amqpServerURL := os.Getenv("AMQP_SERVER_URL")

	// Setup
	exchangeName := "vega-exchange"
	routingKey := "vega.rpcCreated"
	responseQueueName := "vega-response-queue"
	responseRoutingKey := "vega.response"

	type Response struct {
		Message string `json:"message"`
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

	publisher := connection.NewPublisher()

	counter := 0

	r := gin.Default()

	done := make(chan bunnify.ConsumableEvent[Response])
	r.POST("/rpc/:node", func(c *gin.Context) {

		node := c.Params.ByName("node")
		fmt.Println("Node: " + node)
		// Create a message to publish.
		var rpcRequest map[string]interface{}

		if err := c.ShouldBindJSON(&rpcRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		eventToPublish := bunnify.NewPublishableEvent(rpcRequest)

		err := publisher.Publish(context.TODO(), exchangeName, routingKey, eventToPublish)
		if err != nil {
			fmt.Println(err)
			return
		}

		var rpcResponseEvent bunnify.ConsumableEvent[Response]

		receiveRPCResponse := func(ctx context.Context, event bunnify.ConsumableEvent[Response]) error {
			counter++
			fmt.Println("Entering receiveRPCResponse: " + strconv.Itoa(counter))

			//response := event.Payload
			rpcResponseEvent = event
			fmt.Println("rpcResponseEvent: " + rpcResponseEvent.Payload.Message)
			fmt.Println("event: " + event.Payload.Message)

			done <- rpcResponseEvent
			//fmt.Println("Waiting to send " + response.Message + " to receiveRPCChannel")
			//fmt.Println("Done sending to receiveRPCChannel")
			fmt.Println("Leaving receiveRPCResponse: " + strconv.Itoa(counter))
			return nil
		}
		consumer := connection.NewConsumer(
			responseQueueName,
			bunnify.WithQuorumQueue(),
			bunnify.WithBindingToExchange(exchangeName),
			bunnify.WithHandler(responseRoutingKey, receiveRPCResponse))

		if err := consumer.Consume(); err != nil {
			fmt.Println(err)
		}

		//fmt.Println("Waiting for message from messageReceived")
		rpcResponseEvent = <-done
		fmt.Println("Returning rpcResponseEvent: " + rpcResponseEvent.Payload.Message)
		c.String(http.StatusOK, rpcResponseEvent.Payload.Message)
		//time.Sleep(time.Duration(1) * time.Second)
		//fmt.Println("Received message " + rpcResponseEvent.Payload.Message + " receiveRPCChannel")

	})

	r.Run(":3000") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

// ./sender/main.go

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pmorelli92/bunnify/bunnify"
)

type Response struct {
	Message string `json:"message"`
}

var counter = 0
var rpcResponseChannel = make(chan bunnify.ConsumableEvent[Response])

func main() {

	// Define RabbitMQ server URL.
	amqpServerURL := os.Getenv("AMQP_SERVER_URL")

	// Setup
	exchangeName := "vega-exchange"
	routingKey := "vega.rpcCreated"
	responseQueueName := "vega-response-queue"
	responseRoutingKey := "vega.response"

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

	consumer := connection.NewConsumer(
		responseQueueName,
		bunnify.WithQuorumQueue(),
		bunnify.WithBindingToExchange(exchangeName),
		bunnify.WithHandler(responseRoutingKey, handleRPCResponse))

	if err := consumer.Consume(); err != nil {
		log.Println(err)
	}

	r := gin.Default()

	r.POST("/rpc/:node", func(c *gin.Context) {
		log.Println("Received rpc request")

		node := c.Params.ByName("node")
		log.Println("Node: " + node)
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
		log.Println("Published rpc")

		//fmt.Println("Waiting for message from messageReceived")
		log.Println("Waiting for a response on rpcResponseChannel")
		rpcResponse := <-rpcResponseChannel
		log.Println("Received rpcResponse on channel, sending response: " + rpcResponse.Payload.Message)
		c.String(http.StatusOK, rpcResponse.Payload.Message)
		//time.Sleep(time.Duration(1) * time.Second)
		//fmt.Println("Received message " + rpcResponseEvent.Payload.Message + " receiveRPCChannel")

	})

	r.Run(":3000") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

func handleRPCResponse(ctx context.Context, event bunnify.ConsumableEvent[Response]) error {
	counter++
	log.Println("Entering receiveRPCResponse: " + strconv.Itoa(counter))

	log.Println("Sending response to rpcResposneChannel: " + event.Payload.Message)

	rpcResponseChannel <- event
	//fmt.Println("Waiting to send " + response.Message + " to receiveRPCChannel")
	//fmt.Println("Done sending to receiveRPCChannel")
	log.Println("Leaving receiveRPCResponse: " + strconv.Itoa(counter))
	return nil
}

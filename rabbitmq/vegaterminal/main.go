// ./sender/main.go

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
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

	r := gin.Default()
	r.POST("/rpc/:node", func(c *gin.Context) {
		node := c.Params.ByName("name")

		fmt.Println(node)

		var rpcRequest map[string]interface{}

		if err := c.ShouldBindJSON(&rpcRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Create a message to publish.

		eventToPublish := bunnify.NewPublishableEvent(rpcRequest)

		err := publisher.Publish(context.TODO(), exchangeName, routingKey, eventToPublish)
		if err != nil {
			fmt.Println(err)
			return
		}

		c.String(http.StatusOK, "pong")
	})

	r.Run(":3000") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

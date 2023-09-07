// ./sender/main.go

package main

import (
	"context"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
)

func main() {

	// Define RabbitMQ server URL.
	amqpServerURL := os.Getenv("AMQP_SERVER_URL")

	// Create a new RabbitMQ connection.
	connectRabbitMQ, err := amqp.Dial(amqpServerURL)
	if err != nil {
		panic(err)
	}
	defer connectRabbitMQ.Close()

	// Let's start by opening a channel to our RabbitMQ
	// instance over the connection we have already
	// established.
	channelRabbitMQ, err := connectRabbitMQ.Channel()
	if err != nil {
		panic(err)
	}
	defer channelRabbitMQ.Close()

	// With the instance and declare Queues that we can
	// publish and subscribe to.
	_, err = channelRabbitMQ.QueueDeclare(
		"QueueService1", // queue name
		true,            // durable
		false,           // auto delete
		false,           // exclusive
		false,           // no wait
		nil,             // arguments
	)
	if err != nil {
		panic(err)
	}

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		// Create a message to publish.
		message := amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte("ping"),
		}

		// Attempt to publish a message to the queue.
		if err := channelRabbitMQ.PublishWithContext(
			context.Background(), // ctx
			"",                   // exchange
			"QueueService1",      // queue name
			false,                // mandatory
			false,                // immediate
			message,              // message to publish
		); err != nil {
			c.String(http.StatusInternalServerError, "Failed to publish message: "+err.Error())
			return
		}
		c.String(http.StatusOK, "pong")
	})

	r.Run(":3000") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

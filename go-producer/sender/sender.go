package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

type LogMessage struct {
	ServiceName string `json:"service"`
	Level       string `json:"level"`
	Message     string `json:"message"`
	Timestamp   string `json:"timestamp"`
}

func main() {
	connection, err := amqp.Dial("amqp://guest:guest@localhost:5672/") // For connection
	failOnError(err, "Didn't connect RabbitMQ")
	defer connection.Close() // Must use these keywords

	channel, err := connection.Channel() // For process
	failOnError(err, "Channel Error")
	defer channel.Close() // Must use these keywords

	queue, err := channel.QueueDeclare(
		"logs_queue", // Standart
		true,         // durable
		false,        // delete when unused
		false,        // exclusive
		false,        // no-wait
		nil,          // arguments
	)
	failOnError(err, "Didn't create queue")

	services := []string{"Service-1", "Service-2", "Service-3", "Service-4"}
	levels := []string{"INFO", "WARNING", "ERROR", "CRITICAL"}
	messages := []string{
		"Connection timed out",
		"Disk usage at 99%",
		"User logged in successfully",
		"Memory leak detected in pod",
		"Database deadlock occurred",
	}

	for {
		logData := LogMessage{
			ServiceName: services[rand.Intn(len(services))],
			Level:       levels[rand.Intn(len(levels))],
			Message:     messages[rand.Intn(len(messages))],
			Timestamp:   time.Now().Format(time.RFC3339),
		}

		body, err := json.Marshal(logData)
		if err != nil {
			log.Println("JSON error:", err)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		err = channel.PublishWithContext(ctx,
			"",         // exchange
			queue.Name, // routing key ("logs_queue")
			false,      // mandatory
			false,
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "application/json", // JSON Format
				Body:         body,
			})
		cancel()

		failOnError(err, "Failed to send message")
		log.Printf(" [x] Success: %s", body)

		time.Sleep(2 * time.Second)
	}
}

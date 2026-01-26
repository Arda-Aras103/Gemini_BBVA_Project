package main

import (
	"encoding/json"
	"log"
	"strings" 
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

type IncidentLog struct {
	OriginalLog string `json:"original_log"`
	Analysis    string `json:"analysis"`
	Solution    string `json:"solution"`
}

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {
	
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open channel")
	defer ch.Close()

	
	q, err := ch.QueueDeclare(
		"incidents_queue", // name
		true,              // durable
		false,             // delete when unused
		false,             // exclusive
		false,             // no-wait
		nil,               // arguments
	)
	failOnError(err, "Kuyruk açılamadı")

	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to start the consumer")

	

	forever := make(chan struct{})

	go func() {
		for d := range msgs {
			var incident IncidentLog
			
			err := json.Unmarshal(d.Body, &incident)
			if err != nil {
				log.Printf("JSON Error: %s", err)
				continue
			}
			
			if strings.Contains(incident.Analysis, "HIGH") || strings.Contains(incident.Analysis, "CRITICAL") {
				
				log.Printf("[CRITICAL ALARM]")
				log.Printf("    Log: %s", incident.OriginalLog)
				log.Printf("    Analysis: %s", incident.Analysis)
				log.Printf("    Solution: %s", incident.Solution) 
				log.Println("------------------------------------------------")
				
			} else {
				
				log.Printf("✅ [INFO] %s", incident.Analysis)
			}
			
			time.Sleep(50 * time.Millisecond)
		}
	}()

	<-forever
}
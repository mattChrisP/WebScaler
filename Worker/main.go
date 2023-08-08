package main

import (
	"bytes"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func callFlaskAPI(filePath string) error {
	// Read the file
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var requestBody bytes.Buffer

	multiPartWriter := multipart.NewWriter(&requestBody)

	fileWriter, err := multiPartWriter.CreateFormFile("image", filepath.Base(filePath))
	if err != nil {
		return err
	}

	_, err = io.Copy(fileWriter, file)
	if err != nil {
		return err
	}

	// Close the multipart writer to set the correct boundaries
	multiPartWriter.Close()

	// Create the HTTP request to Flask API
	request, err := http.NewRequest("POST", "http://flask-api:5000/upscale", &requestBody)
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", multiPartWriter.FormDataContentType())

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// Here you may want to handle the response, for example, check if the status code is 200

	// Delete the file
	err = os.Remove(filePath)
	if err != nil {
		return err
	}

	return nil
}

const maxRetries = 5
const retryInterval = 5 * time.Second

func connectToRabbitMQ() *amqp.Connection {
	var conn *amqp.Connection
	var err error

	for i := 0; i < maxRetries; i++ {
		conn, err = amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
		if err == nil {
			return conn
		}

		log.Printf("Failed to connect to RabbitMQ (attempt %d/%d). Retrying in %v...", i+1, maxRetries, retryInterval)
		time.Sleep(retryInterval)
	}

	log.Fatalf("Failed to connect to RabbitMQ after %d attempts.", maxRetries)
	return nil
}

func main() {
	conn := connectToRabbitMQ()
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	q, err := ch.QueueDeclare("task_queue", true, false, false, false, nil)
	failOnError(err, "Failed to declare a queue")

	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			log.Printf("Received a message: %s", d.Body)
			err := callFlaskAPI(string(d.Body))
			if err != nil {
				log.Printf("Failed to process image: %s", err)
				continue
			}
			log.Printf("Successfully processed image")
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

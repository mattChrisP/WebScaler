package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func callFlaskAPI(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var requestBody bytes.Buffer
	multiPartWriter := multipart.NewWriter(&requestBody)

	uniqueID := filepath.Base(filePath)                                         // this gets you the filename
	uniqueID = strings.TrimSuffix(uniqueID, "-uploaded"+filepath.Ext(uniqueID)) // this removes the -uploaded and file extension

	// Add the unique ID as a field
	err = multiPartWriter.WriteField("uniqueId", uniqueID)
	if err != nil {
		return err
	}

	fileWriter, err := multiPartWriter.CreateFormFile("image", filepath.Base(filePath))
	if err != nil {
		return err
	}

	_, err = io.Copy(fileWriter, file)
	if err != nil {
		return err
	}
	multiPartWriter.Close()

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

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 status code: %d", response.StatusCode)
	}

	imageData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	// Construct the file path using the unique ID
	outputPath := filepath.Join("/tmp", fmt.Sprintf("%s-upscaled.png", uniqueID))
	err = ioutil.WriteFile(outputPath, imageData, 0644)
	if err != nil {
		return err
	}

	log.Printf("Upscaled image saved at: %s", outputPath)

	goServerRequest, err := http.NewRequest("POST", "http://go-server:8080/receive-upscaled-image", bytes.NewReader(imageData))
	if err != nil {
		return err
	}
	goServerRequest.Header.Set("X-File-Name", outputPath)

	goServerResponse, err := client.Do(goServerRequest)
	if err != nil {
		return err
	}
	log.Printf("Sent image to Go server, received status code: %d", goServerResponse.StatusCode)
	defer goServerResponse.Body.Close()

	if goServerResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send image to Go server, status code: %d", goServerResponse.StatusCode)
	}

	err = os.Remove(filePath)
	log.Printf("Attempting to delete file at: %s", filePath)
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

	ch.Qos(1, 0, false)

	msgs, err := ch.Consume(q.Name, "", false, false, false, false, nil)
	failOnError(err, "Failed to register a consumer")

	forever := make(chan bool)

	go func() {
		for d := range msgs {
			err := callFlaskAPI(string(d.Body))
			if err != nil {
				log.Printf("Error processing the image: %v", err)
				d.Nack(false, true) // Requeue the message
			} else {
				d.Ack(false)
			}
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	<-forever
}

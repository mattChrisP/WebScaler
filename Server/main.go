package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	http.HandleFunc("/upload", enableCors(uploadFile))
	http.HandleFunc("/receive-upscaled-image", enableCors(receiveUpscaledImage))
	http.HandleFunc("/get-upscaled-image", enableCors(getUpscaledImage))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func enableCors(fn http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setCorsHeaders(w)
		if r.Method == "OPTIONS" {
			return
		}
		fn(w, r)
	}
}

func setCorsHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(10 << 20)

	uniqueId := r.FormValue("id")
	filename := fmt.Sprintf("/tmp/%s-%s.png", uniqueId, "uploaded")
	tempFile, err := os.Create(filename)
	if err != nil {
		http.Error(w, "Failed to create file", http.StatusInternalServerError)
		return
	}
	defer tempFile.Close()

	file, _, err := r.FormFile("image")
	if err != nil {
		http.Error(w, "Error retrieving the file", http.StatusBadRequest)
		return
	}
	defer file.Close()

	_, err = io.Copy(tempFile, file)
	if err != nil {
		http.Error(w, "Error writing the file", http.StatusInternalServerError)
		return
	}

	conn, err := amqp.Dial("amqp://guest:guest@rabbitmq:5672/")
	if err != nil {
		http.Error(w, "Failed to connect to RabbitMQ", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		http.Error(w, "Failed to open a channel", http.StatusInternalServerError)
		return
	}
	defer ch.Close()

	q, err := ch.QueueDeclare("task_queue", true, false, false, false, nil)
	if err != nil {
		http.Error(w, "Failed to declare a queue", http.StatusInternalServerError)
		return
	}

	body := tempFile.Name()
	err = ch.Publish("", q.Name, false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "text/plain",
		Body:         []byte(body),
	})
	if err != nil {
		http.Error(w, "Failed to publish a message", http.StatusInternalServerError)
		return
	}

	_, _ = w.Write([]byte("Successfully uploaded file and task queued"))
}

func receiveUpscaledImage(w http.ResponseWriter, r *http.Request) {
	log.Println("Received a request to /receive-upscaled-image")

	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	filename := r.Header.Get("X-File-Name")
	file, err := os.Create(filename)
	if err != nil {
		http.Error(w, "Failed to save image", http.StatusInternalServerError)
		log.Printf("Error: %v", err)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, r.Body)
	if err != nil {
		http.Error(w, "Failed to save image", http.StatusInternalServerError)
		log.Printf("Error: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Image received and saved successfully."))
}

func getUpscaledImage(w http.ResponseWriter, r *http.Request) {
	uniqueId := r.URL.Query().Get("uniqueId")
	filename := fmt.Sprintf("/tmp/%s-upscaled.png", uniqueId)

	file, err := os.Open(filename)
	if err != nil {
		http.Error(w, "Image not found", http.StatusNotFound)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "image/png")
	io.Copy(w, file)
}

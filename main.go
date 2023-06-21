package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"strings"

	"github.com/go-chi/chi"
	"github.com/streadway/amqp"
)

type UploadResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

type Mission struct {
	ID          int                    `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Sequence    map[string]interface{} `json:"seq"`
}

func main() {
	r := chi.NewRouter()
	port := os.Getenv("PORT")
	rawPort := os.Getenv("LOG_PORT")
	amqpURL := os.Getenv("AMQP_URL")

	// Define the POST route for uploading an image
	r.Post("/upload", UploadImage)

	r.Post("/mission", MissionHanlde)

	r.Post("/confirmation/{flag}", ConfirmationHandle)

	log.Printf("Starting server on :%s", port)
	go http.ListenAndServe(":"+port, r)

	log.Printf("Starting raw TCP server on :%s", rawPort)
	l, err := net.Listen("tcp", ":"+rawPort)
	if err != nil {
		log.Fatalf("Failed to start raw TCP server: %s", err.Error())
	}
	defer l.Close()

	// Connect to RabbitMQ
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %s", err.Error())
	}
	channel, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %s", err.Error())
	}
	defer conn.Close()

	for {
		rawConn, err := l.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %s", err.Error())
			continue
		}
		go handleRawConnection(rawConn, channel)
	}
}

func MissionHanlde(w http.ResponseWriter, r *http.Request) {
	// Get json from the request
	mission := Mission{}
	err := json.NewDecoder(r.Body).Decode(&mission)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Write the json to a file with a file name of the mission ID - name
	fileName := fmt.Sprintf("%s/%d-%s.json", os.Getenv("MISSION_PATH"), mission.ID, mission.Name)
	file, err := os.Create(fileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer file.Close()
	json.NewEncoder(file).Encode(mission)
	// return 200
	res := UploadResponse{
		Message: "success",
		Status:  1,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}
func ConfirmationHandle(w http.ResponseWriter, r *http.Request) {
	flag := chi.URLParam(r, "flag")
	// write flag to a tcp port
	conn, err := net.Dial("tcp", os.Getenv("CONTROLSERIVE_TCP_PORT"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	conn.Write([]byte(flag))
	// return 200
	res := UploadResponse{
		Message: "Success send flag to control service",
		Status:  1,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

func UploadImage(w http.ResponseWriter, r *http.Request) {
	// Get the file from the request
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Parse the file name to get the mission ID. Filename looks like: 1-something.jpg
	missionID := strings.Split(header.Filename, "-")[0]
	path := fmt.Sprintf("%s/upload/%s", os.Getenv("BACKEND_URL"), missionID)

	// Create a new multipart writer
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create a new part for the file
	part, err := writer.CreateFormFile("image", header.Filename)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy the file contents to the part
	_, err = io.Copy(part, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Close the multipart writer
	err = writer.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a new request with the multipart body
	req, err := http.NewRequest("POST", path, body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// Forward the request to the backend
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Failed to upload image, make sure the GCS is online", resp.StatusCode)
		return
	}

	res := UploadResponse{
		Message: "success",
		Status:  1,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
}

func handleRawConnection(conn net.Conn, channel *amqp.Channel) {
	defer conn.Close()

	// Handle the raw TCP connection here
	// You can read from and write to the connection as needed

	// Example: Read from the connection
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("Error reading from raw TCP connection: %s", err.Error())
		return
	}
	data := buffer[:n]

	q, err := channel.QueueDeclare(
		"log", // name
		false, // durable
		false, // delete when unused
		false, // exclusive
		false, // no-wait
		nil,   // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare a queue: %s", err.Error())
	}
	// Publish data to RabbitMQ
	err = channel.Publish(
		"",
		q.Name,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
		},
	)
	if err != nil {
		log.Printf("Error publishing message to RabbitMQ: %s", err.Error())
		return
	}

	fmt.Printf("Published data to RabbitMQ: %s", string(data))
}

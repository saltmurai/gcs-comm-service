package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi"
)

type UploadResponse struct {
	Message string `json:"message"`
	Status  int    `json:"status"`
}

func main() {
	r := chi.NewRouter()

	// Define the POST route for uploading an image
	r.Post("/upload", UploadImage)

	log.Println("Server started on :5000")
	http.ListenAndServe(":5000", r)
}

func UploadImage(w http.ResponseWriter, r *http.Request) {
	// Get the file from the request
	file, header, err := r.FormFile("image")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()
	fmt.Println(header.Filename)
	// return 200
	res := UploadResponse{
		Message: "success",
		Status:  1,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(res)
	// Create a new HTTP request to the backend
	// backendURL := "http://backend-server:8081/upload" // Replace with your backend server URL
	// backendRequest, err := http.NewRequest(http.MethodPost, backendURL, file)
	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// backendRequest.Header.Set("Content-Type", header.Header.Get("Content-Type"))

	// // Proxy the request to the backend
	// proxy := httputil.NewSingleHostReverseProxy(&url.URL{
	// 	Scheme: "http",
	// 	Host:   "backend-server:8081", // Replace with your backend server host and port
	// })
	// proxy.ServeHTTP(w, backendRequest)
}

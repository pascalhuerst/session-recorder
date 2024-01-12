package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	presignedURL := flag.String("url", "", "Presigned URL")
	flag.Parse()

	// Replace with the file or data you want to upload
	fileContent := []byte("Hello, MinIO!")

	// Create a reader for the file content
	fileReader := bytes.NewReader(fileContent)

	// Create HTTP PUT request
	req, err := http.NewRequest(http.MethodPut, *presignedURL, fileReader)
	if err != nil {
		log.Fatalln(err)
	}
	req.ContentLength = int64(len(fileContent))

	// Set content type if required
	req.Header.Set("Content-Type", "application/octet-stream")

	// Perform the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Unexpected status: %v\n", resp.Status)
	}

	fmt.Println("File uploaded successfully!")
}

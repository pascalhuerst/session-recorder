package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"
	"github.com/pascalhuerst/session-recorder/render"
)

func main() {
	// Replace the URL with your actual presigned URL
	url := "http://127.0.0.1:9000/session-recorder/22a2a258-bc03-4836-a94a-d322d5e9aaaa/sessions/0f6e7eee-d4fe-4975-9641-29a33dd1a4d9/data.raw?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=rqn5q9WNvH9n5XOrVRrq%2F20240111%2Fus-east-1%2Fs3%2Faws4_request&X-Amz-Date=20240111T232358Z&X-Amz-Expires=3600&X-Amz-SignedHeaders=host&X-Amz-Signature=8609f5a4e2427dd7712ccff56588441e923009dbad8023f7f20bccfc9ae1a571"

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error making HTTP request:", err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		fmt.Println("Error: Unexpected status code", response.StatusCode)
		return
	}

	outFile, err := os.Create("data.flac")
	if err != nil {
		log.Fatal().Err(err).Msg("Cannot create output file")
	}
	defer outFile.Close()

	if err := render.FlacStream(response.Body, outFile); err != nil {
		log.Fatal().Err(err).Msg("Cannot convert raw to flac")
	}
}

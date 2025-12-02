package main

import (
	"context"
	"log"
	"time"
)

// Background worker that processes pending submissions
func startWorker(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := processSubmissions(); err != nil {
				log.Printf("Worker error: %v", err)
			}
		}
	}
}

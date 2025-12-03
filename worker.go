package main

import (
	"context"
	"log"
	"time"
)

// Background worker that processes pending submissions and bracket matches
func startWorker(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Process immediately on start
	if err := processSubmissions(); err != nil {
		log.Printf("Worker error (submissions): %v", err)
	}
	if err := processBracketMatches(); err != nil {
		log.Printf("Worker error (bracket): %v", err)
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := processSubmissions(); err != nil {
				log.Printf("Worker error (submissions): %v", err)
			}
			if err := processBracketMatches(); err != nil {
				log.Printf("Worker error (bracket): %v", err)
			}
		}
	}
}

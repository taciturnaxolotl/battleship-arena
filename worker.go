package main

import (
	"context"
	"log"
	"sync"
	"time"
)

var workerMutex sync.Mutex

// Background worker that processes pending submissions and bracket matches
func startWorker(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// Process immediately on start
	go func() {
		if err := processSubmissionsWithLock(); err != nil {
			log.Printf("Worker error (submissions): %v", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			go func() {
				if err := processSubmissionsWithLock(); err != nil {
					log.Printf("Worker error (submissions): %v", err)
				}
			}()
		}
	}
}

func processSubmissionsWithLock() error {
	// Try to acquire lock, return immediately if already processing
	if !workerMutex.TryLock() {
		log.Printf("Worker already running, skipping this cycle")
		return nil
	}
	defer workerMutex.Unlock()
	
	return processSubmissions()
}

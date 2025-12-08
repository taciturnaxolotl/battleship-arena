package runner

import (
	"context"
	"log"
	"sync"
	"time"

	"battleship-arena/internal/storage"
)

var workerMutex sync.Mutex

func StartWorker(ctx context.Context, uploadDir string, broadcastFunc func(string, int, int, time.Time, []string), notifyFunc func(), completeFunc func(), statusFunc func(string, string, string)) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	go func() {
		if err := processSubmissionsWithLock(uploadDir, broadcastFunc, notifyFunc, completeFunc, statusFunc); err != nil {
			log.Printf("Worker error (submissions): %v", err)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			go func() {
				if err := processSubmissionsWithLock(uploadDir, broadcastFunc, notifyFunc, completeFunc, statusFunc); err != nil {
					log.Printf("Worker error (submissions): %v", err)
				}
			}()
		}
	}
}

func processSubmissionsWithLock(uploadDir string, broadcastFunc func(string, int, int, time.Time, []string), notifyFunc func(), completeFunc func(), statusFunc func(string, string, string)) error {
	if !workerMutex.TryLock() {
		// Silently skip if worker is already running
		return nil
	}
	defer workerMutex.Unlock()
	
	return ProcessSubmissions(uploadDir, broadcastFunc, notifyFunc, completeFunc, statusFunc)
}

func ProcessSubmissions(uploadDir string, broadcastFunc func(string, int, int, time.Time, []string), notifyFunc func(), completeFunc func(), statusFunc func(string, string, string)) error {
	submissions, err := storage.GetPendingSubmissions()
	if err != nil {
		return err
	}
	
	// Only do work if there are pending submissions
	if len(submissions) == 0 {
		return nil
	}

	for _, sub := range submissions {
		log.Printf("⚙️  Compiling %s (%s)", sub.Username, sub.Filename)
		statusFunc(sub.Username, "compiling", "")
		
		if err := CompileSubmission(sub, uploadDir); err != nil {
			log.Printf("❌ Compilation failed for %s: %v", sub.Username, err)
			storage.UpdateSubmissionStatusWithMessage(sub.ID, "compilation_failed", err.Error())
			statusFunc(sub.Username, "compilation_failed", err.Error())
			notifyFunc()
			continue
		}
		
		log.Printf("✓ Compiled %s", sub.Username)
		storage.UpdateSubmissionStatus(sub.ID, "completed")
		statusFunc(sub.Username, "running_matches", "")
		
		RunRoundRobinMatches(sub, uploadDir, broadcastFunc)
		statusFunc(sub.Username, "completed", "")
		notifyFunc()
	}
	
	// Check if queue is now empty
	queuedPlayers := storage.GetQueuedPlayerNames()
	if len(queuedPlayers) == 0 {
		completeFunc()
	}

	return nil
}

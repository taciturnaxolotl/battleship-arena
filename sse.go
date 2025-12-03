package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/tmaxmax/go-sse"
)

var sseServer *sse.Server

func initSSE() {
	sseServer = &sse.Server{}
	log.Printf("SSE server initialized (tmaxmax/go-sse)")
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	log.Printf("SSE client connected from %s", r.RemoteAddr)
	sseServer.ServeHTTP(w, r)
}

// NotifyLeaderboardUpdate sends updated leaderboard to all connected clients
func NotifyLeaderboardUpdate() {
	entries, err := getLeaderboard(50)
	if err != nil {
		log.Printf("Failed to get leaderboard for SSE: %v", err)
		return
	}

	data, err := json.Marshal(entries)
	if err != nil {
		log.Printf("Failed to marshal leaderboard for SSE: %v", err)
		return
	}

	msg := &sse.Message{}
	msg.AppendData(string(data))

	// Publish to default topic
	log.Printf("Publishing to SSE clients (%d bytes)", len(data))
	if err := sseServer.Publish(msg); err != nil {
		log.Printf("Failed to publish SSE message: %v", err)
		return
	}

	log.Printf("Broadcast leaderboard update to SSE clients (%d bytes)", len(data))
}

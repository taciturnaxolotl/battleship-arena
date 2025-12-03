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
}

func handleSSE(w http.ResponseWriter, r *http.Request) {
	sseServer.ServeHTTP(w, r)
}

// NotifyLeaderboardUpdate sends updated leaderboard to all connected clients
func NotifyLeaderboardUpdate() {
	entries, err := getLeaderboard(50)
	if err != nil {
		log.Printf("SSE: failed to get leaderboard: %v", err)
		return
	}

	data, err := json.Marshal(entries)
	if err != nil {
		log.Printf("SSE: failed to marshal leaderboard: %v", err)
		return
	}

	msg := &sse.Message{}
	msg.AppendData(string(data))

	if err := sseServer.Publish(msg); err != nil {
		log.Printf("SSE: publish failed: %v", err)
	}
}

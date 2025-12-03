package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/tmaxmax/go-sse"
)

var sseServer *sse.Server

type ProgressUpdate struct {
	Type              string   `json:"type"` // "progress" or "complete"
	Player            string   `json:"player,omitempty"`
	Opponent          string   `json:"opponent,omitempty"`
	CurrentMatch      int      `json:"current_match,omitempty"`
	TotalMatches      int      `json:"total_matches,omitempty"`
	EstimatedTimeLeft string   `json:"estimated_time_left,omitempty"`
	PercentComplete   float64  `json:"percent_complete,omitempty"`
	QueuedPlayers     []string `json:"queued_players,omitempty"`
}

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

func broadcastProgress(player string, currentMatch, totalMatches int, startTime time.Time, queuedPlayers []string) {
	elapsed := time.Since(startTime)
	avgTimePerMatch := elapsed / time.Duration(currentMatch)
	remainingMatches := totalMatches - currentMatch
	estimatedTimeLeft := avgTimePerMatch * time.Duration(remainingMatches)
	
	percentComplete := float64(currentMatch) / float64(totalMatches) * 100.0
	
	// Format time left
	timeLeftStr := formatDuration(estimatedTimeLeft)
	
	// Filter out current player from queue (they're being shown in progress, not queue)
	filteredQueue := make([]string, 0)
	for _, p := range queuedPlayers {
		if p != player {
			filteredQueue = append(filteredQueue, p)
		}
	}
	
	progress := ProgressUpdate{
		Type:              "progress",
		Player:            player,
		CurrentMatch:      currentMatch,
		TotalMatches:      totalMatches,
		EstimatedTimeLeft: timeLeftStr,
		PercentComplete:   percentComplete,
		QueuedPlayers:     filteredQueue,
	}
	
	data, err := json.Marshal(progress)
	if err != nil {
		log.Printf("Failed to marshal progress: %v", err)
		return
	}
	
	log.Printf("Broadcasting progress: %s [%d/%d] %.1f%% (queue: %d)", player, currentMatch, totalMatches, percentComplete, len(filteredQueue))
	
	msg := &sse.Message{}
	msg.AppendData(string(data))
	// Don't set Type - just send as regular message
	
	if err := sseServer.Publish(msg); err != nil {
		log.Printf("SSE: progress publish failed: %v", err)
	}
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return "< 1 min"
	}
	minutes := int(d.Minutes())
	if minutes < 60 {
		return fmt.Sprintf("%d min", minutes)
	}
	hours := minutes / 60
	mins := minutes % 60
	if mins > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	return fmt.Sprintf("%dh", hours)
}

func broadcastProgressComplete() {
	complete := ProgressUpdate{
		Type: "complete",
	}
	
	data, err := json.Marshal(complete)
	if err != nil {
		return
	}
	
	log.Printf("Broadcasting progress complete")
	
	msg := &sse.Message{}
	msg.AppendData(string(data))
	// Don't set Type - just send as regular message
	
	sseServer.Publish(msg)
}

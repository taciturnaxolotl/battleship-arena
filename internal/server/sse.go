package server

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/alexandrevicenzi/go-sse"
	
	"battleship-arena/internal/storage"
)

var SSEServer *sse.Server

type ProgressUpdate struct {
	Type              string   `json:"type"`
	Player            string   `json:"player,omitempty"`
	Opponent          string   `json:"opponent,omitempty"`
	CurrentMatch      int      `json:"current_match,omitempty"`
	TotalMatches      int      `json:"total_matches,omitempty"`
	EstimatedTimeLeft string   `json:"estimated_time_left,omitempty"`
	PercentComplete   float64  `json:"percent_complete,omitempty"`
	QueuedPlayers     []string `json:"queued_players,omitempty"`
}

func InitSSE() {
	// Disable verbose SSE library logging
	SSEServer = sse.NewServer(nil)
}

func NotifyLeaderboardUpdate() {
	entries, err := storage.GetLeaderboard(50)
	if err != nil {
		log.Printf("SSE: failed to get leaderboard: %v", err)
		return
	}

	data, err := json.Marshal(entries)
	if err != nil {
		log.Printf("SSE: failed to marshal leaderboard: %v", err)
		return
	}

	SSEServer.SendMessage("/events/updates", sse.SimpleMessage(string(data)))
}

func BroadcastProgress(player string, currentMatch, totalMatches int, startTime time.Time, queuedPlayers []string) {
	elapsed := time.Since(startTime)
	avgTimePerMatch := elapsed / time.Duration(currentMatch)
	remainingMatches := totalMatches - currentMatch
	estimatedTimeLeft := avgTimePerMatch * time.Duration(remainingMatches)
	
	percentComplete := float64(currentMatch) / float64(totalMatches) * 100.0
	timeLeftStr := formatDuration(estimatedTimeLeft)
	
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
	
	// Only log every 10th match to reduce noise
	if currentMatch%10 == 0 || currentMatch == totalMatches {
		log.Printf("Progress: %s [%d/%d] %.0f%%", player, currentMatch, totalMatches, percentComplete)
	}
	
	SSEServer.SendMessage("/events/updates", sse.SimpleMessage(string(data)))
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

func BroadcastProgressComplete() {
	complete := ProgressUpdate{
		Type: "complete",
	}
	
	data, err := json.Marshal(complete)
	if err != nil {
		return
	}
	
	// Silent - no log needed for routine completion
	SSEServer.SendMessage("/events/updates", sse.SimpleMessage(string(data)))
}

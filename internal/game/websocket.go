package game

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for now
	},
}

// GameManager manages active games
type GameManager struct {
	games     map[string]*Game
	gameConns map[string]*websocket.Conn // Track which conn owns which game
	mu        sync.RWMutex
}

var Manager = &GameManager{
	games:     make(map[string]*Game),
	gameConns: make(map[string]*websocket.Conn),
}

// CleanupOrphanedGames removes games that have no active connection
func (m *GameManager) CleanupOrphanedGames() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for gameID, game := range m.games {
		if _, exists := m.gameConns[gameID]; !exists {
			log.Printf("Cleaning up orphaned game: %s", gameID)
			game.Close()
			delete(m.games, gameID)
		}
	}
}

// StartCleanupWorker starts a background worker to clean up orphaned games
func (m *GameManager) StartCleanupWorker() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		
		for range ticker.C {
			m.CleanupOrphanedGames()
		}
	}()
}

// Message types for WebSocket communication
type WSMessageIn struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type WSMessageOut struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

type StartGamePayload struct {
	AIName string `json:"aiName"`
}

type MovePayload struct {
	Move string `json:"move"`
}

type ErrorPayload struct {
	Error string `json:"error"`
}

// HandleGameWebSocket handles WebSocket connections for gameplay
func HandleGameWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	var currentGame *Game
	connID := time.Now().UnixNano()
	
	log.Printf("WebSocket connected: %d", connID)

	// Cleanup function to ensure game is properly closed
	cleanupCurrentGame := func() {
		if currentGame != nil {
			log.Printf("Cleaning up game %s for connection %d", currentGame.ID, connID)
			Manager.mu.Lock()
			delete(Manager.games, currentGame.ID)
			delete(Manager.gameConns, currentGame.ID)
			Manager.mu.Unlock()
			currentGame.Close()
			currentGame = nil
		}
	}
	
	defer func() {
		log.Printf("WebSocket disconnected: %d", connID)
		cleanupCurrentGame()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error (conn %d): %v", connID, err)
			}
			break
		}

		var msg WSMessageIn
		if err := json.Unmarshal(message, &msg); err != nil {
			sendError(conn, "Invalid message format")
			continue
		}

		switch msg.Type {
		case "start_game":
			var payload StartGamePayload
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				sendError(conn, "Invalid start_game payload")
				continue
			}

			// Clean up previous game for this connection
			cleanupCurrentGame()

			// Create new game
			game, err := NewGame("Player", payload.AIName)
			if err != nil {
				sendError(conn, err.Error())
				continue
			}

			currentGame = game
			Manager.mu.Lock()
			Manager.games[game.ID] = game
			Manager.gameConns[game.ID] = conn
			Manager.mu.Unlock()
			
			log.Printf("Started game %s for connection %d (AI: %s)", game.ID, connID, payload.AIName)

			// Send initial state
			sendGameState(conn, game)

		case "move":
			if currentGame == nil {
				sendError(conn, "No active game")
				continue
			}

			var payload MovePayload
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				sendError(conn, "Invalid move payload")
				continue
			}

			// Process human move
			result, err := currentGame.HumanMove(payload.Move)
			if err != nil {
				sendError(conn, err.Error())
				continue
			}

			// Send human move result
			sendMoveResult(conn, "human_move_result", result)

			// Only let AI move if game is still ongoing
			// Check both GameOver flag and that it's actually AI's turn
			if !currentGame.GameOver && currentGame.CurrentTurn == "ai" && currentGame.Winner == "" {
				aiResult, err := currentGame.AIMove()
				if err != nil {
					sendError(conn, "AI error: "+err.Error())
					continue
				}
				sendMoveResult(conn, "ai_move_result", aiResult)
			}

			// Send updated state
			sendGameState(conn, currentGame)
			
			// Clean up game if it's over to free AI process immediately
			if currentGame.GameOver {
				log.Printf("Game %s finished (winner: %s), cleaning up", currentGame.ID, currentGame.Winner)
				cleanupCurrentGame()
			}

		case "get_state":
			if currentGame == nil {
				sendError(conn, "No active game")
				continue
			}
			sendGameState(conn, currentGame)

		case "quit":
			cleanupCurrentGame()
			sendJSON(conn, WSMessageOut{Type: "game_ended"})

		default:
			sendError(conn, "Unknown message type: "+msg.Type)
		}
	}
}

func sendJSON(conn *websocket.Conn, v interface{}) {
	data, err := json.Marshal(v)
	if err != nil {
		log.Printf("JSON marshal error: %v", err)
		return
	}
	if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("WebSocket write error: %v", err)
	}
}

func sendError(conn *websocket.Conn, errMsg string) {
	sendJSON(conn, WSMessageOut{
		Type:    "error",
		Payload: ErrorPayload{Error: errMsg},
	})
}

func sendGameState(conn *websocket.Conn, game *Game) {
	state := game.GetState()
	sendJSON(conn, WSMessageOut{
		Type:    "game_state",
		Payload: state,
	})
}

func sendMoveResult(conn *websocket.Conn, msgType string, result *MoveResult) {
	sendJSON(conn, WSMessageOut{
		Type:    msgType,
		Payload: result,
	})
}

// GetAvailableAIs returns list of AI submissions that can be played against
func GetAvailableAIs() ([]string, error) {
	// This would query the storage for active submissions
	// For now, return a placeholder
	return []string{}, nil
}

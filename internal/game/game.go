// Package game provides human vs AI battleship game functionality
// using the isolated player process architecture
package game

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	BoardSize = 10
	
	// Cell states
	CellEmpty   = ' '
	CellShip    = 'S'
	CellHit     = 'X'
	CellMiss    = 'O'
	CellSunk    = '#'
	
	// Result codes (matching C++ kasbs.h)
	ResultMiss = 0
	ResultHit  = 8
	ResultSunk = 16
	ResultShip = 7
	
	// Ship types (matching C++ kasbs.h)
	ShipAC = 1
	ShipBS = 2
	ShipCR = 3
	ShipSB = 4
	ShipDS = 5
)

// Ship represents a battleship
type Ship struct {
	Name     string
	Size     int
	ShipType int  // Ship number (AC=1, BS=2, etc.)
	Marker   byte
	Hits     int
	Cells    [][2]int // [row, col] pairs
}

// Board represents a player's game board
type Board struct {
	Grid  [BoardSize][BoardSize]byte
	Ships []Ship
}

// Game represents a human vs AI battleship game
type Game struct {
	ID           string
	HumanBoard   Board
	AIBoard      Board
	PlayerName   string
	AIName       string
	CurrentTurn  string // "human" or "ai"
	GameOver     bool
	Winner       string
	MoveCount    int
	
	aiProcess    *AIProcess
	mu           sync.Mutex
}

// AIProcess manages communication with an AI player process
type AIProcess struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	alive  bool
}

var enginePath = getEnginePath()

func getEnginePath() string {
	if path := os.Getenv("BATTLESHIP_ENGINE_PATH"); path != "" {
		return path
	}
	return "./battleship-engine"
}

// compileAI compiles an AI submission if the source exists
func compileAI(prefix string) error {
	srcDir := filepath.Join(enginePath, "src")
	buildDir := filepath.Join(enginePath, "build")
	
	srcFile := filepath.Join(srcDir, fmt.Sprintf("memory_functions_%s.cpp", prefix))
	headerFile := filepath.Join(srcDir, fmt.Sprintf("memory_functions_%s.h", prefix))
	
	// Check if source exists
	if _, err := os.Stat(srcFile); os.IsNotExist(err) {
		return fmt.Errorf("source file not found: %s", srcFile)
	}
	
	// Parse function suffix from source
	content, err := os.ReadFile(srcFile)
	if err != nil {
		return err
	}
	
	functionSuffix, err := parseFunctionNames(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse function names: %v", err)
	}
	
	// Generate header if missing
	if _, err := os.Stat(headerFile); os.IsNotExist(err) {
		headerContent := generateHeader(fmt.Sprintf("memory_functions_%s.h", prefix), functionSuffix)
		if err := os.WriteFile(headerFile, []byte(headerContent), 0644); err != nil {
			return err
		}
	}
	
	// Compile player binary
	playerBinary := filepath.Join(buildDir, "ai_"+prefix)
	
	compileArgs := []string{
		"g++", "-std=c++11", "-O3",
		"-I", srcDir,
		fmt.Sprintf("-DPLAYER_SUFFIX=%s", functionSuffix),
		fmt.Sprintf(`-DPLAYER_HEADER="memory_functions_%s.h"`, prefix),
		"-o", playerBinary,
		filepath.Join(srcDir, "player_wrapper.cpp"),
		filepath.Join(srcDir, "battleship.cpp"),
		srcFile,
	}
	
	log.Printf("Compiling AI binary: %s", prefix)
	
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	
	cmd := exec.CommandContext(ctx, compileArgs[0], compileArgs[1:]...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compilation failed: %s", output)
	}
	
	log.Printf("AI binary compiled: %s", playerBinary)
	return nil
}

func parseFunctionNames(cppContent string) (string, error) {
	re := regexp.MustCompile(`void\s+initMemory(\w+)\s*\(`)
	matches := re.FindStringSubmatch(cppContent)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not find initMemory function")
	}
	return matches[1], nil
}

func generateHeader(filename, prefix string) string {
	guard := strings.ToUpper(strings.Replace(filename, ".", "_", -1))
	
	return fmt.Sprintf(`#ifndef %s
#define %s

#include "memory.h"
#include "battleship.h"
#include <string>

void initMemory%s(ComputerMemory &memory);
std::string smartMove%s(const ComputerMemory &memory);
void updateMemory%s(int row, int col, int result, ComputerMemory &memory);

#endif
`, guard, guard, prefix, prefix, prefix)
}

// NewGame creates a new human vs AI game
func NewGame(playerName, aiSubmissionPrefix string) (*Game, error) {
	game := &Game{
		ID:          generateGameID(),
		PlayerName:  playerName,
		AIName:      aiSubmissionPrefix,
		CurrentTurn: "human",
	}
	
	// Initialize boards with random ship placement
	initializeBoard(&game.HumanBoard)
	initializeBoard(&game.AIBoard)
	
	// Start AI process (compile if needed)
	aiPath := filepath.Join(enginePath, "build", "ai_"+aiSubmissionPrefix)
	if _, err := os.Stat(aiPath); os.IsNotExist(err) {
		// Try to compile the AI
		log.Printf("AI binary not found, attempting to compile: %s", aiSubmissionPrefix)
		if err := compileAI(aiSubmissionPrefix); err != nil {
			return nil, fmt.Errorf("failed to compile AI %s: %v", aiSubmissionPrefix, err)
		}
	}
	
	aiProc, err := startAIProcess(aiPath)
	if err != nil {
		return nil, fmt.Errorf("failed to start AI: %v", err)
	}
	game.aiProcess = aiProc
	
	// Initialize AI
	if err := game.initAI(); err != nil {
		game.Close()
		return nil, fmt.Errorf("failed to initialize AI: %v", err)
	}
	
	return game, nil
}

func generateGameID() string {
	return fmt.Sprintf("game-%d", time.Now().UnixNano())
}

func startAIProcess(binaryPath string) (*AIProcess, error) {
	cmd := exec.Command(binaryPath)
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, err
	}
	
	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, err
	}
	
	return &AIProcess{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
		alive:  true,
	}, nil
}

func (p *AIProcess) sendLine(line string) error {
	if !p.alive {
		return fmt.Errorf("AI process not alive")
	}
	_, err := fmt.Fprintln(p.stdin, line)
	return err
}

func (p *AIProcess) readLine() (string, error) {
	if !p.alive {
		return "", fmt.Errorf("AI process not alive")
	}
	line, err := p.stdout.ReadString('\n')
	if err != nil {
		p.alive = false
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (p *AIProcess) close() {
	if p.stdin != nil {
		p.sendLine("QUIT")
		p.stdin.Close()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		p.cmd.Process.Kill()
		p.cmd.Wait()
	}
	p.alive = false
}

func (g *Game) initAI() error {
	// Handshake
	if err := g.aiProcess.sendLine("HELLO 1"); err != nil {
		return err
	}
	
	resp, err := g.aiProcess.readLine()
	if err != nil {
		return err
	}
	if resp != "HELLO OK" {
		return fmt.Errorf("bad handshake response: %s", resp)
	}
	
	// Init for game
	if err := g.aiProcess.sendLine("INIT"); err != nil {
		return err
	}
	
	resp, err = g.aiProcess.readLine()
	if err != nil {
		return err
	}
	if resp != "OK" {
		return fmt.Errorf("bad init response: %s", resp)
	}
	
	return nil
}

// Close cleans up game resources
func (g *Game) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if g.aiProcess != nil {
		g.aiProcess.close()
		g.aiProcess = nil
	}
}

// HumanMove processes a human player's move
func (g *Game) HumanMove(move string) (*MoveResult, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if g.GameOver {
		return nil, fmt.Errorf("game is over")
	}
	
	if g.CurrentTurn != "human" {
		return nil, fmt.Errorf("not your turn")
	}
	
	// Parse move (e.g., "A5" -> row=0, col=4)
	row, col, err := parseMove(move)
	if err != nil {
		return nil, err
	}
	
	// Check if already targeted
	if g.AIBoard.Grid[row][col] == CellHit || g.AIBoard.Grid[row][col] == CellMiss || g.AIBoard.Grid[row][col] == CellSunk {
		return nil, fmt.Errorf("cell already targeted")
	}
	
	// Execute move on AI's board
	result := g.executeMove(&g.AIBoard, row, col)
	g.MoveCount++
	
	// Check for win
	if g.checkAllSunk(&g.AIBoard) {
		g.GameOver = true
		g.Winner = "human"
	} else {
		g.CurrentTurn = "ai"
	}
	
	return result, nil
}

// AIMove gets and executes the AI's move
func (g *Game) AIMove() (*MoveResult, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if g.GameOver {
		return nil, fmt.Errorf("game is over")
	}
	
	if g.CurrentTurn != "ai" {
		return nil, fmt.Errorf("not AI's turn")
	}
	
	// Get move from AI
	if err := g.aiProcess.sendLine("GET_MOVE"); err != nil {
		return nil, fmt.Errorf("failed to request AI move: %v", err)
	}
	
	resp, err := g.aiProcess.readLine()
	if err != nil {
		return nil, fmt.Errorf("failed to read AI move: %v", err)
	}
	
	if !strings.HasPrefix(resp, "MOVE ") {
		return nil, fmt.Errorf("invalid AI response: %s", resp)
	}
	
	move := strings.TrimPrefix(resp, "MOVE ")
	row, col, err := parseMove(move)
	if err != nil {
		// AI gave invalid move, use random
		row, col = g.randomValidMove(&g.HumanBoard)
	}
	
	// Check if already targeted, use random if so
	for g.HumanBoard.Grid[row][col] == CellHit || g.HumanBoard.Grid[row][col] == CellMiss || g.HumanBoard.Grid[row][col] == CellSunk {
		row, col = g.randomValidMove(&g.HumanBoard)
	}
	
	// Execute move on human's board
	result := g.executeMove(&g.HumanBoard, row, col)
	result.Move = formatMove(row, col)
	g.MoveCount++
	
	// Update AI with result
	updateCmd := fmt.Sprintf("UPDATE %d %d %d", row, col, result.ResultCode)
	if err := g.aiProcess.sendLine(updateCmd); err != nil {
		return nil, fmt.Errorf("failed to update AI: %v", err)
	}
	
	resp, err = g.aiProcess.readLine()
	if err != nil {
		return nil, fmt.Errorf("failed to read AI update response: %v", err)
	}
	
	// Check for win
	if g.checkAllSunk(&g.HumanBoard) {
		g.GameOver = true
		// Only set winner if not already set (human won on their last turn)
		if g.Winner == "" {
			g.Winner = "ai"
		}
	} else {
		g.CurrentTurn = "human"
	}
	
	return result, nil
}

func (g *Game) randomValidMove(board *Board) (int, int) {
	for {
		row := rand.Intn(BoardSize)
		col := rand.Intn(BoardSize)
		cell := board.Grid[row][col]
		if cell != CellHit && cell != CellMiss && cell != CellSunk {
			return row, col
		}
	}
}

// MoveResult represents the result of a move
type MoveResult struct {
	Move       string `json:"move"`
	Row        int    `json:"row"`
	Col        int    `json:"col"`
	Hit        bool   `json:"hit"`
	Sunk       bool   `json:"sunk"`
	ShipName   string `json:"shipName,omitempty"`
	ResultCode int    `json:"resultCode"`
}

func (g *Game) executeMove(board *Board, row, col int) *MoveResult {
	result := &MoveResult{
		Move: formatMove(row, col),
		Row:  row,
		Col:  col,
	}
	
	cell := board.Grid[row][col]
	
	// Check if it's a ship
	for i := range board.Ships {
		ship := &board.Ships[i]
		for _, pos := range ship.Cells {
			if pos[0] == row && pos[1] == col {
				// Hit!
				ship.Hits++
				result.Hit = true
				
				if ship.Hits >= ship.Size {
					// Sunk!
					result.Sunk = true
					result.ShipName = ship.Name
					result.ResultCode = ResultSunk | ship.ShipType
					// Mark all cells as sunk
					for _, sunkPos := range ship.Cells {
						board.Grid[sunkPos[0]][sunkPos[1]] = CellSunk
					}
				} else {
					board.Grid[row][col] = CellHit
					result.ResultCode = ResultHit | ship.ShipType
				}
				return result
			}
		}
	}
	
	// Miss
	if cell == CellEmpty {
		board.Grid[row][col] = CellMiss
	}
	result.Hit = false
	result.ResultCode = ResultMiss
	return result
}

func (g *Game) checkAllSunk(board *Board) bool {
	for _, ship := range board.Ships {
		if ship.Hits < ship.Size {
			return false
		}
	}
	return true
}

// GetState returns the current game state for the UI
func (g *Game) GetState() *GameState {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	state := &GameState{
		GameID:      g.ID,
		PlayerName:  g.PlayerName,
		AIName:      g.AIName,
		CurrentTurn: g.CurrentTurn,
		GameOver:    g.GameOver,
		Winner:      g.Winner,
		MoveCount:   g.MoveCount,
	}
	
	// Human's view of AI board (hide ships)
	for row := 0; row < BoardSize; row++ {
		for col := 0; col < BoardSize; col++ {
			cell := g.AIBoard.Grid[row][col]
			switch cell {
			case CellHit, CellMiss, CellSunk:
				state.EnemyBoard[row][col] = cell
			default:
				state.EnemyBoard[row][col] = CellEmpty
			}
		}
	}
	
	// Human's own board (show ships)
	for row := 0; row < BoardSize; row++ {
		for col := 0; col < BoardSize; col++ {
			state.OwnBoard[row][col] = g.HumanBoard.Grid[row][col]
		}
	}
	
	// Ship status
	for _, ship := range g.HumanBoard.Ships {
		state.OwnShips = append(state.OwnShips, ShipStatus{
			Name:  ship.Name,
			Size:  ship.Size,
			Hits:  ship.Hits,
			Sunk:  ship.Hits >= ship.Size,
		})
	}
	
	for _, ship := range g.AIBoard.Ships {
		state.EnemyShips = append(state.EnemyShips, ShipStatus{
			Name:  ship.Name,
			Size:  ship.Size,
			Hits:  ship.Hits,
			Sunk:  ship.Hits >= ship.Size,
		})
	}
	
	return state
}

// GameState represents the game state for JSON serialization
type GameState struct {
	GameID      string                      `json:"gameId"`
	PlayerName  string                      `json:"playerName"`
	AIName      string                      `json:"aiName"`
	CurrentTurn string                      `json:"currentTurn"`
	GameOver    bool                        `json:"gameOver"`
	Winner      string                      `json:"winner,omitempty"`
	MoveCount   int                         `json:"moveCount"`
	OwnBoard    [BoardSize][BoardSize]byte  `json:"ownBoard"`
	EnemyBoard  [BoardSize][BoardSize]byte  `json:"enemyBoard"`
	OwnShips    []ShipStatus                `json:"ownShips"`
	EnemyShips  []ShipStatus                `json:"enemyShips"`
}

type ShipStatus struct {
	Name string `json:"name"`
	Size int    `json:"size"`
	Hits int    `json:"hits"`
	Sunk bool   `json:"sunk"`
}

// Helper functions

func parseMove(move string) (int, int, error) {
	move = strings.ToUpper(strings.TrimSpace(move))
	if len(move) < 2 || len(move) > 3 {
		return 0, 0, fmt.Errorf("invalid move format")
	}
	
	row := int(move[0] - 'A')
	if row < 0 || row >= BoardSize {
		return 0, 0, fmt.Errorf("invalid row")
	}
	
	col, err := strconv.Atoi(move[1:])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid column")
	}
	col-- // Convert to 0-indexed
	
	if col < 0 || col >= BoardSize {
		return 0, 0, fmt.Errorf("invalid column")
	}
	
	return row, col, nil
}

func formatMove(row, col int) string {
	return fmt.Sprintf("%c%d", 'A'+row, col+1)
}

func initializeBoard(board *Board) {
	// Initialize empty grid
	for row := 0; row < BoardSize; row++ {
		for col := 0; col < BoardSize; col++ {
			board.Grid[row][col] = CellEmpty
		}
	}
	
	// Define ships
	shipDefs := []struct {
		name     string
		size     int
		shipType int
		marker   byte
	}{
		{"Aircraft Carrier", 5, ShipAC, 'A'},
		{"Battleship", 4, ShipBS, 'B'},
		{"Cruiser", 3, ShipCR, 'C'},
		{"Submarine", 3, ShipSB, 'S'},
		{"Destroyer", 2, ShipDS, 'D'},
	}
	
	rand.Seed(time.Now().UnixNano())
	
	for _, def := range shipDefs {
		ship := Ship{
			Name:     def.name,
			Size:     def.size,
			ShipType: def.shipType,
			Marker:   def.marker,
		}
		
		// Try to place ship randomly
		placed := false
		for attempts := 0; attempts < 1000 && !placed; attempts++ {
			row := rand.Intn(BoardSize)
			col := rand.Intn(BoardSize)
			horizontal := rand.Intn(2) == 0
			
			if canPlaceShip(board, row, col, def.size, horizontal) {
				placeShip(board, &ship, row, col, horizontal)
				placed = true
			}
		}
		
		board.Ships = append(board.Ships, ship)
	}
}

func canPlaceShip(board *Board, row, col, size int, horizontal bool) bool {
	for i := 0; i < size; i++ {
		r, c := row, col
		if horizontal {
			c += i
		} else {
			r += i
		}
		
		if r >= BoardSize || c >= BoardSize {
			return false
		}
		if board.Grid[r][c] != CellEmpty {
			return false
		}
	}
	return true
}

func placeShip(board *Board, ship *Ship, row, col int, horizontal bool) {
	for i := 0; i < ship.Size; i++ {
		r, c := row, col
		if horizontal {
			c += i
		} else {
			r += i
		}
		board.Grid[r][c] = CellShip
		ship.Cells = append(ship.Cells, [2]int{r, c})
	}
}

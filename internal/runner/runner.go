package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"battleship-arena/internal/storage"
)

var enginePath = getEnginePath()

func getEnginePath() string {
	if path := os.Getenv("BATTLESHIP_ENGINE_PATH"); path != "" {
		return path
	}
	return "./battleship-engine"
}

// runSandboxed executes a command in a systemd-run sandbox with resource limits
func runSandboxed(ctx context.Context, name string, args []string, timeoutSec int) ([]byte, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()
	
	// Check if systemd-run is available (not on macOS/local dev)
	_, err := exec.LookPath("systemd-run")
	if err != nil {
		// Fallback: run directly without sandbox (development only)
		log.Printf("systemd-run not available, running without sandbox: %v", args)
		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		output, err := cmd.CombinedOutput()
		
		if ctx.Err() == context.DeadlineExceeded {
			return output, fmt.Errorf("command timed out after %d seconds", timeoutSec)
		}
		
		return output, err
	}
	
	// Build systemd-run command with security properties
	// Using service unit (not scope) to get access to network/filesystem isolation
	systemdArgs := []string{
		"--wait",           // Wait for service to complete
		"--pipe",           // Pipe stdout/stderr to capture output
		"--unit=" + name,   // Give it a descriptive name
		"--quiet",          // Suppress systemd output
		"--collect",        // Automatically clean up after exit
		"--service-type=exec",  // Run until process exits
		"--working-directory=/var/lib/battleship-arena",  // Ensure proper working directory
		"--property=MemoryMax=512M",        // Max 512MB RAM
		"--property=CPUQuota=200%",         // Max 2 CPU cores worth
		"--property=TasksMax=50",           // Max 50 processes/threads
		"--property=PrivateNetwork=true",   // Isolate network (no internet)
		"--property=PrivateTmp=true",       // Private /tmp
		"--property=NoNewPrivileges=true",  // Prevent privilege escalation
		"--property=ReadWritePaths=/var/lib/battleship-arena",  // Allow writes to battleship directory
		"--",
	}
	systemdArgs = append(systemdArgs, args...)
	
	cmd := exec.CommandContext(ctx, "systemd-run", systemdArgs...)
	
	// Set process group for cleanup
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
	
	output, err := cmd.CombinedOutput()
	
	// Check for timeout
	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("command timed out after %d seconds", timeoutSec)
	}
	
	// Check if process was killed by a signal
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				// Direct execution: check if signaled
				if status.Signaled() {
					sig := status.Signal()
					return output, fmt.Errorf("killed by signal: %s", sig.String())
				}
				// systemd-run execution: exit code 128+N means killed by signal N
				exitCode := status.ExitStatus()
				if exitCode >= 128 && exitCode <= 192 {
					sigNum := exitCode - 128
					sigName := "unknown"
					switch sigNum {
					case 1: sigName = "SIGHUP"
					case 2: sigName = "SIGINT"
					case 3: sigName = "SIGQUIT"
					case 4: sigName = "SIGILL"
					case 5: sigName = "SIGTRAP"
					case 6: sigName = "SIGABRT"
					case 7: sigName = "SIGBUS"
					case 8: sigName = "SIGFPE"
					case 9: sigName = "SIGKILL"
					case 10: sigName = "SIGUSR1"
					case 11: sigName = "SIGSEGV"
					case 12: sigName = "SIGUSR2"
					case 13: sigName = "SIGPIPE"
					case 14: sigName = "SIGALRM"
					case 15: sigName = "SIGTERM"
					default: sigName = fmt.Sprintf("signal %d", sigNum)
					}
					return output, fmt.Errorf("killed by %s (exit code %d)", sigName, exitCode)
				}
			}
		}
	}
	
	return output, err
}

// ensureArenaBuilt compiles the arena binary if it doesn't exist
func ensureArenaBuilt() error {
	buildDir := filepath.Join(enginePath, "build")
	arenaBinary := filepath.Join(buildDir, "arena")
	
	// Check if arena binary exists
	if _, err := os.Stat(arenaBinary); err == nil {
		return nil
	}
	
	os.MkdirAll(buildDir, 0755)
	
	srcDir := filepath.Join(enginePath, "src")
	
	compileArgs := []string{
		"g++", "-std=c++11", "-O3",
		"-I", srcDir,
		"-o", arenaBinary,
		filepath.Join(srcDir, "arena.cpp"),
		filepath.Join(srcDir, "battleship.cpp"),
	}
	
	log.Printf("Building arena binary...")
	output, err := runSandboxed(context.Background(), "build-arena", compileArgs, 120)
	if err != nil {
		return fmt.Errorf("failed to build arena: %s", output)
	}
	
	log.Printf("Arena binary built successfully")
	return nil
}

func CompileSubmission(sub storage.Submission, uploadDir string) error {
	storage.UpdateSubmissionStatus(sub.ID, "testing")

	re := regexp.MustCompile(`memory_functions_(\w+)\.cpp`)
	matches := re.FindStringSubmatch(sub.Filename)
	if len(matches) < 2 {
		return fmt.Errorf("invalid filename format")
	}
	prefix := matches[1]

	buildDir := filepath.Join(enginePath, "build")
	os.MkdirAll(buildDir, 0755)
	
	srcDir := filepath.Join(enginePath, "src")
	os.MkdirAll(srcDir, 0755)

	srcPath := filepath.Join(uploadDir, sub.Username, sub.Filename)
	dstPath := filepath.Join(enginePath, "src", sub.Filename)
	
	log.Printf("Copying %s to %s", srcPath, dstPath)
	input, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	
	if err := os.WriteFile(dstPath, input, 0644); err != nil {
		return err
	}

	functionSuffix, err := parseFunctionNames(string(input))
	if err != nil {
		return fmt.Errorf("failed to parse function names: %v", err)
	}
	
	log.Printf("Detected function suffix: %s", functionSuffix)

	headerFilename := fmt.Sprintf("memory_functions_%s.h", prefix)
	headerPath := filepath.Join(enginePath, "src", headerFilename)
	headerContent := generateHeader(headerFilename, functionSuffix)
	if err := os.WriteFile(headerPath, []byte(headerContent), 0644); err != nil {
		return err
	}

	// Compile player wrapper binary (isolated process for this AI)
	playerBinary := filepath.Join(buildDir, "ai_"+prefix)
	
	log.Printf("Compiling isolated player binary for %s", prefix)
	
	compileArgs := []string{
		"g++", "-std=c++11", "-O3",
		"-I", srcDir,
		fmt.Sprintf("-DPLAYER_SUFFIX=%s", functionSuffix),
		fmt.Sprintf(`-DPLAYER_HEADER="memory_functions_%s.h"`, prefix),
		"-o", playerBinary,
		filepath.Join(srcDir, "player_wrapper.cpp"),
		filepath.Join(srcDir, "battleship.cpp"),
		filepath.Join(srcDir, fmt.Sprintf("memory_functions_%s.cpp", prefix)),
	}
	
	output, err := runSandboxed(context.Background(), "compile-"+prefix, compileArgs, 120)
	if err != nil {
		return fmt.Errorf("compilation failed: %s", output)
	}

	log.Printf("Player binary compiled: %s", playerBinary)
	return nil
}

func RunHeadToHead(player1, player2 storage.Submission, numGames int) (int, int, int, string) {
	// Ensure arena is built
	if err := ensureArenaBuilt(); err != nil {
		return 0, 0, 0, fmt.Sprintf("Failed to build arena: %v", err)
	}
	
	re := regexp.MustCompile(`memory_functions_(\w+)\.cpp`)
	matches1 := re.FindStringSubmatch(player1.Filename)
	matches2 := re.FindStringSubmatch(player2.Filename)
	
	if len(matches1) < 2 || len(matches2) < 2 {
		return 0, 0, 0, "Invalid filename format"
	}
	
	prefix1 := matches1[1]
	prefix2 := matches2[1]
	
	buildDir := filepath.Join(enginePath, "build")
	arenaBinary := filepath.Join(buildDir, "arena")
	player1Binary := filepath.Join(buildDir, "ai_"+prefix1)
	player2Binary := filepath.Join(buildDir, "ai_"+prefix2)
	
	// Check binaries exist
	if _, err := os.Stat(player1Binary); os.IsNotExist(err) {
		return 0, 0, 0, fmt.Sprintf("Player 1 binary missing: %s", player1Binary)
	}
	
	if _, err := os.Stat(player2Binary); os.IsNotExist(err) {
		return 0, 0, 0, fmt.Sprintf("Player 2 binary missing: %s", player2Binary)
	}
	
	// Run arena with both player binaries
	// Arena spawns each player in its own isolated process
	runArgs := []string{arenaBinary, strconv.Itoa(numGames), player1Binary, player2Binary}
	
	log.Printf("Running isolated match: %s vs %s (%d games)", prefix1, prefix2, numGames)
	
	output, err := runSandboxed(context.Background(), "run-match", runArgs, 600)
	if err != nil {
		log.Printf("Match execution failed: %v\n%s", err, output)
		errMsg := strings.TrimSpace(string(output))
		if errMsg != "" {
			return 0, 0, 0, fmt.Sprintf("Runtime error: %s (%s)", errMsg, err.Error())
		}
		return 0, 0, 0, fmt.Sprintf("Runtime error: %s", err.Error())
	}
	
	p1, p2, moves := parseMatchOutput(string(output))
	return p1, p2, moves, ""
}

func RunRoundRobinMatches(newSub storage.Submission, uploadDir string, broadcastFunc func(string, int, int, time.Time, []string)) {
	activeSubmissions, err := storage.GetActiveSubmissions()
	if err != nil {
		log.Printf("Failed to get active submissions: %v", err)
		return
	}

	var unplayedOpponents []storage.Submission
	for _, opponent := range activeSubmissions {
		if opponent.ID == newSub.ID {
			continue
		}
		
		hasMatch, err := storage.HasMatchBetween(newSub.ID, opponent.ID)
		if err != nil {
			log.Printf("Error checking match history: %v", err)
			continue
		}
		
		if !hasMatch {
			// Ensure opponent file exists in engine/src and is compiled
			opponentSrcPath := filepath.Join(uploadDir, opponent.Username, opponent.Filename)
			opponentDstPath := filepath.Join(enginePath, "src", opponent.Filename)
			
			if _, err := os.Stat(opponentDstPath); os.IsNotExist(err) {
				// Copy opponent file to engine/src
				opponentContent, err := os.ReadFile(opponentSrcPath)
				if err != nil {
					log.Printf("Failed to read opponent file %s: %v", opponentSrcPath, err)
					continue
				}
				if err := os.WriteFile(opponentDstPath, opponentContent, 0644); err != nil {
					log.Printf("Failed to copy opponent file to engine: %v", err)
					continue
				}
				
				// Generate opponent header if missing
				re := regexp.MustCompile(`memory_functions_(\w+)\.cpp`)
				matches := re.FindStringSubmatch(opponent.Filename)
				if len(matches) >= 2 {
					prefix := matches[1]
					functionSuffix, err := parseFunctionNames(string(opponentContent))
					if err == nil {
						headerFilename := fmt.Sprintf("memory_functions_%s.h", prefix)
						headerPath := filepath.Join(enginePath, "src", headerFilename)
						headerContent := generateHeader(headerFilename, functionSuffix)
						os.WriteFile(headerPath, []byte(headerContent), 0644)
					}
				}
			}
			
			// Ensure opponent binary is compiled
			re := regexp.MustCompile(`memory_functions_(\w+)\.cpp`)
			matches := re.FindStringSubmatch(opponent.Filename)
			if len(matches) >= 2 {
				prefix := matches[1]
				playerBinary := filepath.Join(enginePath, "build", "ai_"+prefix)
				if _, err := os.Stat(playerBinary); os.IsNotExist(err) {
					// Compile opponent
					if err := CompileSubmission(opponent, uploadDir); err != nil {
						log.Printf("Failed to compile opponent %s: %v", opponent.Username, err)
						continue
					}
				}
			}
			
			unplayedOpponents = append(unplayedOpponents, opponent)
		}
	}
	
	totalMatches := len(unplayedOpponents)
	if totalMatches <= 0 {
		log.Printf("No new opponents for %s, all matches already played", newSub.Username)
		return
	}

	log.Printf("Starting round-robin for %s (%d opponents)", newSub.Username, totalMatches)
	matchNum := 0
	startTime := time.Now()

	for _, opponent := range unplayedOpponents {
		matchNum++
		
		queuedPlayers := storage.GetQueuedPlayerNames()
		broadcastFunc(newSub.Username, matchNum, totalMatches, startTime, queuedPlayers)
		
		player1Wins, player2Wins, totalMoves, errMsg := RunHeadToHead(newSub, opponent, 1000)
		
		// If match failed (returned 0-0-0), mark submission as match_failed with error message
		if player1Wins == 0 && player2Wins == 0 && totalMoves == 0 {
			log.Printf("❌ Match execution failed for %s vs %s - marking as match_failed", newSub.Username, opponent.Username)
			storage.UpdateSubmissionStatusWithMessage(newSub.ID, "match_failed", errMsg)
			return
		}
		
		var winnerID int
		avgMoves := totalMoves / 1000
		
		if player1Wins > player2Wins {
			winnerID = newSub.ID
			log.Printf("[%d/%d] %s defeats %s (%d-%d, %d moves avg)", matchNum, totalMatches, newSub.Username, opponent.Username, player1Wins, player2Wins, avgMoves)
		} else if player2Wins > player1Wins {
			winnerID = opponent.ID
			log.Printf("[%d/%d] %s defeats %s (%d-%d, %d moves avg)", matchNum, totalMatches, opponent.Username, newSub.Username, player2Wins, player1Wins, avgMoves)
		} else {
			if totalMoves%2 == 0 {
				winnerID = newSub.ID
			} else {
				winnerID = opponent.ID
			}
			log.Printf("[%d/%d] Tie %d-%d, coin flip winner: %s", matchNum, totalMatches, player1Wins, player2Wins, 
				map[int]string{newSub.ID: newSub.Username, opponent.ID: opponent.Username}[winnerID])
		}
		
		_, err := storage.AddMatch(newSub.ID, opponent.ID, winnerID, player1Wins, player2Wins, avgMoves, avgMoves)
		if err != nil {
			log.Printf("Failed to store match result: %v", err)
		}
	}
	
	log.Printf("✓ Round-robin complete for %s (%d matches)", newSub.Username, totalMatches)
	
	// Update Glicko-2 ratings using proper rating periods (batch all matches together)
	log.Printf("Updating Glicko-2 ratings (proper rating period)...")
	if err := storage.RecalculateAllGlicko2Ratings(); err != nil {
		log.Printf("Failed to update Glicko-2 ratings: %v", err)
	} else {
		log.Printf("✓ Glicko-2 ratings updated")
	}
}

func recordRatingSnapshot(submissionID, matchID int) {
	var rating, rd, volatility float64
	err := storage.DB.QueryRow(
		"SELECT glicko_rating, glicko_rd, glicko_volatility FROM submissions WHERE id = ?",
		submissionID,
	).Scan(&rating, &rd, &volatility)
	
	if err == nil {
		storage.RecordRatingHistory(submissionID, matchID, rating, rd, volatility)
	}
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

func parseMatchOutput(output string) (int, int, int) {
	player1Wins := 0
	player2Wins := 0
	totalMoves := 0
	
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "PLAYER1_WINS=") {
			fmt.Sscanf(line, "PLAYER1_WINS=%d", &player1Wins)
		} else if strings.HasPrefix(line, "PLAYER2_WINS=") {
			fmt.Sscanf(line, "PLAYER2_WINS=%d", &player2Wins)
		} else if strings.HasPrefix(line, "TOTAL_MOVES=") {
			fmt.Sscanf(line, "TOTAL_MOVES=%d", &totalMoves)
		}
	}
	
	return player1Wins, player2Wins, totalMoves
}

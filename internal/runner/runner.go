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
	
	// Build systemd-run command with security properties
	// Using service unit (not scope) to get access to network/filesystem isolation
	systemdArgs := []string{
		"--unit=" + name,   // Give it a descriptive name
		"--quiet",          // Suppress systemd output
		"--collect",        // Automatically clean up after exit
		"--service-type=exec",  // Run until process exits
		"--property=MemoryMax=512M",        // Max 512MB RAM
		"--property=CPUQuota=200%",         // Max 2 CPU cores worth
		"--property=TasksMax=50",           // Max 50 processes/threads
		"--property=PrivateNetwork=true",   // Isolate network (no internet)
		"--property=PrivateTmp=true",       // Private /tmp
		"--property=NoNewPrivileges=true",  // Prevent privilege escalation
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
	
	return output, err
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

	log.Printf("Compiling submission %d for %s", sub.ID, prefix)
	
	// Compile in sandbox with 60 second timeout
	compileArgs := []string{
		"g++", "-std=c++11", "-c", "-O3",
		"-I", filepath.Join(enginePath, "src"),
		"-o", filepath.Join(buildDir, "ai_"+prefix+".o"),
		filepath.Join(enginePath, "src", sub.Filename),
	}
	
	output, err := runSandboxed(context.Background(), "compile-"+prefix, compileArgs, 60)
	if err != nil {
		return fmt.Errorf("compilation failed: %s", output)
	}

	return nil
}

func RunHeadToHead(player1, player2 storage.Submission, numGames int) (int, int, int) {
	re := regexp.MustCompile(`memory_functions_(\w+)\.cpp`)
	matches1 := re.FindStringSubmatch(player1.Filename)
	matches2 := re.FindStringSubmatch(player2.Filename)
	
	if len(matches1) < 2 || len(matches2) < 2 {
		return 0, 0, 0
	}
	
	prefix1 := matches1[1]
	prefix2 := matches2[1]
	
	cpp1Path := filepath.Join(enginePath, "src", player1.Filename)
	cpp2Path := filepath.Join(enginePath, "src", player2.Filename)
	
	// Ensure both files exist in engine/src (copy from uploads if missing)
	if _, err := os.Stat(cpp1Path); os.IsNotExist(err) {
		log.Printf("Player1 file missing in engine/src, skipping: %s", cpp1Path)
		return 0, 0, 0
	}
	
	if _, err := os.Stat(cpp2Path); os.IsNotExist(err) {
		log.Printf("Player2 file missing in engine/src, skipping: %s", cpp2Path)
		return 0, 0, 0
	}
	
	cpp1Content, err := os.ReadFile(cpp1Path)
	if err != nil {
		log.Printf("Failed to read %s: %v", cpp1Path, err)
		return 0, 0, 0
	}
	
	cpp2Content, err := os.ReadFile(cpp2Path)
	if err != nil {
		log.Printf("Failed to read %s: %v", cpp2Path, err)
		return 0, 0, 0
	}
	
	suffix1, err := parseFunctionNames(string(cpp1Content))
	if err != nil {
		log.Printf("Failed to parse function names for %s: %v", player1.Filename, err)
		return 0, 0, 0
	}
	
	suffix2, err := parseFunctionNames(string(cpp2Content))
	if err != nil {
		log.Printf("Failed to parse function names for %s: %v", player2.Filename, err)
		return 0, 0, 0
	}
	
	buildDir := filepath.Join(enginePath, "build")
	combinedBinary := filepath.Join(buildDir, fmt.Sprintf("match_%s_vs_%s", prefix1, prefix2))
	
	mainContent := generateMatchMain(prefix1, prefix2, suffix1, suffix2)
	mainPath := filepath.Join(enginePath, "src", fmt.Sprintf("match_%s_vs_%s.cpp", prefix1, prefix2))
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		log.Printf("Failed to write match main: %v", err)
		return 0, 0, 0
	}
	
	// Compile match binary in sandbox with 120 second timeout
	compileArgs := []string{"g++"}
	compileArgs = append(compileArgs, "-std=c++11", "-O3",
		"-o", combinedBinary,
		mainPath,
		filepath.Join(enginePath, "src", "battleship_light.cpp"),
	)
	
	if prefix1 == prefix2 {
		compileArgs = append(compileArgs, filepath.Join(enginePath, "src", fmt.Sprintf("memory_functions_%s.cpp", prefix1)))
	} else {
		compileArgs = append(compileArgs,
			filepath.Join(enginePath, "src", fmt.Sprintf("memory_functions_%s.cpp", prefix1)),
			filepath.Join(enginePath, "src", fmt.Sprintf("memory_functions_%s.cpp", prefix2)),
		)
	}
	
	output, err := runSandboxed(context.Background(), "compile-match", compileArgs, 120)
	if err != nil {
		log.Printf("Failed to compile match binary: %s", output)
		return 0, 0, 0
	}
	
	// Run match in sandbox with 300 second timeout (1000 games should be ~60s, give headroom)
	runArgs := []string{combinedBinary, strconv.Itoa(numGames)}
	output, err = runSandboxed(context.Background(), "run-match", runArgs, 300)
	if err != nil {
		log.Printf("Match execution failed: %v\n%s", err, output)
		return 0, 0, 0
	}
	
	return parseMatchOutput(string(output))
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
			// Ensure opponent file exists in engine/src
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
		
		player1Wins, player2Wins, totalMoves := RunHeadToHead(newSub, opponent, 1000)
		
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
		
		matchID, err := storage.AddMatch(newSub.ID, opponent.ID, winnerID, player1Wins, player2Wins, avgMoves, avgMoves)
		if err != nil {
			log.Printf("Failed to store match result: %v", err)
		} else {
			if err := storage.UpdateGlicko2Ratings(newSub.ID, opponent.ID, player1Wins, player2Wins); err != nil {
				log.Printf("Glicko-2 update failed: %v", err)
			} else {
				recordRatingSnapshot(newSub.ID, int(matchID))
				recordRatingSnapshot(opponent.ID, int(matchID))
			}
		}
	}
	
	log.Printf("✓ Round-robin complete for %s (%d matches)", newSub.Username, totalMatches)
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
	functionSuffix := strings.ToUpper(prefix[0:1]) + prefix[1:]
	
	return fmt.Sprintf(`#ifndef %s
#define %s

#include "memory.h"
#include "battleship_light.h"
#include <string>

void initMemory%s(ComputerMemory &memory);
std::string smartMove%s(const ComputerMemory &memory);
void updateMemory%s(int row, int col, int result, ComputerMemory &memory);

#endif
`, guard, guard, functionSuffix, functionSuffix, functionSuffix)
}

func generateMatchMain(prefix1, prefix2, suffix1, suffix2 string) string {
	return fmt.Sprintf(`#include "battleship_light.h"
#include "memory.h"
#include "memory_functions_%s.h"
#include "memory_functions_%s.h"
#include <iostream>
#include <cstdlib>
#include <ctime>

using namespace std;

struct MatchResult {
    int player1Wins = 0;
    int player2Wins = 0;
    int ties = 0;
    int totalMoves = 0;
};

MatchResult runMatch(int numGames) {
    MatchResult result;
    srand(time(NULL));
    
    for (int game = 0; game < numGames; game++) {
        Board board1, board2;
        ComputerMemory memory1, memory2;
        
        initializeBoard(board1);
        initializeBoard(board2);
        initMemory%s(memory1);
        initMemory%s(memory2);
        
        int shipsSunk1 = 0;
        int shipsSunk2 = 0;
        int moveCount = 0;
        
        while (true) {
            moveCount++;
            
            string move1 = smartMove%s(memory1);
            int row1, col1;
            int check1 = checkMove(move1, board2, row1, col1);
            while (check1 != VALID_MOVE) {
                move1 = randomMove();
                check1 = checkMove(move1, board2, row1, col1);
            }
            
            string move2 = smartMove%s(memory2);
            int row2, col2;
            int check2 = checkMove(move2, board1, row2, col2);
            while (check2 != VALID_MOVE) {
                move2 = randomMove();
                check2 = checkMove(move2, board1, row2, col2);
            }
            
            int result1 = playMove(row1, col1, board2);
            int result2 = playMove(row2, col2, board1);
            
            updateMemory%s(row1, col1, result1, memory1);
            updateMemory%s(row2, col2, result2, memory2);
            
            if (isASunk(result1)) shipsSunk1++;
            if (isASunk(result2)) shipsSunk2++;
            
            if (shipsSunk1 == 5 || shipsSunk2 == 5) {
                break;
            }
        }
        
        result.totalMoves += moveCount;
        
        if (shipsSunk1 == 5 && shipsSunk2 == 5) {
            result.ties++;
        } else if (shipsSunk1 == 5) {
            result.player1Wins++;
        } else {
            result.player2Wins++;
        }
    }
    
    return result;
}

int main(int argc, char* argv[]) {
    if (argc < 2) {
        cerr << "Usage: " << argv[0] << " <num_games>" << endl;
        return 1;
    }
    
    int numGames = atoi(argv[1]);
    if (numGames <= 0) numGames = 10;
    
    setDebugMode(false);
    
    MatchResult result = runMatch(numGames);
    
    cout << "PLAYER1_WINS=" << result.player1Wins << endl;
    cout << "PLAYER2_WINS=" << result.player2Wins << endl;
    cout << "TIES=" << result.ties << endl;
    cout << "TOTAL_MOVES=" << result.totalMoves << endl;
    cout << "AVG_MOVES=" << (result.totalMoves / numGames) << endl;
    
    return 0;
}
`, prefix1, prefix2, suffix1, suffix2, suffix1, suffix2, suffix1, suffix2)
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

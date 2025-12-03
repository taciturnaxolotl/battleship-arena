package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const enginePath = "./battleship-engine"

func processSubmissions() error {
	submissions, err := getPendingSubmissions()
	if err != nil {
		return err
	}

	for _, sub := range submissions {
		log.Printf("⚙️  Compiling %s (%s)", sub.Username, sub.Filename)
		
		if err := compileSubmission(sub); err != nil {
			log.Printf("❌ Compilation failed for %s: %v", sub.Username, err)
			updateSubmissionStatus(sub.ID, "failed")
			continue
		}
		
		log.Printf("✓ Compiled %s", sub.Username)
		updateSubmissionStatus(sub.ID, "completed")
		
		// Run round-robin matches
		runRoundRobinMatches(sub)
	}

	return nil
}

func generateHeader(filename, prefix string) string {
	guard := strings.ToUpper(strings.Replace(filename, ".", "_", -1))
	
	// Capitalize first letter of prefix for function names
	functionSuffix := strings.ToUpper(prefix[0:1]) + prefix[1:]
	
	return fmt.Sprintf(`#ifndef %s
#define %s

#include "memory.h"
#include <string>

void initMemory%s(ComputerMemory &memory);
std::string smartMove%s(const ComputerMemory &memory);
void updateMemory%s(int row, int col, int result, ComputerMemory &memory);

#endif
`, guard, guard, functionSuffix, functionSuffix, functionSuffix)
}

func parseFunctionNames(cppContent string) (string, error) {
	// Look for the initMemory function to extract the suffix
	re := regexp.MustCompile(`void\s+initMemory(\w+)\s*\(`)
	matches := re.FindStringSubmatch(cppContent)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not find initMemory function")
	}
	return matches[1], nil
}

func compileSubmission(sub Submission) error {
	updateSubmissionStatus(sub.ID, "testing")

	// Extract prefix from filename (memory_functions_XXXXX.cpp -> XXXXX)
	re := regexp.MustCompile(`memory_functions_(\w+)\.cpp`)
	matches := re.FindStringSubmatch(sub.Filename)
	if len(matches) < 2 {
		return fmt.Errorf("invalid filename format")
	}
	prefix := matches[1]

	// Create temporary build directory
	buildDir := filepath.Join(enginePath, "build")
	os.MkdirAll(buildDir, 0755)

	// Copy submission to engine
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

	// Parse function names from the cpp file
	functionSuffix, err := parseFunctionNames(string(input))
	if err != nil {
		return fmt.Errorf("failed to parse function names: %v", err)
	}
	
	log.Printf("Detected function suffix: %s", functionSuffix)

	// Generate header file with parsed function names
	headerFilename := fmt.Sprintf("memory_functions_%s.h", prefix)
	headerPath := filepath.Join(enginePath, "src", headerFilename)
	headerContent := generateHeader(headerFilename, functionSuffix)
	if err := os.WriteFile(headerPath, []byte(headerContent), 0644); err != nil {
		return err
	}

	log.Printf("Compiling submission %d for %s", sub.ID, prefix)
	
	// Compile check only (no linking) to validate syntax
	cmd := exec.Command("g++", "-std=c++11", "-c", "-O3",
		"-I", filepath.Join(enginePath, "src"),
		"-o", filepath.Join(buildDir, "ai_"+prefix+".o"),
		filepath.Join(enginePath, "src", sub.Filename),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compilation failed: %s", output)
	}

	return nil
}



func getSubmissionByID(id int) (Submission, error) {
	var sub Submission
	err := globalDB.QueryRow(
		"SELECT id, username, filename, upload_time, status FROM submissions WHERE id = ?",
		id,
	).Scan(&sub.ID, &sub.Username, &sub.Filename, &sub.UploadTime, &sub.Status)
	return sub, err
}

func runRoundRobinMatches(newSub Submission) {
	// Get all active submissions
	activeSubmissions, err := getActiveSubmissions()
	if err != nil {
		log.Printf("Failed to get active submissions: %v", err)
		return
	}

	// Filter to only opponents we haven't played yet
	var unplayedOpponents []Submission
	for _, opponent := range activeSubmissions {
		if opponent.ID == newSub.ID {
			continue
		}
		
		// Check if match already exists
		hasMatch, err := hasMatchBetween(newSub.ID, opponent.ID)
		if err != nil {
			log.Printf("Error checking match history: %v", err)
			continue
		}
		
		if !hasMatch {
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

	// Run matches against unplayed opponents only
	for _, opponent := range unplayedOpponents {
		matchNum++
		
		// Run match (1000 games total)
		player1Wins, player2Wins, totalMoves := runHeadToHead(newSub, opponent, 1000)
		
		// Determine winner
		var winnerID int
		avgMoves := totalMoves / 1000
		
		if player1Wins > player2Wins {
			winnerID = newSub.ID
			log.Printf("[%d/%d] %s defeats %s (%d-%d, %d moves avg)", matchNum, totalMatches, newSub.Username, opponent.Username, player1Wins, player2Wins, avgMoves)
		} else if player2Wins > player1Wins {
			winnerID = opponent.ID
			log.Printf("[%d/%d] %s defeats %s (%d-%d, %d moves avg)", matchNum, totalMatches, opponent.Username, newSub.Username, player2Wins, player1Wins, avgMoves)
		} else {
			// Tie - coin flip
			if totalMoves%2 == 0 {
				winnerID = newSub.ID
			} else {
				winnerID = opponent.ID
			}
			log.Printf("[%d/%d] Tie %d-%d, coin flip winner: %s", matchNum, totalMatches, player1Wins, player2Wins, 
				map[int]string{newSub.ID: newSub.Username, opponent.ID: opponent.Username}[winnerID])
		}
		
		// Store match result
		if err := addMatch(newSub.ID, opponent.ID, winnerID, player1Wins, player2Wins, avgMoves, avgMoves); err != nil {
			log.Printf("Failed to store match result: %v", err)
		} else {
			// Update ELO ratings based on actual win percentages
			if err := updateEloRatings(newSub.ID, opponent.ID, player1Wins, player2Wins); err != nil {
				log.Printf("ELO update failed: %v", err)
			}
			
			NotifyLeaderboardUpdate()
		}
	}
	
	log.Printf("✓ Round-robin complete for %s (%d matches)", newSub.Username, totalMatches)
}

func runHeadToHead(player1, player2 Submission, numGames int) (int, int, int) {
	re := regexp.MustCompile(`memory_functions_(\w+)\.cpp`)
	matches1 := re.FindStringSubmatch(player1.Filename)
	matches2 := re.FindStringSubmatch(player2.Filename)
	
	if len(matches1) < 2 || len(matches2) < 2 {
		return 0, 0, 0
	}
	
	prefix1 := matches1[1]
	prefix2 := matches2[1]
	
	// Read both cpp files to extract function suffixes
	cpp1Path := filepath.Join(enginePath, "src", player1.Filename)
	cpp2Path := filepath.Join(enginePath, "src", player2.Filename)
	
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
	
	// Create a combined binary with both AIs
	combinedBinary := filepath.Join(buildDir, fmt.Sprintf("match_%s_vs_%s", prefix1, prefix2))
	
	// Generate main file that uses both AIs with correct function suffixes
	mainContent := generateMatchMain(prefix1, prefix2, suffix1, suffix2)
	mainPath := filepath.Join(enginePath, "src", fmt.Sprintf("match_%s_vs_%s.cpp", prefix1, prefix2))
	if err := os.WriteFile(mainPath, []byte(mainContent), 0644); err != nil {
		log.Printf("Failed to write match main: %v", err)
		return 0, 0, 0
	}
	
	// Compile combined binary
	compileArgs := []string{"-std=c++11", "-O3",
		"-o", combinedBinary,
		mainPath,
		filepath.Join(enginePath, "src", "battleship_light.cpp"),
	}
	
	// Add player files (avoid duplicates if same AI)
	if prefix1 == prefix2 {
		// Same AI - only compile once
		compileArgs = append(compileArgs, filepath.Join(enginePath, "src", fmt.Sprintf("memory_functions_%s.cpp", prefix1)))
	} else {
		// Different AIs
		compileArgs = append(compileArgs,
			filepath.Join(enginePath, "src", fmt.Sprintf("memory_functions_%s.cpp", prefix1)),
			filepath.Join(enginePath, "src", fmt.Sprintf("memory_functions_%s.cpp", prefix2)),
		)
	}
	
	cmd := exec.Command("g++", compileArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to compile match binary: %s", output)
		return 0, 0, 0
	}
	
	// Run the match
	cmd = exec.Command(combinedBinary, strconv.Itoa(numGames))
	output, err = cmd.CombinedOutput()
	if err != nil {
		log.Printf("Match execution failed: %v", err)
		return 0, 0, 0
	}
	
	// Parse results
	return parseMatchOutput(string(output))
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
            
            // Player 1 move
            string move1 = smartMove%s(memory1);
            int row1, col1;
            int check1 = checkMove(move1, board2, row1, col1);
            while (check1 != VALID_MOVE) {
                move1 = randomMove();
                check1 = checkMove(move1, board2, row1, col1);
            }
            
            // Player 2 move
            string move2 = smartMove%s(memory2);
            int row2, col2;
            int check2 = checkMove(move2, board1, row2, col2);
            while (check2 != VALID_MOVE) {
                move2 = randomMove();
                check2 = checkMove(move2, board1, row2, col2);
            }
            
            // Execute moves
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
    
    // Output in parseable format
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

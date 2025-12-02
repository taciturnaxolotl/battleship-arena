package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const battleshipRepoPath = "/Users/kierank/code/school/cs1210-battleship"

func processSubmissions() error {
	submissions, err := getPendingSubmissions()
	if err != nil {
		return err
	}

	for _, sub := range submissions {
		log.Printf("Starting test for submission %d: %s by %s", sub.ID, sub.Filename, sub.Username)
		
		if err := testSubmission(sub); err != nil {
			log.Printf("Submission %d failed: %v", sub.ID, err)
			updateSubmissionStatus(sub.ID, "failed")
			continue
		}
		
		log.Printf("Submission %d completed successfully: %s by %s", sub.ID, sub.Filename, sub.Username)
		updateSubmissionStatus(sub.ID, "completed")
	}

	return nil
}

func testSubmission(sub Submission) error {
	log.Printf("Setting submission %d to testing status", sub.ID)
	updateSubmissionStatus(sub.ID, "testing")

	// Copy submission to battleship repo
	srcPath := filepath.Join(uploadDir, sub.Username, sub.Filename)
	dstPath := filepath.Join(battleshipRepoPath, "src", sub.Filename)

	log.Printf("Copying %s to %s", srcPath, dstPath)
	input, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	if err := os.WriteFile(dstPath, input, 0644); err != nil {
		return err
	}

	// Extract student ID from filename (memory_functions_NNNN.cpp)
	re := regexp.MustCompile(`memory_functions_(\w+)\.cpp`)
	matches := re.FindStringSubmatch(sub.Filename)
	if len(matches) < 2 {
		return fmt.Errorf("invalid filename format")
	}
	studentID := matches[1]

	// Build the battleship program
	buildDir := filepath.Join(battleshipRepoPath, "build")
	os.MkdirAll(buildDir, 0755)

	log.Printf("Compiling submission %d for student %s", sub.ID, studentID)
	// Compile using the light version for testing
	cmd := exec.Command("g++", "-std=c++11", "-O3",
		"-o", filepath.Join(buildDir, "battle_"+studentID),
		filepath.Join(battleshipRepoPath, "src", "battle_light.cpp"),
		filepath.Join(battleshipRepoPath, "src", "battleship_light.cpp"),
		filepath.Join(battleshipRepoPath, "src", sub.Filename),
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compilation failed: %s", output)
	}

	log.Printf("Running benchmark for submission %d (100 games)", sub.ID)
	// Run benchmark tests (100 games)
	cmd = exec.Command(filepath.Join(buildDir, "battle_"+studentID), "--benchmark", "100")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("benchmark failed: %s", output)
	}

	// Parse results and store in database
	log.Printf("Parsing results for submission %d", sub.ID)
	results := parseResults(string(output))
	for opponent, result := range results {
		addResult(sub.ID, opponent, result.Result, result.Moves)
	}

	return nil
}

type GameResult struct {
	Result string
	Moves  int
}

func parseResults(output string) map[string]GameResult {
	results := make(map[string]GameResult)
	
	// Parse win/loss stats from benchmark output
	// Example: "Smart AI wins: 95 (95.0%)"
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Smart AI wins:") {
			// Extract win count
			re := regexp.MustCompile(`Smart AI wins: (\d+)`)
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 2 {
				// For now, just record as wins against "random"
				results["random"] = GameResult{Result: "win", Moves: 50}
			}
		}
	}
	
	return results
}

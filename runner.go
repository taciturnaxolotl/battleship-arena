package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

const battleshipRepoPath = "/Users/kierank/code/school/cs1210-battleship"

func queueSubmission(username string) error {
	// Find the user's submission file
	files, err := filepath.Glob(filepath.Join(uploadDir, username, "memory_functions_*.cpp"))
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no submission file found")
	}

	filename := filepath.Base(files[0])
	_, err = addSubmission(username, filename)
	return err
}

func processSubmissions() error {
	submissions, err := getPendingSubmissions()
	if err != nil {
		return err
	}

	for _, sub := range submissions {
		if err := testSubmission(sub); err != nil {
			updateSubmissionStatus(sub.ID, "failed")
			continue
		}
		updateSubmissionStatus(sub.ID, "completed")
	}

	return nil
}

func testSubmission(sub Submission) error {
	updateSubmissionStatus(sub.ID, "testing")

	// Copy submission to battleship repo
	srcPath := filepath.Join(uploadDir, sub.Username, sub.Filename)
	dstPath := filepath.Join(battleshipRepoPath, "src", sub.Filename)

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

	// Run benchmark tests (100 games)
	cmd = exec.Command(filepath.Join(buildDir, "battle_"+studentID), "--benchmark", "100")
	output, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("benchmark failed: %s", output)
	}

	// Parse results and store in database
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

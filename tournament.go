package main

import (
	"fmt"
	"log"
	"math"
	"sort"
)

// seedSubmissions sorts submissions by average moves (better performance = lower seed number)
func seedSubmissions(submissions []Submission) []Submission {
	// Calculate historical average moves for each submission
	type seedEntry struct {
		submission Submission
		avgMoves   float64
	}
	
	var entries []seedEntry
	for _, sub := range submissions {
		// Get historical performance
		var avgMoves float64
		err := globalDB.QueryRow(`
			SELECT AVG(CASE 
				WHEN m.player1_id = ? THEN m.player1_moves 
				ELSE m.player2_moves 
			END)
			FROM matches m
			WHERE m.player1_id = ? OR m.player2_id = ?
		`, sub.ID, sub.ID, sub.ID).Scan(&avgMoves)
		
		if err != nil || avgMoves == 0 {
			// No history, use worst seed (100 moves)
			avgMoves = 100
		}
		
		entries = append(entries, seedEntry{sub, avgMoves})
	}
	
	// Sort by avgMoves (lower is better)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].avgMoves < entries[j].avgMoves
	})
	
	var seeded []Submission
	for _, entry := range entries {
		seeded = append(seeded, entry.submission)
	}
	
	return seeded
}

// createBracket generates bracket matches for round 1
func createBracket(tournament *Tournament) error {
	// Get all active submissions
	submissions, err := getActiveSubmissions()
	if err != nil {
		return err
	}
	
	if len(submissions) < 2 {
		return fmt.Errorf("need at least 2 players for tournament")
	}
	
	// Seed players
	seeded := seedSubmissions(submissions)
	
	log.Printf("Tournament %d: Seeded %d players", tournament.ID, len(seeded))
	for i, sub := range seeded {
		log.Printf("  Seed %d: %s", i+1, sub.Username)
	}
	
	// Find next power of 2
	numPlayers := len(seeded)
	bracketSize := int(math.Pow(2, math.Ceil(math.Log2(float64(numPlayers)))))
	
	log.Printf("Tournament %d: Bracket size=%d, players=%d", 
		tournament.ID, bracketSize, numPlayers)
	
	// Create round 1 matches using standard bracket seeding
	// Seed 1 vs Seed 16, Seed 2 vs Seed 15, etc.
	// For non-power-of-2, higher seeds get byes
	
	numFirstRoundMatches := bracketSize / 2
	
	for matchPos := 0; matchPos < numFirstRoundMatches; matchPos++ {
		// Standard bracket pairing: top seed vs bottom seed
		topSeedIdx := matchPos
		bottomSeedIdx := numPlayers - 1 - matchPos
		
		// Stop if we've paired all players (indices would overlap)
		if topSeedIdx >= bottomSeedIdx && topSeedIdx < numPlayers && bottomSeedIdx >= 0 {
			// Only one player left unpaired - give them a bye
			if topSeedIdx == bottomSeedIdx {
				player1ID := seeded[topSeedIdx].ID
				player1Name := seeded[topSeedIdx].Username
				
				err = addBracketMatch(tournament.ID, 1, matchPos, player1ID, 0)
				if err != nil {
					return err
				}
				
				globalDB.Exec(`
					UPDATE bracket_matches 
					SET winner_id = ?, status = 'completed', 
					    player1_wins = 0, player2_wins = 0,
					    player1_moves = 0, player2_moves = 0
					WHERE tournament_id = ? AND round = 1 AND position = ?
				`, player1ID, tournament.ID, matchPos)
				
				log.Printf("  Match %d: %s vs BYE (auto-advance)", matchPos, player1Name)
			}
			break
		}
		
		var player1ID, player2ID int
		var player1Name, player2Name string
		
		// Check if top seed exists
		if topSeedIdx < numPlayers {
			player1ID = seeded[topSeedIdx].ID
			player1Name = seeded[topSeedIdx].Username
		} else {
			player1ID = 0
			player1Name = "BYE"
		}
		
		// Check if bottom seed exists
		if bottomSeedIdx >= 0 && bottomSeedIdx < numPlayers {
			player2ID = seeded[bottomSeedIdx].ID
			player2Name = seeded[bottomSeedIdx].Username
		} else {
			player2ID = 0
			player2Name = "BYE"
		}
		
		// Skip if both are byes
		if player1ID == 0 && player2ID == 0 {
			continue
		}
		
		// Create match
		err = addBracketMatch(tournament.ID, 1, matchPos, player1ID, player2ID)
		if err != nil {
			return err
		}
		
		// If one is a bye, auto-complete
		if player1ID == 0 || player2ID == 0 {
			winnerID := player1ID
			if player1ID == 0 {
				winnerID = player2ID
			}
			
			globalDB.Exec(`
				UPDATE bracket_matches 
				SET winner_id = ?, status = 'completed', 
				    player1_wins = 0, player2_wins = 0,
				    player1_moves = 0, player2_moves = 0
				WHERE tournament_id = ? AND round = 1 AND position = ?
			`, winnerID, tournament.ID, matchPos)
			
			log.Printf("  Match %d: %s vs %s (BYE - winner: %s)", 
				matchPos, player1Name, player2Name, 
				map[bool]string{true: player1Name, false: player2Name}[player1ID == winnerID])
		} else {
			log.Printf("  Match %d: %s (seed %d) vs %s (seed %d)", 
				matchPos, player1Name, topSeedIdx+1, player2Name, bottomSeedIdx+1)
		}
	}
	
	return nil
}

// advanceWinners creates next round matches from current round winners
func advanceWinners(tournamentID, currentRound int) error {
	// Get completed matches from current round
	query := `
		SELECT id, position, winner_id 
		FROM bracket_matches 
		WHERE tournament_id = ? AND round = ? AND status = 'completed'
		ORDER BY position
	`
	
	rows, err := globalDB.Query(query, tournamentID, currentRound)
	if err != nil {
		return err
	}
	defer rows.Close()
	
	var winners []struct {
		matchID  int
		position int
		winnerID int
	}
	
	for rows.Next() {
		var w struct {
			matchID  int
			position int
			winnerID int
		}
		rows.Scan(&w.matchID, &w.position, &w.winnerID)
		winners = append(winners, w)
	}
	
	if len(winners) == 1 {
		// Tournament complete!
		log.Printf("Tournament %d complete! Winner: ID %d", tournamentID, winners[0].winnerID)
		return completeTournament(tournamentID, winners[0].winnerID)
	}
	
	// Create matches for next round
	nextRound := currentRound + 1
	log.Printf("Advancing to round %d with %d winners", nextRound, len(winners))
	
	for i := 0; i < len(winners); i += 2 {
		if i+1 >= len(winners) {
			// Odd number of winners, give bye
			log.Printf("  Round %d Match %d: BYE (winner: ID %d)", nextRound, i/2, winners[i].winnerID)
			err = addBracketMatch(tournamentID, nextRound, i/2, winners[i].winnerID, 0)
			if err != nil {
				return err
			}
			// Auto-complete bye
			globalDB.Exec(`
				UPDATE bracket_matches 
				SET winner_id = ?, status = 'completed',
				    player1_wins = 0, player2_wins = 0,
				    player1_moves = 0, player2_moves = 0
				WHERE tournament_id = ? AND round = ? AND position = ?
			`, winners[i].winnerID, tournamentID, nextRound, i/2)
		} else {
			log.Printf("  Round %d Match %d: ID %d vs ID %d", nextRound, i/2, winners[i].winnerID, winners[i+1].winnerID)
			err = addBracketMatch(tournamentID, nextRound, i/2, winners[i].winnerID, winners[i+1].winnerID)
			if err != nil {
				return err
			}
		}
	}
	
	// Update tournament round
	return updateTournamentRound(tournamentID, nextRound)
}

// ensureTournamentExists creates a tournament if none exists AND there are new submissions
func ensureTournamentExists() (*Tournament, error) {
	tournament, err := getActiveTournament()
	if err != nil {
		return nil, err
	}
	
	if tournament != nil {
		// Active tournament exists
		return tournament, nil
	}
	
	// No active tournament - check if we should create one
	// Only create if there's been a new submission since last tournament
	latestTournament, err := getLatestTournament()
	if err != nil {
		return nil, err
	}
	
	submissions, err := getActiveSubmissions()
	if err != nil {
		return nil, err
	}
	
	if len(submissions) < 2 {
		log.Printf("Not enough players for tournament (%d/2), waiting...", len(submissions))
		return nil, fmt.Errorf("need at least 2 players")
	}
	
	// Check if any submission is newer than the last tournament
	if latestTournament != nil {
		hasNewSubmission := false
		for _, sub := range submissions {
			if sub.UploadTime.After(latestTournament.CreatedAt) {
				hasNewSubmission = true
				break
			}
		}
		
		if !hasNewSubmission {
			log.Printf("No new submissions since last tournament, not creating new tournament")
			return nil, fmt.Errorf("no new submissions")
		}
	}
	
	log.Printf("Creating tournament with %d players...", len(submissions))
	tournament, err = createTournament()
	if err != nil {
		return nil, err
	}
	
	// Generate bracket
	err = createBracket(tournament)
	if err != nil {
		return nil, err
	}
	log.Printf("Created tournament %d with bracket", tournament.ID)
	
	return tournament, nil
}

package storage

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"sort"
)

func GetActiveTournament() (*Tournament, error) {
	var t Tournament
	var winnerID sql.NullInt64
	err := DB.QueryRow(
		"SELECT id, created_at, status, current_round, winner_id FROM tournaments WHERE status = 'active' ORDER BY id DESC LIMIT 1",
	).Scan(&t.ID, &t.CreatedAt, &t.Status, &t.CurrentRound, &winnerID)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if winnerID.Valid {
		t.WinnerID = int(winnerID.Int64)
	}
	return &t, err
}

func GetLatestTournament() (*Tournament, error) {
	var t Tournament
	var winnerID sql.NullInt64
	err := DB.QueryRow(
		"SELECT id, created_at, status, current_round, winner_id FROM tournaments ORDER BY id DESC LIMIT 1",
	).Scan(&t.ID, &t.CreatedAt, &t.Status, &t.CurrentRound, &winnerID)
	
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if winnerID.Valid {
		t.WinnerID = int(winnerID.Int64)
	}
	return &t, err
}

func CreateTournament() (*Tournament, error) {
	result, err := DB.Exec("INSERT INTO tournaments (status, current_round) VALUES ('active', 1)")
	if err != nil {
		return nil, err
	}
	
	id, _ := result.LastInsertId()
	return &Tournament{
		ID:           int(id),
		Status:       "active",
		CurrentRound: 1,
	}, nil
}

func UpdateTournamentRound(tournamentID, round int) error {
	_, err := DB.Exec("UPDATE tournaments SET current_round = ? WHERE id = ?", round, tournamentID)
	return err
}

func CompleteTournament(tournamentID, winnerID int) error {
	_, err := DB.Exec("UPDATE tournaments SET status = 'completed', winner_id = ? WHERE id = ?", winnerID, tournamentID)
	return err
}

func AddBracketMatch(tournamentID, round, position, player1ID, player2ID int) error {
	_, err := DB.Exec(
		"INSERT INTO bracket_matches (tournament_id, round, position, player1_id, player2_id, status) VALUES (?, ?, ?, ?, ?, 'pending')",
		tournamentID, round, position, player1ID, player2ID,
	)
	return err
}

func GetPendingBracketMatches(tournamentID int) ([]BracketMatch, error) {
	query := `
	SELECT 
		bm.id, bm.tournament_id, bm.round, bm.position,
		bm.player1_id, bm.player2_id, bm.winner_id,
		bm.player1_wins, bm.player2_wins,
		bm.player1_moves, bm.player2_moves, bm.status,
		s1.username as player1_name, s2.username as player2_name
	FROM bracket_matches bm
	JOIN submissions s1 ON bm.player1_id = s1.id
	JOIN submissions s2 ON bm.player2_id = s2.id
	WHERE bm.tournament_id = ? AND bm.status = 'pending'
	ORDER BY bm.round, bm.position
	`
	
	rows, err := DB.Query(query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var matches []BracketMatch
	for rows.Next() {
		var m BracketMatch
		var winnerID sql.NullInt64
		var player1Moves, player2Moves sql.NullInt64
		err := rows.Scan(
			&m.ID, &m.TournamentID, &m.Round, &m.Position,
			&m.Player1ID, &m.Player2ID, &winnerID,
			&m.Player1Wins, &m.Player2Wins,
			&player1Moves, &player2Moves, &m.Status,
			&m.Player1Name, &m.Player2Name,
		)
		if err != nil {
			return nil, err
		}
		if winnerID.Valid {
			m.WinnerID = int(winnerID.Int64)
		}
		if player1Moves.Valid {
			m.Player1Moves = int(player1Moves.Int64)
		}
		if player2Moves.Valid {
			m.Player2Moves = int(player2Moves.Int64)
		}
		matches = append(matches, m)
	}
	
	return matches, rows.Err()
}

func GetAllBracketMatches(tournamentID int) ([]BracketMatch, error) {
	query := `
	SELECT 
		bm.id, bm.tournament_id, bm.round, bm.position,
		bm.player1_id, bm.player2_id, bm.winner_id,
		bm.player1_wins, bm.player2_wins,
		bm.player1_moves, bm.player2_moves, bm.status,
		s1.username as player1_name, s2.username as player2_name
	FROM bracket_matches bm
	LEFT JOIN submissions s1 ON bm.player1_id = s1.id
	LEFT JOIN submissions s2 ON bm.player2_id = s2.id
	WHERE bm.tournament_id = ?
	ORDER BY bm.round, bm.position
	`
	
	rows, err := DB.Query(query, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var matches []BracketMatch
	for rows.Next() {
		var m BracketMatch
		var player1Name, player2Name sql.NullString
		var winnerID, player1Moves, player2Moves sql.NullInt64
		err := rows.Scan(
			&m.ID, &m.TournamentID, &m.Round, &m.Position,
			&m.Player1ID, &m.Player2ID, &winnerID,
			&m.Player1Wins, &m.Player2Wins,
			&player1Moves, &player2Moves, &m.Status,
			&player1Name, &player2Name,
		)
		if err != nil {
			return nil, err
		}
		if winnerID.Valid {
			m.WinnerID = int(winnerID.Int64)
		}
		if player1Moves.Valid {
			m.Player1Moves = int(player1Moves.Int64)
		}
		if player2Moves.Valid {
			m.Player2Moves = int(player2Moves.Int64)
		}
		if player1Name.Valid {
			m.Player1Name = player1Name.String
		}
		if player2Name.Valid {
			m.Player2Name = player2Name.String
		}
		matches = append(matches, m)
	}
	
	return matches, rows.Err()
}

func UpdateBracketMatchResult(matchID, winnerID, player1Wins, player2Wins, player1Moves, player2Moves int) error {
	_, err := DB.Exec(
		`UPDATE bracket_matches 
		SET winner_id = ?, player1_wins = ?, player2_wins = ?, 
		    player1_moves = ?, player2_moves = ?, status = 'completed' 
		WHERE id = ?`,
		winnerID, player1Wins, player2Wins, player1Moves, player2Moves, matchID,
	)
	return err
}

func IsRoundComplete(tournamentID, round int) (bool, error) {
	var pendingCount int
	err := DB.QueryRow(
		"SELECT COUNT(*) FROM bracket_matches WHERE tournament_id = ? AND round = ? AND status != 'completed'",
		tournamentID, round,
	).Scan(&pendingCount)
	
	return pendingCount == 0, err
}

func SeedSubmissions(submissions []Submission) []Submission {
	type seedEntry struct {
		submission Submission
		avgMoves   float64
	}
	
	var entries []seedEntry
	for _, sub := range submissions {
		var avgMoves float64
		err := DB.QueryRow(`
			SELECT AVG(CASE 
				WHEN m.player1_id = ? THEN m.player1_moves 
				ELSE m.player2_moves 
			END)
			FROM matches m
			WHERE m.player1_id = ? OR m.player2_id = ?
		`, sub.ID, sub.ID, sub.ID).Scan(&avgMoves)
		
		if err != nil || avgMoves == 0 {
			avgMoves = 100
		}
		
		entries = append(entries, seedEntry{sub, avgMoves})
	}
	
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].avgMoves < entries[j].avgMoves
	})
	
	var seeded []Submission
	for _, entry := range entries {
		seeded = append(seeded, entry.submission)
	}
	
	return seeded
}

func CreateBracket(tournament *Tournament) error {
	submissions, err := GetActiveSubmissions()
	if err != nil {
		return err
	}
	
	if len(submissions) < 2 {
		return fmt.Errorf("need at least 2 players for tournament")
	}
	
	seeded := SeedSubmissions(submissions)
	
	log.Printf("Tournament %d: Seeded %d players", tournament.ID, len(seeded))
	for i, sub := range seeded {
		log.Printf("  Seed %d: %s", i+1, sub.Username)
	}
	
	numPlayers := len(seeded)
	bracketSize := int(math.Pow(2, math.Ceil(math.Log2(float64(numPlayers)))))
	
	log.Printf("Tournament %d: Bracket size=%d, players=%d", 
		tournament.ID, bracketSize, numPlayers)
	
	numFirstRoundMatches := bracketSize / 2
	
	for matchPos := 0; matchPos < numFirstRoundMatches; matchPos++ {
		topSeedIdx := matchPos
		bottomSeedIdx := numPlayers - 1 - matchPos
		
		if topSeedIdx >= bottomSeedIdx && topSeedIdx < numPlayers && bottomSeedIdx >= 0 {
			if topSeedIdx == bottomSeedIdx {
				player1ID := seeded[topSeedIdx].ID
				player1Name := seeded[topSeedIdx].Username
				
				err = AddBracketMatch(tournament.ID, 1, matchPos, player1ID, 0)
				if err != nil {
					return err
				}
				
				DB.Exec(`
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
		
		if topSeedIdx < numPlayers {
			player1ID = seeded[topSeedIdx].ID
			player1Name = seeded[topSeedIdx].Username
		} else {
			player1ID = 0
			player1Name = "BYE"
		}
		
		if bottomSeedIdx >= 0 && bottomSeedIdx < numPlayers {
			player2ID = seeded[bottomSeedIdx].ID
			player2Name = seeded[bottomSeedIdx].Username
		} else {
			player2ID = 0
			player2Name = "BYE"
		}
		
		if player1ID == 0 && player2ID == 0 {
			continue
		}
		
		err = AddBracketMatch(tournament.ID, 1, matchPos, player1ID, player2ID)
		if err != nil {
			return err
		}
		
		if player1ID == 0 || player2ID == 0 {
			winnerID := player1ID
			if player1ID == 0 {
				winnerID = player2ID
			}
			
			DB.Exec(`
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

func AdvanceWinners(tournamentID, currentRound int) error {
	query := `
		SELECT id, position, winner_id 
		FROM bracket_matches 
		WHERE tournament_id = ? AND round = ? AND status = 'completed'
		ORDER BY position
	`
	
	rows, err := DB.Query(query, tournamentID, currentRound)
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
		log.Printf("Tournament %d complete! Winner: ID %d", tournamentID, winners[0].winnerID)
		return CompleteTournament(tournamentID, winners[0].winnerID)
	}
	
	nextRound := currentRound + 1
	log.Printf("Advancing to round %d with %d winners", nextRound, len(winners))
	
	for i := 0; i < len(winners); i += 2 {
		if i+1 >= len(winners) {
			log.Printf("  Round %d Match %d: BYE (winner: ID %d)", nextRound, i/2, winners[i].winnerID)
			err = AddBracketMatch(tournamentID, nextRound, i/2, winners[i].winnerID, 0)
			if err != nil {
				return err
			}
			DB.Exec(`
				UPDATE bracket_matches 
				SET winner_id = ?, status = 'completed',
				    player1_wins = 0, player2_wins = 0,
				    player1_moves = 0, player2_moves = 0
				WHERE tournament_id = ? AND round = ? AND position = ?
			`, winners[i].winnerID, tournamentID, nextRound, i/2)
		} else {
			log.Printf("  Round %d Match %d: ID %d vs ID %d", nextRound, i/2, winners[i].winnerID, winners[i+1].winnerID)
			err = AddBracketMatch(tournamentID, nextRound, i/2, winners[i].winnerID, winners[i+1].winnerID)
			if err != nil {
				return err
			}
		}
	}
	
	return UpdateTournamentRound(tournamentID, nextRound)
}

func EnsureTournamentExists() (*Tournament, error) {
	tournament, err := GetActiveTournament()
	if err != nil {
		return nil, err
	}
	
	if tournament != nil {
		return tournament, nil
	}
	
	latestTournament, err := GetLatestTournament()
	if err != nil {
		return nil, err
	}
	
	submissions, err := GetActiveSubmissions()
	if err != nil {
		return nil, err
	}
	
	if len(submissions) < 2 {
		log.Printf("Not enough players for tournament (%d/2), waiting...", len(submissions))
		return nil, fmt.Errorf("need at least 2 players")
	}
	
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
	tournament, err = CreateTournament()
	if err != nil {
		return nil, err
	}
	
	err = CreateBracket(tournament)
	if err != nil {
		return nil, err
	}
	log.Printf("Created tournament %d with bracket", tournament.ID)
	
	return tournament, nil
}

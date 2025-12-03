package main

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var globalDB *sql.DB

type LeaderboardEntry struct {
	Username  string
	Wins      int
	Losses    int
	AvgMoves  float64
	Stage     string
	LastPlayed time.Time
}

type Submission struct {
	ID         int
	Username   string
	Filename   string
	UploadTime time.Time
	Status     string // pending, testing, completed, failed
}

type Tournament struct {
	ID           int
	CreatedAt    time.Time
	Status       string // active, completed
	CurrentRound int
	WinnerID     int    // ID of winning submission
}

type BracketMatch struct {
	ID           int
	TournamentID int
	Round        int
	Position     int
	Player1ID    int
	Player2ID    int
	WinnerID     int
	Player1Wins  int
	Player2Wins  int
	Player1Moves int
	Player2Moves int
	Status       string // pending, in_progress, completed
	Player1Name  string // For display
	Player2Name  string
}

func initDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path+"?parseTime=true")
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS submissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL,
		filename TEXT NOT NULL,
		upload_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		status TEXT DEFAULT 'pending',
		is_active BOOLEAN DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS tournaments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		status TEXT DEFAULT 'active',
		current_round INTEGER DEFAULT 1,
		winner_id INTEGER,
		FOREIGN KEY (winner_id) REFERENCES submissions(id)
	);

	CREATE TABLE IF NOT EXISTS bracket_matches (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tournament_id INTEGER,
		round INTEGER,
		position INTEGER,
		player1_id INTEGER,
		player2_id INTEGER,
		winner_id INTEGER,
		player1_wins INTEGER DEFAULT 0,
		player2_wins INTEGER DEFAULT 0,
		player1_moves INTEGER,
		player2_moves INTEGER,
		status TEXT DEFAULT 'pending',
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (tournament_id) REFERENCES tournaments(id),
		FOREIGN KEY (player1_id) REFERENCES submissions(id),
		FOREIGN KEY (player2_id) REFERENCES submissions(id),
		FOREIGN KEY (winner_id) REFERENCES submissions(id)
	);

	CREATE TABLE IF NOT EXISTS matches (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		player1_id INTEGER,
		player2_id INTEGER,
		winner_id INTEGER,
		player1_moves INTEGER,
		player2_moves INTEGER,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (player1_id) REFERENCES submissions(id),
		FOREIGN KEY (player2_id) REFERENCES submissions(id),
		FOREIGN KEY (winner_id) REFERENCES submissions(id)
	);

	CREATE INDEX IF NOT EXISTS idx_bracket_matches_tournament ON bracket_matches(tournament_id);
	CREATE INDEX IF NOT EXISTS idx_bracket_matches_status ON bracket_matches(status);
	CREATE INDEX IF NOT EXISTS idx_tournaments_status ON tournaments(status);
	CREATE INDEX IF NOT EXISTS idx_matches_player1 ON matches(player1_id);
	CREATE INDEX IF NOT EXISTS idx_matches_player2 ON matches(player2_id);
	CREATE INDEX IF NOT EXISTS idx_submissions_username ON submissions(username);
	CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions(status);
	CREATE INDEX IF NOT EXISTS idx_submissions_active ON submissions(is_active);
	`

	_, err = db.Exec(schema)
	return db, err
}

func getLeaderboard(limit int) ([]LeaderboardEntry, error) {
	query := `
	SELECT 
		s.username,
		COUNT(CASE WHEN bm.winner_id = s.id THEN 1 END) as wins,
		COUNT(CASE WHEN (bm.player1_id = s.id OR bm.player2_id = s.id) AND bm.winner_id != s.id AND bm.winner_id IS NOT NULL THEN 1 END) as losses,
		AVG(CASE WHEN bm.player1_id = s.id THEN bm.player1_moves ELSE bm.player2_moves END) as avg_moves,
		MAX(bm.timestamp) as last_played
	FROM submissions s
	LEFT JOIN bracket_matches bm ON (bm.player1_id = s.id OR bm.player2_id = s.id) AND bm.status = 'completed'
	WHERE s.is_active = 1
	GROUP BY s.username
	HAVING COUNT(bm.id) > 0
	ORDER BY wins DESC, losses ASC, avg_moves ASC
	LIMIT ?
	`

	rows, err := globalDB.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		var lastPlayed string
		err := rows.Scan(&e.Username, &e.Wins, &e.Losses, &e.AvgMoves, &lastPlayed)
		if err != nil {
			return nil, err
		}
		
		// Parse the timestamp string
		e.LastPlayed, _ = time.Parse("2006-01-02 15:04:05", lastPlayed)
		
		// Determine stage based on average moves
		// Based on random AI benchmark: avg=95.459, p25=94, p75=99
		if e.AvgMoves >= 99 {
			e.Stage = "Beginner"
		} else if e.AvgMoves >= 95 {
			e.Stage = "Intermediate"
		} else if e.AvgMoves >= 85 {
			e.Stage = "Advanced"
		} else {
			e.Stage = "Expert"
		}
		
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

func addSubmission(username, filename string) (int64, error) {
	// Mark old submission as inactive
	_, err := globalDB.Exec(
		"UPDATE submissions SET is_active = 0 WHERE username = ?",
		username,
	)
	if err != nil {
		return 0, err
	}
	
	// Insert new submission
	result, err := globalDB.Exec(
		"INSERT INTO submissions (username, filename, is_active) VALUES (?, ?, 1)",
		username, filename,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func addMatch(player1ID, player2ID, winnerID, player1Moves, player2Moves int) error {
	_, err := globalDB.Exec(
		"INSERT INTO matches (player1_id, player2_id, winner_id, player1_moves, player2_moves) VALUES (?, ?, ?, ?, ?)",
		player1ID, player2ID, winnerID, player1Moves, player2Moves,
	)
	return err
}

func updateSubmissionStatus(id int, status string) error {
	_, err := globalDB.Exec("UPDATE submissions SET status = ? WHERE id = ?", status, id)
	return err
}

func getPendingSubmissions() ([]Submission, error) {
	rows, err := globalDB.Query(
		"SELECT id, username, filename, upload_time, status FROM submissions WHERE status = 'pending' AND is_active = 1 ORDER BY upload_time",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []Submission
	for rows.Next() {
		var s Submission
		err := rows.Scan(&s.ID, &s.Username, &s.Filename, &s.UploadTime, &s.Status)
		if err != nil {
			return nil, err
		}
		submissions = append(submissions, s)
	}

	return submissions, rows.Err()
}

func getActiveSubmissions() ([]Submission, error) {
	rows, err := globalDB.Query(
		"SELECT id, username, filename, upload_time, status FROM submissions WHERE is_active = 1 AND status = 'completed' ORDER BY username",
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []Submission
	for rows.Next() {
		var s Submission
		err := rows.Scan(&s.ID, &s.Username, &s.Filename, &s.UploadTime, &s.Status)
		if err != nil {
			return nil, err
		}
		submissions = append(submissions, s)
	}

	return submissions, rows.Err()
}

func getUserSubmissions(username string) ([]Submission, error) {
	rows, err := globalDB.Query(
		"SELECT id, username, filename, upload_time, status FROM submissions WHERE username = ? ORDER BY upload_time DESC LIMIT 10",
		username,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []Submission
	for rows.Next() {
		var s Submission
		err := rows.Scan(&s.ID, &s.Username, &s.Filename, &s.UploadTime, &s.Status)
		if err != nil {
			return nil, err
		}
		submissions = append(submissions, s)
	}

	return submissions, rows.Err()
}


type MatchResult struct {
	Player1Username string
	Player2Username string
	WinnerUsername  string
	AvgMoves        int
}

func getAllMatches() ([]MatchResult, error) {
	query := `
	SELECT 
		s1.username as player1,
		s2.username as player2,
		sw.username as winner,
		m.player1_moves as avg_moves
	FROM matches m
	JOIN submissions s1 ON m.player1_id = s1.id
	JOIN submissions s2 ON m.player2_id = s2.id
	JOIN submissions sw ON m.winner_id = sw.id
	WHERE s1.is_active = 1 AND s2.is_active = 1
	ORDER BY m.timestamp DESC
	`
	
	rows, err := globalDB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var matches []MatchResult
	for rows.Next() {
		var m MatchResult
		err := rows.Scan(&m.Player1Username, &m.Player2Username, &m.WinnerUsername, &m.AvgMoves)
		if err != nil {
			return nil, err
		}
		matches = append(matches, m)
	}
	
	return matches, rows.Err()
}

// Tournament functions

func getActiveTournament() (*Tournament, error) {
	var t Tournament
	var winnerID sql.NullInt64
	err := globalDB.QueryRow(
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

func getLatestTournament() (*Tournament, error) {
	var t Tournament
	var winnerID sql.NullInt64
	err := globalDB.QueryRow(
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

func createTournament() (*Tournament, error) {
	result, err := globalDB.Exec("INSERT INTO tournaments (status, current_round) VALUES ('active', 1)")
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

func updateTournamentRound(tournamentID, round int) error {
	_, err := globalDB.Exec("UPDATE tournaments SET current_round = ? WHERE id = ?", round, tournamentID)
	return err
}

func completeTournament(tournamentID, winnerID int) error {
	_, err := globalDB.Exec("UPDATE tournaments SET status = 'completed', winner_id = ? WHERE id = ?", winnerID, tournamentID)
	return err
}

func addBracketMatch(tournamentID, round, position, player1ID, player2ID int) error {
	_, err := globalDB.Exec(
		"INSERT INTO bracket_matches (tournament_id, round, position, player1_id, player2_id, status) VALUES (?, ?, ?, ?, ?, 'pending')",
		tournamentID, round, position, player1ID, player2ID,
	)
	return err
}

func getPendingBracketMatches(tournamentID int) ([]BracketMatch, error) {
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
	
	rows, err := globalDB.Query(query, tournamentID)
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

func getAllBracketMatches(tournamentID int) ([]BracketMatch, error) {
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
	
	rows, err := globalDB.Query(query, tournamentID)
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

func updateBracketMatchResult(matchID, winnerID, player1Wins, player2Wins, player1Moves, player2Moves int) error {
	_, err := globalDB.Exec(
		`UPDATE bracket_matches 
		SET winner_id = ?, player1_wins = ?, player2_wins = ?, 
		    player1_moves = ?, player2_moves = ?, status = 'completed' 
		WHERE id = ?`,
		winnerID, player1Wins, player2Wins, player1Moves, player2Moves, matchID,
	)
	return err
}

func isRoundComplete(tournamentID, round int) (bool, error) {
	var pendingCount int
	err := globalDB.QueryRow(
		"SELECT COUNT(*) FROM bracket_matches WHERE tournament_id = ? AND round = ? AND status != 'completed'",
		tournamentID, round,
	).Scan(&pendingCount)
	
	return pendingCount == 0, err
}


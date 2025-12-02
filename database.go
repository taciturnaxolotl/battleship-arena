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
	LastPlayed time.Time
}

type Submission struct {
	ID         int
	Username   string
	Filename   string
	UploadTime time.Time
	Status     string // pending, testing, completed, failed
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
		COUNT(CASE WHEN m.winner_id = s.id THEN 1 END) as wins,
		COUNT(CASE WHEN (m.player1_id = s.id OR m.player2_id = s.id) AND m.winner_id != s.id THEN 1 END) as losses,
		AVG(CASE WHEN m.player1_id = s.id THEN m.player1_moves ELSE m.player2_moves END) as avg_moves,
		MAX(m.timestamp) as last_played
	FROM submissions s
	LEFT JOIN matches m ON (m.player1_id = s.id OR m.player2_id = s.id)
	WHERE s.is_active = 1
	GROUP BY s.username
	HAVING COUNT(m.id) > 0
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

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
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS submissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL,
		filename TEXT NOT NULL,
		upload_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		status TEXT DEFAULT 'pending'
	);

	CREATE TABLE IF NOT EXISTS results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		submission_id INTEGER,
		opponent TEXT,
		result TEXT, -- win, loss, tie
		moves INTEGER,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (submission_id) REFERENCES submissions(id)
	);

	CREATE INDEX IF NOT EXISTS idx_results_submission ON results(submission_id);
	CREATE INDEX IF NOT EXISTS idx_submissions_username ON submissions(username);
	CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions(status);
	`

	_, err = db.Exec(schema)
	return db, err
}

func getLeaderboard(limit int) ([]LeaderboardEntry, error) {
	query := `
	SELECT 
		s.username,
		SUM(CASE WHEN r.result = 'win' THEN 1 ELSE 0 END) as wins,
		SUM(CASE WHEN r.result = 'loss' THEN 1 ELSE 0 END) as losses,
		AVG(r.moves) as avg_moves,
		MAX(r.timestamp) as last_played
	FROM submissions s
	JOIN results r ON s.id = r.submission_id
	GROUP BY s.username
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
		err := rows.Scan(&e.Username, &e.Wins, &e.Losses, &e.AvgMoves, &e.LastPlayed)
		if err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

func addSubmission(username, filename string) (int64, error) {
	result, err := globalDB.Exec(
		"INSERT INTO submissions (username, filename) VALUES (?, ?)",
		username, filename,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func addResult(submissionID int, opponent, result string, moves int) error {
	_, err := globalDB.Exec(
		"INSERT INTO results (submission_id, opponent, result, moves) VALUES (?, ?, ?, ?)",
		submissionID, opponent, result, moves,
	)
	return err
}

func updateSubmissionStatus(id int, status string) error {
	_, err := globalDB.Exec("UPDATE submissions SET status = ? WHERE id = ?", status, id)
	return err
}

func getPendingSubmissions() ([]Submission, error) {
	rows, err := globalDB.Query(
		"SELECT id, username, filename, upload_time, status FROM submissions WHERE status = 'pending' ORDER BY upload_time",
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

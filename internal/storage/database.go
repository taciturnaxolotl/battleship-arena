package storage

import (
	"database/sql"
	"math"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

type LeaderboardEntry struct {
	Username   string
	Wins       int
	Losses     int
	WinPct     float64
	Rating     int
	RD         int
	AvgMoves   float64
	Stage      string
	LastPlayed time.Time
	IsPending  bool
}

type Submission struct {
	ID         int
	Username   string
	Filename   string
	UploadTime time.Time
	Status     string
	IsActive   bool
}

type SubmissionWithStats struct {
	Submission
	Rating     int
	RD         int
	Wins       int
	Losses     int
	WinPct     float64
	AvgMoves   float64
	LastPlayed time.Time
	HasMatches bool
}

type Tournament struct {
	ID           int
	CreatedAt    time.Time
	Status       string
	CurrentRound int
	WinnerID     int
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
	Status       string
	Player1Name  string
	Player2Name  string
}

type MatchResult struct {
	Player1Username string
	Player2Username string
	WinnerUsername  string
	AvgMoves        int
}

type RatingHistoryPoint struct {
	Rating     int
	RD         int
	Volatility float64
	Timestamp  time.Time
	MatchID    int
}

func InitDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", path+"?parseTime=true")
	if err != nil {
		return nil, err
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		bio TEXT,
		link TEXT,
		public_key TEXT UNIQUE NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_login_at TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS submissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL,
		filename TEXT NOT NULL,
		upload_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		status TEXT DEFAULT 'pending',
		is_active BOOLEAN DEFAULT 1,
		glicko_rating REAL DEFAULT 1500.0,
		glicko_rd REAL DEFAULT 350.0,
		glicko_volatility REAL DEFAULT 0.06
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
		player1_wins INTEGER DEFAULT 0,
		player2_wins INTEGER DEFAULT 0,
		player1_moves INTEGER,
		player2_moves INTEGER,
		is_valid BOOLEAN DEFAULT 1,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (player1_id) REFERENCES submissions(id),
		FOREIGN KEY (player2_id) REFERENCES submissions(id),
		FOREIGN KEY (winner_id) REFERENCES submissions(id)
	);

	CREATE TABLE IF NOT EXISTS rating_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		submission_id INTEGER NOT NULL,
		rating REAL NOT NULL,
		rd REAL NOT NULL,
		volatility REAL NOT NULL,
		match_id INTEGER,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (submission_id) REFERENCES submissions(id),
		FOREIGN KEY (match_id) REFERENCES matches(id)
	);

	CREATE INDEX IF NOT EXISTS idx_bracket_matches_tournament ON bracket_matches(tournament_id);
	CREATE INDEX IF NOT EXISTS idx_bracket_matches_status ON bracket_matches(status);
	CREATE INDEX IF NOT EXISTS idx_tournaments_status ON tournaments(status);
	CREATE INDEX IF NOT EXISTS idx_matches_player1 ON matches(player1_id);
	CREATE INDEX IF NOT EXISTS idx_matches_player2 ON matches(player2_id);
	CREATE INDEX IF NOT EXISTS idx_matches_valid ON matches(is_valid);
	CREATE INDEX IF NOT EXISTS idx_submissions_username ON submissions(username);
	CREATE INDEX IF NOT EXISTS idx_submissions_status ON submissions(status);
	CREATE INDEX IF NOT EXISTS idx_submissions_active ON submissions(is_active);
	CREATE UNIQUE INDEX IF NOT EXISTS idx_matches_unique_pair ON matches(player1_id, player2_id, is_valid) WHERE is_valid = 1;
	CREATE INDEX IF NOT EXISTS idx_rating_history_submission ON rating_history(submission_id, timestamp);
	`

	_, err = db.Exec(schema)
	return db, err
}

func GetLeaderboard(limit int) ([]LeaderboardEntry, error) {
	// Get submissions with matches
	query := `
	SELECT 
		s.username,
		COALESCE(s.glicko_rating, 1500.0) as rating,
		COALESCE(s.glicko_rd, 350.0) as rd,
		SUM(CASE WHEN m.player1_id = s.id THEN m.player1_wins WHEN m.player2_id = s.id THEN m.player2_wins ELSE 0 END) as total_wins,
		SUM(CASE WHEN m.player1_id = s.id THEN m.player2_wins WHEN m.player2_id = s.id THEN m.player1_wins ELSE 0 END) as total_losses,
		AVG(CASE WHEN m.player1_id = s.id THEN m.player1_moves ELSE m.player2_moves END) as avg_moves,
		MAX(m.timestamp) as last_played,
		0 as is_pending
	FROM submissions s
	LEFT JOIN matches m ON (m.player1_id = s.id OR m.player2_id = s.id) AND m.is_valid = 1
	WHERE s.is_active = 1
	GROUP BY s.username, s.glicko_rating, s.glicko_rd
	HAVING COUNT(m.id) > 0
	
	UNION ALL
	
	SELECT 
		s.username,
		1500.0 as rating,
		350.0 as rd,
		0 as total_wins,
		0 as total_losses,
		0.0 as avg_moves,
		s.upload_time as last_played,
		1 as is_pending
	FROM submissions s
	LEFT JOIN matches m ON (m.player1_id = s.id OR m.player2_id = s.id) AND m.is_valid = 1
	WHERE s.is_active = 1 AND s.status IN ('pending', 'testing')
	GROUP BY s.username, s.upload_time
	HAVING COUNT(m.id) = 0
	
	ORDER BY is_pending ASC, rating DESC, total_wins DESC
	LIMIT ?
	`

	rows, err := DB.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []LeaderboardEntry
	for rows.Next() {
		var e LeaderboardEntry
		var lastPlayed string
		var rating, rd float64
		var isPending int
		err := rows.Scan(&e.Username, &rating, &rd, &e.Wins, &e.Losses, &e.AvgMoves, &lastPlayed, &isPending)
		if err != nil {
			return nil, err
		}
		
		e.Rating = int(rating)
		e.RD = int(rd)
		e.IsPending = isPending == 1
		
		totalGames := e.Wins + e.Losses
		if totalGames > 0 {
			e.WinPct = float64(e.Wins) / float64(totalGames) * 100.0
		}
		
		e.LastPlayed, _ = time.Parse("2006-01-02 15:04:05", lastPlayed)
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

func AddSubmission(username, filename string) (int64, error) {
	_, err := DB.Exec(
		`UPDATE matches SET is_valid = 0 
		 WHERE player1_id IN (SELECT id FROM submissions WHERE username = ?)
		 OR player2_id IN (SELECT id FROM submissions WHERE username = ?)`,
		username, username,
	)
	if err != nil {
		return 0, err
	}
	
	_, err = DB.Exec(
		"UPDATE submissions SET is_active = 0 WHERE username = ?",
		username,
	)
	if err != nil {
		return 0, err
	}
	
	result, err := DB.Exec(
		"INSERT INTO submissions (username, filename, is_active, glicko_rating, glicko_rd, glicko_volatility) VALUES (?, ?, 1, 1500.0, 350.0, 0.06)",
		username, filename,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func AddMatch(player1ID, player2ID, winnerID, player1Wins, player2Wins, player1Moves, player2Moves int) (int64, error) {
	result, err := DB.Exec(
		"INSERT INTO matches (player1_id, player2_id, winner_id, player1_wins, player2_wins, player1_moves, player2_moves) VALUES (?, ?, ?, ?, ?, ?, ?)",
		player1ID, player2ID, winnerID, player1Wins, player2Wins, player1Moves, player2Moves,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func UpdateSubmissionStatus(id int, status string) error {
	_, err := DB.Exec("UPDATE submissions SET status = ? WHERE id = ?", status, id)
	return err
}

func GetPendingSubmissions() ([]Submission, error) {
	rows, err := DB.Query(
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

func GetActiveSubmissions() ([]Submission, error) {
	rows, err := DB.Query(
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

func GetUserSubmissions(username string) ([]Submission, error) {
	rows, err := DB.Query(
		"SELECT id, username, filename, upload_time, status, is_active FROM submissions WHERE username = ? ORDER BY upload_time DESC LIMIT 10",
		username,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []Submission
	for rows.Next() {
		var s Submission
		err := rows.Scan(&s.ID, &s.Username, &s.Filename, &s.UploadTime, &s.Status, &s.IsActive)
		if err != nil {
			return nil, err
		}
		submissions = append(submissions, s)
	}

	return submissions, rows.Err()
}

func GetUserSubmissionsWithStats(username string) ([]SubmissionWithStats, error) {
	query := `
	SELECT 
		s.id,
		s.username,
		s.filename,
		s.upload_time,
		s.status,
		s.is_active,
		COALESCE(s.glicko_rating, 1500.0) as rating,
		COALESCE(s.glicko_rd, 350.0) as rd,
		COALESCE(SUM(CASE WHEN m.player1_id = s.id THEN m.player1_wins WHEN m.player2_id = s.id THEN m.player2_wins ELSE 0 END), 0) as total_wins,
		COALESCE(SUM(CASE WHEN m.player1_id = s.id THEN m.player2_wins WHEN m.player2_id = s.id THEN m.player1_wins ELSE 0 END), 0) as total_losses,
		COALESCE(AVG(CASE WHEN m.player1_id = s.id THEN m.player1_moves ELSE m.player2_moves END), 0) as avg_moves,
		MAX(m.timestamp) as last_played,
		COUNT(m.id) as match_count
	FROM submissions s
	LEFT JOIN matches m ON (m.player1_id = s.id OR m.player2_id = s.id) AND m.is_valid = 1
	WHERE s.username = ?
	GROUP BY s.id, s.username, s.filename, s.upload_time, s.status, s.is_active, s.glicko_rating, s.glicko_rd
	ORDER BY s.upload_time DESC
	LIMIT 10
	`
	
	rows, err := DB.Query(query, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var submissions []SubmissionWithStats
	for rows.Next() {
		var s SubmissionWithStats
		var lastPlayed *string
		var rating, rd float64
		var matchCount int
		
		err := rows.Scan(
			&s.ID, &s.Username, &s.Filename, &s.UploadTime, &s.Status, &s.IsActive,
			&rating, &rd, &s.Wins, &s.Losses, &s.AvgMoves, &lastPlayed, &matchCount,
		)
		if err != nil {
			return nil, err
		}
		
		s.Rating = int(rating)
		s.RD = int(rd)
		s.HasMatches = matchCount > 0
		
		totalGames := s.Wins + s.Losses
		if totalGames > 0 {
			s.WinPct = float64(s.Wins) / float64(totalGames) * 100.0
		}
		
		if lastPlayed != nil {
			s.LastPlayed, _ = time.Parse("2006-01-02 15:04:05", *lastPlayed)
		}
		
		submissions = append(submissions, s)
	}

	return submissions, rows.Err()
}

func GetSubmissionByID(id int) (Submission, error) {
	var sub Submission
	err := DB.QueryRow(
		"SELECT id, username, filename, upload_time, status FROM submissions WHERE id = ?",
		id,
	).Scan(&sub.ID, &sub.Username, &sub.Filename, &sub.UploadTime, &sub.Status)
	return sub, err
}

func HasMatchBetween(player1ID, player2ID int) (bool, error) {
	var count int
	err := DB.QueryRow(
		`SELECT COUNT(*) FROM matches 
		 WHERE is_valid = 1 
		 AND ((player1_id = ? AND player2_id = ?) OR (player1_id = ? AND player2_id = ?))`,
		player1ID, player2ID, player2ID, player1ID,
	).Scan(&count)
	return count > 0, err
}

func GetAllMatches() ([]MatchResult, error) {
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
	WHERE s1.is_active = 1 AND s2.is_active = 1 AND m.is_valid = 1
	ORDER BY m.timestamp DESC
	`
	
	rows, err := DB.Query(query)
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

func RecordRatingHistory(submissionID int, matchID int, rating, rd, volatility float64) error {
	_, err := DB.Exec(
		"INSERT INTO rating_history (submission_id, match_id, rating, rd, volatility) VALUES (?, ?, ?, ?, ?)",
		submissionID, matchID, rating, rd, volatility,
	)
	return err
}

func GetRatingHistory(submissionID int) ([]RatingHistoryPoint, error) {
	rows, err := DB.Query(`
		SELECT rating, rd, volatility, timestamp, match_id 
		FROM rating_history 
		WHERE submission_id = ? 
		ORDER BY timestamp ASC
	`, submissionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	var history []RatingHistoryPoint
	for rows.Next() {
		var h RatingHistoryPoint
		var rating, rd float64
		var matchID sql.NullInt64
		err := rows.Scan(&rating, &rd, &h.Volatility, &h.Timestamp, &matchID)
		if err != nil {
			return nil, err
		}
		h.Rating = int(rating)
		h.RD = int(rd)
		if matchID.Valid {
			h.MatchID = int(matchID.Int64)
		}
		history = append(history, h)
	}
	
	return history, rows.Err()
}

func GetQueuedPlayerNames() []string {
	rows, err := DB.Query(
		"SELECT username FROM submissions WHERE (status = 'pending' OR status = 'testing') AND is_active = 1 ORDER BY upload_time",
	)
	if err != nil {
		return []string{}
	}
	defer rows.Close()
	
	var names []string
	for rows.Next() {
		var username string
		if err := rows.Scan(&username); err == nil {
			names = append(names, username)
		}
	}
	return names
}

// Glicko-2 constants
const (
	glickoTau     = 0.5
	glickoEpsilon = 0.000001
	glicko2Scale  = 173.7178
)

type Glicko2Player struct {
	Rating     float64
	RD         float64
	Volatility float64
}

type Glicko2Result struct {
	OpponentRating float64
	OpponentRD     float64
	Score          float64
}

func toGlicko2Scale(rating, rd float64) (float64, float64) {
	return (rating - 1500.0) / glicko2Scale, rd / glicko2Scale
}

func fromGlicko2Scale(mu, phi float64) (float64, float64) {
	return mu*glicko2Scale + 1500.0, phi * glicko2Scale
}

func g(phi float64) float64 {
	return 1.0 / math.Sqrt(1.0+3.0*phi*phi/(math.Pi*math.Pi))
}

func eFunc(mu, muJ, phiJ float64) float64 {
	return 1.0 / (1.0 + math.Exp(-g(phiJ)*(mu-muJ)))
}

func updateGlicko2(player Glicko2Player, results []Glicko2Result) Glicko2Player {
	mu, phi := toGlicko2Scale(player.Rating, player.RD)
	sigma := player.Volatility
	
	if len(results) == 0 {
		phiStar := math.Sqrt(phi*phi + sigma*sigma)
		rating, rd := fromGlicko2Scale(mu, phiStar)
		return Glicko2Player{Rating: rating, RD: rd, Volatility: sigma}
	}
	
	var vInv float64
	for _, result := range results {
		muJ, phiJ := toGlicko2Scale(result.OpponentRating, result.OpponentRD)
		gPhiJ := g(phiJ)
		eVal := eFunc(mu, muJ, phiJ)
		vInv += gPhiJ * gPhiJ * eVal * (1.0 - eVal)
	}
	v := 1.0 / vInv
	
	var delta float64
	for _, result := range results {
		muJ, phiJ := toGlicko2Scale(result.OpponentRating, result.OpponentRD)
		gPhiJ := g(phiJ)
		eVal := eFunc(mu, muJ, phiJ)
		delta += gPhiJ * (result.Score - eVal)
	}
	delta *= v
	
	a := math.Log(sigma * sigma)
	deltaSquared := delta * delta
	phiSquared := phi * phi
	
	fFunc := func(x float64) float64 {
		eX := math.Exp(x)
		num := eX * (deltaSquared - phiSquared - v - eX)
		denom := 2.0 * (phiSquared + v + eX) * (phiSquared + v + eX)
		return num/denom - (x-a)/(glickoTau*glickoTau)
	}
	
	A := a
	var B float64
	if deltaSquared > phiSquared+v {
		B = math.Log(deltaSquared - phiSquared - v)
	} else {
		k := 1.0
		for fFunc(a-k*glickoTau) < 0 {
			k++
		}
		B = a - k*glickoTau
	}
	
	fA := fFunc(A)
	fB := fFunc(B)
	
	for math.Abs(B-A) > glickoEpsilon {
		C := A + (A-B)*fA/(fB-fA)
		fC := fFunc(C)
		
		if fC*fB < 0 {
			A = B
			fA = fB
		} else {
			fA = fA / 2.0
		}
		
		B = C
		fB = fC
	}
	
	sigmaNew := math.Exp(A / 2.0)
	phiStar := math.Sqrt(phiSquared + sigmaNew*sigmaNew)
	phiNew := 1.0 / math.Sqrt(1.0/(phiStar*phiStar)+1.0/v)
	
	var muNew float64
	for _, result := range results {
		muJ, phiJ := toGlicko2Scale(result.OpponentRating, result.OpponentRD)
		muNew += g(phiJ) * (result.Score - eFunc(mu, muJ, phiJ))
	}
	muNew = mu + phiNew*phiNew*muNew
	
	rating, rd := fromGlicko2Scale(muNew, phiNew)
	return Glicko2Player{Rating: rating, RD: rd, Volatility: sigmaNew}
}

func UpdateGlicko2Ratings(player1ID, player2ID, player1Wins, player2Wins int) error {
	var p1Rating, p1RD, p1Vol, p2Rating, p2RD, p2Vol float64
	
	err := DB.QueryRow(
		"SELECT COALESCE(glicko_rating, 1500.0), COALESCE(glicko_rd, 350.0), COALESCE(glicko_volatility, 0.06) FROM submissions WHERE id = ?",
		player1ID,
	).Scan(&p1Rating, &p1RD, &p1Vol)
	if err != nil {
		return err
	}
	
	err = DB.QueryRow(
		"SELECT COALESCE(glicko_rating, 1500.0), COALESCE(glicko_rd, 350.0), COALESCE(glicko_volatility, 0.06) FROM submissions WHERE id = ?",
		player2ID,
	).Scan(&p2Rating, &p2RD, &p2Vol)
	if err != nil {
		return err
	}
	
	totalGames := player1Wins + player2Wins
	player1Score := float64(player1Wins) / float64(totalGames)
	player2Score := float64(player2Wins) / float64(totalGames)
	
	p1 := Glicko2Player{Rating: p1Rating, RD: p1RD, Volatility: p1Vol}
	p1Results := []Glicko2Result{{OpponentRating: p2Rating, OpponentRD: p2RD, Score: player1Score}}
	p1New := updateGlicko2(p1, p1Results)
	
	p2 := Glicko2Player{Rating: p2Rating, RD: p2RD, Volatility: p2Vol}
	p2Results := []Glicko2Result{{OpponentRating: p1Rating, OpponentRD: p1RD, Score: player2Score}}
	p2New := updateGlicko2(p2, p2Results)
	
	_, err = DB.Exec(
		"UPDATE submissions SET glicko_rating = ?, glicko_rd = ?, glicko_volatility = ? WHERE id = ?",
		p1New.Rating, p1New.RD, p1New.Volatility, player1ID,
	)
	if err != nil {
		return err
	}
	
	_, err = DB.Exec(
		"UPDATE submissions SET glicko_rating = ?, glicko_rd = ?, glicko_volatility = ? WHERE id = ?",
		p2New.Rating, p2New.RD, p2New.Volatility, player2ID,
	)
	return err
}

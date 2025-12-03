package main

import (
	"database/sql"
	"math"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var globalDB *sql.DB

type LeaderboardEntry struct {
	Username   string
	Wins       int
	Losses     int
	WinPct     float64
	Rating     int     // Glicko-2 rating
	RD         int     // Rating Deviation (uncertainty)
	AvgMoves   float64
	Stage      string
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
	`

	_, err = db.Exec(schema)
	return db, err
}

func getLeaderboard(limit int) ([]LeaderboardEntry, error) {
	query := `
	SELECT 
		s.username,
		s.glicko_rating,
		s.glicko_rd,
		SUM(CASE WHEN m.player1_id = s.id THEN m.player1_wins WHEN m.player2_id = s.id THEN m.player2_wins ELSE 0 END) as total_wins,
		SUM(CASE WHEN m.player1_id = s.id THEN m.player2_wins WHEN m.player2_id = s.id THEN m.player1_wins ELSE 0 END) as total_losses,
		AVG(CASE WHEN m.player1_id = s.id THEN m.player1_moves ELSE m.player2_moves END) as avg_moves,
		MAX(m.timestamp) as last_played
	FROM submissions s
	LEFT JOIN matches m ON (m.player1_id = s.id OR m.player2_id = s.id) AND m.is_valid = 1
	WHERE s.is_active = 1
	GROUP BY s.username, s.glicko_rating, s.glicko_rd
	HAVING COUNT(m.id) > 0
	ORDER BY s.glicko_rating DESC, total_wins DESC
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
		var rating, rd float64
		err := rows.Scan(&e.Username, &rating, &rd, &e.Wins, &e.Losses, &e.AvgMoves, &lastPlayed)
		if err != nil {
			return nil, err
		}
		
		e.Rating = int(rating)
		e.RD = int(rd)
		
		// Calculate win percentage
		totalGames := e.Wins + e.Losses
		if totalGames > 0 {
			e.WinPct = float64(e.Wins) / float64(totalGames) * 100.0
		}
		
		// Parse the timestamp string
		e.LastPlayed, _ = time.Parse("2006-01-02 15:04:05", lastPlayed)
		
		entries = append(entries, e)
	}

	return entries, rows.Err()
}

func addSubmission(username, filename string) (int64, error) {
	// Invalidate all matches involving this user's submissions
	_, err := globalDB.Exec(
		`UPDATE matches SET is_valid = 0 
		 WHERE player1_id IN (SELECT id FROM submissions WHERE username = ?)
		 OR player2_id IN (SELECT id FROM submissions WHERE username = ?)`,
		username, username,
	)
	if err != nil {
		return 0, err
	}
	
	// Mark old submission as inactive
	_, err = globalDB.Exec(
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

func addMatch(player1ID, player2ID, winnerID, player1Wins, player2Wins, player1Moves, player2Moves int) error {
	_, err := globalDB.Exec(
		"INSERT INTO matches (player1_id, player2_id, winner_id, player1_wins, player2_wins, player1_moves, player2_moves) VALUES (?, ?, ?, ?, ?, ?, ?)",
		player1ID, player2ID, winnerID, player1Wins, player2Wins, player1Moves, player2Moves,
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

// Glicko-2 rating system implementation
// Based on Mark Glickman's paper: http://www.glicko.net/glicko/glicko2.pdf

const (
	glickoTau        = 0.5    // System constant (volatility change constraint)
	glickoEpsilon    = 0.000001 // Convergence tolerance
	glicko2Scale     = 173.7178 // Conversion factor: rating / 173.7178
)

type Glicko2Player struct {
	Rating     float64 // μ in Glicko-2 scale
	RD         float64 // φ in Glicko-2 scale  
	Volatility float64 // σ
}

type Glicko2Result struct {
	OpponentRating float64
	OpponentRD     float64
	Score          float64 // 0.0 to 1.0
}

// Convert rating from standard scale to Glicko-2 scale
func toGlicko2Scale(rating, rd float64) (float64, float64) {
	return (rating - 1500.0) / glicko2Scale, rd / glicko2Scale
}

// Convert rating from Glicko-2 scale to standard scale
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
	// Step 2: Convert to Glicko-2 scale
	mu, phi := toGlicko2Scale(player.Rating, player.RD)
	sigma := player.Volatility
	
	if len(results) == 0 {
		// No games played - increase RD due to inactivity
		phiStar := math.Sqrt(phi*phi + sigma*sigma)
		rating, rd := fromGlicko2Scale(mu, phiStar)
		return Glicko2Player{Rating: rating, RD: rd, Volatility: sigma}
	}
	
	// Step 3: Compute v (variance)
	var vInv float64
	for _, result := range results {
		muJ, phiJ := toGlicko2Scale(result.OpponentRating, result.OpponentRD)
		gPhiJ := g(phiJ)
		eVal := eFunc(mu, muJ, phiJ)
		vInv += gPhiJ * gPhiJ * eVal * (1.0 - eVal)
	}
	v := 1.0 / vInv
	
	// Step 4: Compute delta (improvement)
	var delta float64
	for _, result := range results {
		muJ, phiJ := toGlicko2Scale(result.OpponentRating, result.OpponentRD)
		gPhiJ := g(phiJ)
		eVal := eFunc(mu, muJ, phiJ)
		delta += gPhiJ * (result.Score - eVal)
	}
	delta *= v
	
	// Step 5: Determine new volatility using Illinois algorithm
	a := math.Log(sigma * sigma)
	
	deltaSquared := delta * delta
	phiSquared := phi * phi
	
	fFunc := func(x float64) float64 {
		eX := math.Exp(x)
		num := eX * (deltaSquared - phiSquared - v - eX)
		denom := 2.0 * (phiSquared + v + eX) * (phiSquared + v + eX)
		return num/denom - (x-a)/(glickoTau*glickoTau)
	}
	
	// Find bounds
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
	
	// Illinois algorithm iteration
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
	
	// Step 6: Update rating deviation
	phiStar := math.Sqrt(phiSquared + sigmaNew*sigmaNew)
	
	// Step 7: Update rating and RD
	phiNew := 1.0 / math.Sqrt(1.0/(phiStar*phiStar)+1.0/v)
	
	var muNew float64
	for _, result := range results {
		muJ, phiJ := toGlicko2Scale(result.OpponentRating, result.OpponentRD)
		muNew += g(phiJ) * (result.Score - eFunc(mu, muJ, phiJ))
	}
	muNew = mu + phiNew*phiNew*muNew
	
	// Step 8: Convert back to standard scale
	rating, rd := fromGlicko2Scale(muNew, phiNew)
	
	return Glicko2Player{Rating: rating, RD: rd, Volatility: sigmaNew}
}

func updateGlicko2Ratings(player1ID, player2ID, player1Wins, player2Wins int) error {
	// Get current Glicko-2 values for both players
	var p1Rating, p1RD, p1Vol, p2Rating, p2RD, p2Vol float64
	
	err := globalDB.QueryRow(
		"SELECT glicko_rating, glicko_rd, glicko_volatility FROM submissions WHERE id = ?",
		player1ID,
	).Scan(&p1Rating, &p1RD, &p1Vol)
	if err != nil {
		return err
	}
	
	err = globalDB.QueryRow(
		"SELECT glicko_rating, glicko_rd, glicko_volatility FROM submissions WHERE id = ?",
		player2ID,
	).Scan(&p2Rating, &p2RD, &p2Vol)
	if err != nil {
		return err
	}
	
	// Calculate scores
	totalGames := player1Wins + player2Wins
	player1Score := float64(player1Wins) / float64(totalGames)
	player2Score := float64(player2Wins) / float64(totalGames)
	
	// Update player 1
	p1 := Glicko2Player{Rating: p1Rating, RD: p1RD, Volatility: p1Vol}
	p1Results := []Glicko2Result{{OpponentRating: p2Rating, OpponentRD: p2RD, Score: player1Score}}
	p1New := updateGlicko2(p1, p1Results)
	
	// Update player 2
	p2 := Glicko2Player{Rating: p2Rating, RD: p2RD, Volatility: p2Vol}
	p2Results := []Glicko2Result{{OpponentRating: p1Rating, OpponentRD: p1RD, Score: player2Score}}
	p2New := updateGlicko2(p2, p2Results)
	
	// Save updated ratings
	_, err = globalDB.Exec(
		"UPDATE submissions SET glicko_rating = ?, glicko_rd = ?, glicko_volatility = ? WHERE id = ?",
		p1New.Rating, p1New.RD, p1New.Volatility, player1ID,
	)
	if err != nil {
		return err
	}
	
	_, err = globalDB.Exec(
		"UPDATE submissions SET glicko_rating = ?, glicko_rd = ?, glicko_volatility = ? WHERE id = ?",
		p2New.Rating, p2New.RD, p2New.Volatility, player2ID,
	)
	return err
}

func hasMatchBetween(player1ID, player2ID int) (bool, error) {
	var count int
	err := globalDB.QueryRow(
		`SELECT COUNT(*) FROM matches 
		 WHERE is_valid = 1 
		 AND ((player1_id = ? AND player2_id = ?) OR (player1_id = ? AND player2_id = ?))`,
		player1ID, player2ID, player2ID, player1ID,
	).Scan(&count)
	return count > 0, err
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
	WHERE s1.is_active = 1 AND s2.is_active = 1 AND m.is_valid = 1
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


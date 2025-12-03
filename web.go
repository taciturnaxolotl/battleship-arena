package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
)

const leaderboardHTML = `
<!DOCTYPE html>
<html>
<head>
    <title>Battleship Arena - Leaderboard</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="stylesheet" href="/static/brackets-viewer.min.css" />
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif;
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
        }
        .container {
            background: white;
            border-radius: 12px;
            box-shadow: 0 20px 60px rgba(0,0,0,0.3);
            padding: 40px;
        }
        h1 {
            color: #333;
            text-align: center;
            margin-bottom: 10px;
            font-size: 2.5em;
        }
        .subtitle {
            text-align: center;
            color: #666;
            margin-bottom: 40px;
            font-size: 1.1em;
        }
        table {
            width: 100%;
            border-collapse: collapse;
            margin-top: 20px;
        }
        th {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 15px;
            text-align: left;
            font-weight: 600;
        }
        td {
            padding: 12px 15px;
            border-bottom: 1px solid #eee;
        }
        tr:hover {
            background: #f8f9fa;
        }
        .rank {
            font-weight: bold;
            font-size: 1.1em;
        }
        .rank-1 { color: #FFD700; }
        .rank-2 { color: #C0C0C0; }
        .rank-3 { color: #CD7F32; }
        .win-rate {
            font-weight: 600;
        }
        .win-rate-high { color: #10b981; }
        .win-rate-med { color: #f59e0b; }
        .win-rate-low { color: #ef4444; }
        .stage {
            display: inline-block;
            padding: 4px 12px;
            border-radius: 12px;
            font-size: 0.85em;
            font-weight: 600;
        }
        .stage-Expert {
            background: #10b981;
            color: white;
        }
        .stage-Advanced {
            background: #3b82f6;
            color: white;
        }
        .stage-Intermediate {
            background: #f59e0b;
            color: white;
        }
        .stage-Beginner {
            background: #6b7280;
            color: white;
        }
        .stats {
            display: flex;
            justify-content: space-around;
            margin-top: 40px;
            padding-top: 30px;
            border-top: 2px solid #eee;
        }
        .stat {
            text-align: center;
        }
        .stat-value {
            font-size: 2em;
            font-weight: bold;
            color: #667eea;
        }
        .stat-label {
            color: #666;
            margin-top: 5px;
        }
        .instructions {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 8px;
            margin-top: 30px;
        }
        .instructions h3 {
            margin-top: 0;
            color: #333;
        }
        .instructions code {
            background: #e9ecef;
            padding: 3px 8px;
            border-radius: 4px;
            font-family: 'Monaco', 'Courier New', monospace;
        }
        .refresh-note {
            text-align: center;
            color: #999;
            font-size: 0.9em;
            margin-top: 20px;
        }
        .bracket-section {
            margin: 40px 0;
            background: white;
            padding: 20px;
            border-radius: 12px;
        }
        .bracket-section h2 {
            text-align: center;
            color: #333;
            margin-bottom: 30px;
        }
    </style>
    <script type="text/javascript" src="/static/brackets-viewer.min.js"></script>
    <script>
        // Auto-refresh every 30 seconds
        setTimeout(() => location.reload(), 30000);
        
        // Load and render bracket data
        window.addEventListener('DOMContentLoaded', async () => {
            try {
                const response = await fetch('/api/bracket');
                const data = await response.json();
                
                if (data.matches && data.matches.length > 0) {
                    window.bracketsViewer.render({
                        stages: data.stages,
                        matches: data.matches,
                        matchGames: data.matchGames,
                        participants: data.participants,
                    });
                }
            } catch (error) {
                console.error('Failed to load bracket data:', error);
            }
        });
    </script>
</head>
<body>
    <div class="container">
        <h1>🚢 Battleship Arena</h1>
        <p class="subtitle">Smart AI Competition</p>
        
        <div class="bracket-section">
            <h2>⚔️ Tournament Bracket</h2>
            <div class="brackets-viewer"></div>
        </div>
        
        <h2 style="text-align: center; color: #333; margin-top: 60px;">📊 Rankings</h2>
        <table>
            <thead>
                <tr>
                    <th>Rank</th>
                    <th>Player</th>
                    <th>Stage</th>
                    <th>Wins</th>
                    <th>Losses</th>
                    <th>Win Rate</th>
                    <th>Avg Moves</th>
                    <th>Last Played</th>
                </tr>
            </thead>
            <tbody>
                {{if .Entries}}
                {{range $i, $e := .Entries}}
                <tr>
                    <td class="rank rank-{{add $i 1}}">{{if lt $i 3}}{{medal $i}}{{else}}#{{add $i 1}}{{end}}</td>
                    <td><strong>{{$e.Username}}</strong></td>
                    <td><span class="stage stage-{{$e.Stage}}">{{$e.Stage}}</span></td>
                    <td>{{$e.Wins}}</td>
                    <td>{{$e.Losses}}</td>
                    <td class="win-rate {{winRateClass $e}}">{{winRate $e}}%</td>
                    <td>{{printf "%.1f" $e.AvgMoves}}</td>
                    <td>{{$e.LastPlayed.Format "Jan 2, 3:04 PM"}}</td>
                </tr>
                {{end}}
                {{else}}
                <tr>
                    <td colspan="8" style="text-align: center; padding: 40px; color: #999;">
                        No submissions yet. Be the first to compete!
                    </td>
                </tr>
                {{end}}
            </tbody>
        </table>

        <div class="stats">
            <div class="stat">
                <div class="stat-value">{{.TotalPlayers}}</div>
                <div class="stat-label">Players</div>
            </div>
            <div class="stat">
                <div class="stat-value">{{.TotalGames}}</div>
                <div class="stat-label">Games Played</div>
            </div>
        </div>

        <div class="instructions">
            <h3>📤 How to Submit</h3>
            <p>Upload your battleship AI implementation via SSH:</p>
            <code>ssh -p 2222 username@localhost</code>
            <p style="margin-top: 10px;">Then navigate to upload your <code>memory_functions_*.cpp</code> file.</p>
        </div>

        <p class="refresh-note">Page auto-refreshes every 30 seconds</p>
    </div>
</body>
</html>
`

var tmpl = template.Must(template.New("leaderboard").Funcs(template.FuncMap{
	"add": func(a, b int) int {
		return a + b
	},
	"medal": func(i int) string {
		medals := []string{"🥇", "🥈", "🥉"}
		if i < len(medals) {
			return medals[i]
		}
		return ""
	},
	"winRate": func(e LeaderboardEntry) string {
		total := e.Wins + e.Losses
		if total == 0 {
			return "0.0"
		}
		rate := float64(e.Wins) / float64(total) * 100
		return formatFloat(rate, 1)
	},
	"winRateClass": func(e LeaderboardEntry) string {
		total := e.Wins + e.Losses
		if total == 0 {
			return "win-rate-low"
		}
		rate := float64(e.Wins) / float64(total) * 100
		if rate >= 80 {
			return "win-rate-high"
		} else if rate >= 50 {
			return "win-rate-med"
		}
		return "win-rate-low"
	},
}).Parse(leaderboardHTML))

func formatFloat(f float64, decimals int) string {
	return fmt.Sprintf("%.1f", f)
}

func handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := getLeaderboard(50)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load leaderboard: %v", err), http.StatusInternalServerError)
		return
	}

	// Empty leaderboard is fine
	if entries == nil {
		entries = []LeaderboardEntry{}
	}
	
	// Get matches for bracket
	matches, err := getAllMatches()
	if err != nil {
		matches = []MatchResult{}
	}

	data := struct {
		Entries      []LeaderboardEntry
		Matches      []MatchResult
		TotalPlayers int
		TotalGames   int
	}{
		Entries:      entries,
		Matches:      matches,
		TotalPlayers: len(entries),
		TotalGames:   calculateTotalGames(entries),
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
	}
}

func handleAPILeaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := getLeaderboard(50)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load leaderboard: %v", err), http.StatusInternalServerError)
		return
	}

	// Empty leaderboard is fine
	if entries == nil {
		entries = []LeaderboardEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func handleBracketData(w http.ResponseWriter, r *http.Request) {
	// Get latest tournament (active or completed)
	tournament, err := getLatestTournament()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load tournament: %v", err), http.StatusInternalServerError)
		return
	}
	
	if tournament == nil {
		// No tournament yet
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"stages":       []map[string]interface{}{},
			"matches":      []map[string]interface{}{},
			"participants": []map[string]interface{}{},
		})
		return
	}
	
	// Get all bracket matches
	matches, err := getAllBracketMatches(tournament.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load matches: %v", err), http.StatusInternalServerError)
		return
	}

	if matches == nil {
		matches = []BracketMatch{}
	}

	// Get unique participants (skip byes where ID = 0)
	participantMap := make(map[int]int) // submissionID -> participantID
	participants := []map[string]interface{}{}
	participantID := 1

	for _, match := range matches {
		if match.Player1ID > 0 && match.Player1Name != "" {
			if _, exists := participantMap[match.Player1ID]; !exists {
				participantMap[match.Player1ID] = participantID
				participants = append(participants, map[string]interface{}{
					"id":   participantID,
					"name": match.Player1Name,
				})
				participantID++
			}
		}
		if match.Player2ID > 0 && match.Player2Name != "" {
			if _, exists := participantMap[match.Player2ID]; !exists {
				participantMap[match.Player2ID] = participantID
				participants = append(participants, map[string]interface{}{
					"id":   participantID,
					"name": match.Player2Name,
				})
				participantID++
			}
		}
	}

	// Group matches by round for bracket format
	roundMatches := make(map[int][]BracketMatch)
	maxRound := 0
	for _, match := range matches {
		roundMatches[match.Round] = append(roundMatches[match.Round], match)
		if match.Round > maxRound {
			maxRound = match.Round
		}
	}

	// Create match data in brackets-viewer format (single elimination)
	bracketMatches := []map[string]interface{}{}
	matchNumber := 1
	
	for round := 1; round <= maxRound; round++ {
		for _, match := range roundMatches[round] {
			var opponent1, opponent2 map[string]interface{}
			
			// Player 1
			if match.Player1ID > 0 {
				result := "loss"
				if match.WinnerID == match.Player1ID {
					result = "win"
				}
				opponent1 = map[string]interface{}{
					"id":     participantMap[match.Player1ID],
					"result": result,
					"score":  match.Player1Wins,
				}
			} else {
				opponent1 = nil // Bye
			}
			
			// Player 2
			if match.Player2ID > 0 {
				result := "loss"
				if match.WinnerID == match.Player2ID {
					result = "win"
				}
				opponent2 = map[string]interface{}{
					"id":     participantMap[match.Player2ID],
					"result": result,
					"score":  match.Player2Wins,
				}
			} else {
				opponent2 = nil // Bye
			}

			status := "pending"
			if match.Status == "completed" {
				status = "completed"
			}

			bracketMatches = append(bracketMatches, map[string]interface{}{
				"id":         matchNumber,
				"stage_id":   1,
				"group_id":   1,
				"round_id":   round,
				"number":     match.Position + 1,
				"opponent1":  opponent1,
				"opponent2":  opponent2,
				"status":     status,
			})
			matchNumber++
		}
	}

	// Create stage data for single elimination
	// Calculate bracket size (next power of 2)
	bracketSize := 1
	for bracketSize < len(participants) {
		bracketSize *= 2
	}
	
	stages := []map[string]interface{}{
		{
			"id":     1,
			"name":   "Tournament",
			"type":   "single_elimination",
			"number": 1,
			"settings": map[string]interface{}{
				"size":           bracketSize,
				"seedOrdering":   []string{"natural"},
				"grandFinal":     "none",
				"skipFirstRound": false,
			},
		},
	}
	
	// Create groups array (required for brackets-viewer)
	groups := []map[string]interface{}{
		{
			"id":       1,
			"stage_id": 1,
			"number":   1,
		},
	}

	data := map[string]interface{}{
		"stages":       stages,
		"groups":       groups,
		"matches":      bracketMatches,
		"matchGames":   []map[string]interface{}{},
		"participants": participants,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func calculateTotalGames(entries []LeaderboardEntry) int {
	total := 0
	for _, e := range entries {
		total += e.Wins + e.Losses
	}
	return total / 2 // Each game counted twice (win+loss)
}

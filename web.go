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
    </style>
    <script>
        // Auto-refresh every 30 seconds
        setTimeout(() => location.reload(), 30000);
    </script>
</head>
<body>
    <div class="container">
        <h1>🚢 Battleship Arena</h1>
        <p class="subtitle">Smart AI Competition Leaderboard</p>
        
        <table>
            <thead>
                <tr>
                    <th>Rank</th>
                    <th>Player</th>
                    <th>Wins</th>
                    <th>Losses</th>
                    <th>Win Rate</th>
                    <th>Avg Moves</th>
                    <th>Last Played</th>
                </tr>
            </thead>
            <tbody>
                {{range $i, $e := .Entries}}
                <tr>
                    <td class="rank rank-{{add $i 1}}">{{if lt $i 3}}{{medal $i}}{{else}}#{{add $i 1}}{{end}}</td>
                    <td><strong>{{$e.Username}}</strong></td>
                    <td>{{$e.Wins}}</td>
                    <td>{{$e.Losses}}</td>
                    <td class="win-rate {{winRateClass $e}}">{{winRate $e}}%</td>
                    <td>{{printf "%.1f" $e.AvgMoves}}</td>
                    <td>{{$e.LastPlayed.Format "Jan 2, 3:04 PM"}}</td>
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
		http.Error(w, "Failed to load leaderboard", http.StatusInternalServerError)
		return
	}

	data := struct {
		Entries      []LeaderboardEntry
		TotalPlayers int
		TotalGames   int
	}{
		Entries:      entries,
		TotalPlayers: len(entries),
		TotalGames:   calculateTotalGames(entries),
	}

	tmpl.Execute(w, data)
}

func handleAPILeaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := getLeaderboard(50)
	if err != nil {
		http.Error(w, "Failed to load leaderboard", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func calculateTotalGames(entries []LeaderboardEntry) int {
	total := 0
	for _, e := range entries {
		total += e.Wins + e.Losses
	}
	return total / 2 // Each game counted twice (win+loss)
}

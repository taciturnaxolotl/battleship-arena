package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
)

const leaderboardHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Battleship Arena</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        
        body {
            font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
            background: #0f172a;
            color: #e2e8f0;
            min-height: 100vh;
            padding: 2rem 1rem;
        }
        
        .container {
            max-width: 1400px;
            margin: 0 auto;
        }
        
        header {
            text-align: center;
            margin-bottom: 3rem;
        }
        
        h1 {
            font-size: 3rem;
            font-weight: 800;
            background: linear-gradient(135deg, #3b82f6 0%, #8b5cf6 50%, #ec4899 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
            margin-bottom: 0.5rem;
        }
        
        .subtitle {
            font-size: 1.125rem;
            color: #94a3b8;
        }
        
        .status-bar {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 0.5rem;
            margin: 1.5rem 0;
            padding: 0.75rem;
            background: rgba(16, 185, 129, 0.1);
            border: 1px solid rgba(16, 185, 129, 0.2);
            border-radius: 0.5rem;
            font-size: 0.875rem;
            color: #10b981;
        }
        
        .live-dot {
            width: 8px;
            height: 8px;
            background: #10b981;
            border-radius: 50%;
            animation: pulse 2s ease-in-out infinite;
        }
        
        @keyframes pulse {
            0%, 100% { opacity: 1; transform: scale(1); }
            50% { opacity: 0.5; transform: scale(1.1); }
        }
        
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1.5rem;
            margin-bottom: 2rem;
        }
        
        .stat-card {
            background: #1e293b;
            border: 1px solid #334155;
            border-radius: 0.75rem;
            padding: 1.5rem;
            text-align: center;
        }
        
        .stat-value {
            font-size: 2.5rem;
            font-weight: 700;
            background: linear-gradient(135deg, #3b82f6, #8b5cf6);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            background-clip: text;
        }
        
        .stat-label {
            font-size: 0.875rem;
            color: #94a3b8;
            margin-top: 0.5rem;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }
        
        .leaderboard {
            background: #1e293b;
            border: 1px solid #334155;
            border-radius: 0.75rem;
            overflow: hidden;
            margin-bottom: 2rem;
        }
        
        .leaderboard-header {
            padding: 1.5rem;
            background: linear-gradient(135deg, #1e293b 0%, #334155 100%);
            border-bottom: 1px solid #334155;
        }
        
        .leaderboard-header h2 {
            font-size: 1.5rem;
            font-weight: 700;
        }
        
        table {
            width: 100%;
            border-collapse: collapse;
        }
        
        thead {
            background: #0f172a;
        }
        
        th {
            padding: 1rem 1.5rem;
            text-align: left;
            font-size: 0.75rem;
            font-weight: 600;
            color: #94a3b8;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }
        
        th:first-child { width: 80px; }
        th:nth-child(3) { width: 90px; }  /* ELO */
        th:nth-child(4), th:nth-child(5) { width: 100px; }  /* Wins, Losses */
        th:nth-child(6) { width: 120px; }  /* Win Rate */
        th:nth-child(7) { width: 120px; }  /* Avg Moves */
        th:last-child { width: 150px; }  /* Last Active */
        
        tbody tr {
            border-bottom: 1px solid #334155;
            transition: background 0.2s;
        }
        
        tbody tr:hover {
            background: rgba(59, 130, 246, 0.05);
        }
        
        tbody tr:last-child {
            border-bottom: none;
        }
        
        td {
            padding: 1.25rem 1.5rem;
            font-size: 0.9375rem;
        }
        
        .rank {
            font-size: 1.25rem;
            font-weight: 700;
        }
        
        .rank-1 { color: #fbbf24; }
        .rank-2 { color: #d1d5db; }
        .rank-3 { color: #f59e0b; }
        
        .player-name {
            font-weight: 600;
            color: #e2e8f0;
        }
        
        .win-rate {
            font-weight: 600;
            padding: 0.25rem 0.75rem;
            border-radius: 0.375rem;
            display: inline-block;
        }
        
        .win-rate-high { 
            background: rgba(16, 185, 129, 0.1);
            color: #10b981;
        }
        
        .win-rate-med { 
            background: rgba(245, 158, 11, 0.1);
            color: #f59e0b;
        }
        
        .win-rate-low { 
            background: rgba(239, 68, 68, 0.1);
            color: #ef4444;
        }
        
        .info-card {
            background: #1e293b;
            border: 1px solid #334155;
            border-radius: 0.75rem;
            padding: 2rem;
        }
        
        .info-card h3 {
            font-size: 1.25rem;
            margin-bottom: 1rem;
            color: #e2e8f0;
        }
        
        .info-card p {
            color: #94a3b8;
            line-height: 1.6;
            margin-bottom: 0.75rem;
        }
        
        code {
            background: #0f172a;
            padding: 0.375rem 0.75rem;
            border-radius: 0.375rem;
            font-family: 'Monaco', 'Courier New', monospace;
            font-size: 0.875rem;
            color: #3b82f6;
        }
        
        .empty-state {
            text-align: center;
            padding: 4rem 2rem;
            color: #64748b;
        }
        
        .empty-state-icon {
            font-size: 3rem;
            margin-bottom: 1rem;
        }
        
        @media (max-width: 768px) {
            h1 { font-size: 2rem; }
            .subtitle { font-size: 1rem; }
            th, td { padding: 0.75rem 1rem; font-size: 0.875rem; }
            .stat-value { font-size: 2rem; }
        }
    </style>
    <script>
        let eventSource;
        
        function connectSSE() {
            console.log('Connecting to SSE...');
            eventSource = new EventSource('http://localhost:8081');
            
            eventSource.onopen = () => {
                console.log('SSE connection established');
                document.querySelector('.status-bar').style.borderColor = 'rgba(16, 185, 129, 0.4)';
            };
            
            eventSource.onmessage = (event) => {
                try {
                    const entries = JSON.parse(event.data);
                    console.log('Updating leaderboard with', entries.length, 'entries');
                    updateLeaderboard(entries);
                } catch (error) {
                    console.error('Failed to parse SSE data:', error);
                }
            };
            
            eventSource.onerror = (error) => {
                console.error('SSE error, reconnecting...', error);
                document.querySelector('.status-bar').style.borderColor = 'rgba(239, 68, 68, 0.4)';
                eventSource.close();
                setTimeout(connectSSE, 5000);
            };
        }
        
        function updateLeaderboard(entries) {
            const tbody = document.querySelector('tbody');
            if (!tbody) return;
            
            if (entries.length === 0) {
                tbody.innerHTML = '<tr><td colspan="8"><div class="empty-state"><div class="empty-state-icon">🎯</div><div>No submissions yet. Be the first to compete!</div></div></td></tr>';
                return;
            }
            
            tbody.innerHTML = entries.map((e, i) => {
                const rank = i + 1;
                const winRate = e.WinPct.toFixed(1);
                const winRateClass = e.WinPct >= 60 ? 'win-rate-high' : e.WinPct >= 40 ? 'win-rate-med' : 'win-rate-low';
                const medals = ['🥇', '🥈', '🥉'];
                const medal = medals[i] || rank;
                const lastPlayed = new Date(e.LastPlayed).toLocaleString('en-US', { 
                    month: 'short', 
                    day: 'numeric',
                    hour: 'numeric',
                    minute: '2-digit'
                });
                
                return '<tr>' +
                    '<td class="rank rank-' + rank + '">' + medal + '</td>' +
                    '<td class="player-name">' + e.Username + '</td>' +
                    '<td><strong>' + e.Elo + '</strong></td>' +
                    '<td>' + e.Wins.toLocaleString() + '</td>' +
                    '<td>' + e.Losses.toLocaleString() + '</td>' +
                    '<td><span class="win-rate ' + winRateClass + '">' + winRate + '%</span></td>' +
                    '<td>' + e.AvgMoves.toFixed(1) + '</td>' +
                    '<td style="color: #64748b;">' + lastPlayed + '</td>' +
                    '</tr>';
            }).join('');
            
            // Update stats
            const statValues = document.querySelectorAll('.stat-value');
            statValues[0].textContent = entries.length;
            const totalGames = entries.reduce((sum, e) => sum + e.Wins + e.Losses, 0);
            statValues[1].textContent = totalGames.toLocaleString();
        }
        
        window.addEventListener('DOMContentLoaded', () => {
            connectSSE();
        });
    </script>
</head>
<body>
    <div class="container">
        <header>
            <h1>⚓ BATTLESHIP ARENA</h1>
            <p class="subtitle">AI Strategy Competition</p>
        </header>
        
        <div class="status-bar">
            <div class="live-dot"></div>
            <span>Live Updates</span>
        </div>
        
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-value">{{.TotalPlayers}}</div>
                <div class="stat-label">Active Players</div>
            </div>
            <div class="stat-card">
                <div class="stat-value">{{.TotalGames}}</div>
                <div class="stat-label">Games Played</div>
            </div>
        </div>
        
        <div class="leaderboard">
            <div class="leaderboard-header">
                <h2>🏆 Leaderboard</h2>
            </div>
            <table>
                <thead>
                    <tr>
                        <th>Rank</th>
                        <th>Player</th>
                        <th>ELO</th>
                        <th>Wins</th>
                        <th>Losses</th>
                        <th>Win Rate</th>
                        <th>Avg Moves</th>
                        <th>Last Active</th>
                    </tr>
                </thead>
                <tbody>
                    {{if .Entries}}
                    {{range $i, $e := .Entries}}
                    <tr>
                        <td class="rank rank-{{add $i 1}}">{{if lt $i 3}}{{medal $i}}{{else}}{{add $i 1}}{{end}}</td>
                        <td class="player-name">{{$e.Username}}</td>
                        <td><strong>{{$e.Elo}}</strong></td>
                        <td>{{$e.Wins}}</td>
                        <td>{{$e.Losses}}</td>
                        <td><span class="win-rate {{winRateClass $e}}">{{winRate $e}}%</span></td>
                        <td>{{printf "%.1f" $e.AvgMoves}}</td>
                        <td style="color: #64748b;">{{$e.LastPlayed.Format "Jan 2, 3:04 PM"}}</td>
                    </tr>
                    {{end}}
                    {{else}}
                    <tr>
                        <td colspan="8">
                            <div class="empty-state">
                                <div class="empty-state-icon">🎯</div>
                                <div>No submissions yet. Be the first to compete!</div>
                            </div>
                        </td>
                    </tr>
                    {{end}}
                </tbody>
            </table>
        </div>
        
        <div class="info-card">
            <h3>📤 How to Submit</h3>
            <p>Connect via SSH to submit your battleship AI:</p>
            <p><code>ssh -p 2222 username@localhost</code></p>
            <p style="margin-top: 1rem;">Upload your <code>memory_functions_*.cpp</code> file and compete in the arena!</p>
        </div>
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
		return formatFloat(e.WinPct, 1)
	},
	"winRateClass": func(e LeaderboardEntry) string {
		if e.WinPct >= 60 {
			return "win-rate-high"
		} else if e.WinPct >= 40 {
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



func calculateTotalGames(entries []LeaderboardEntry) int {
	total := 0
	for _, e := range entries {
		total += e.Wins + e.Losses
	}
	return total / 2 // Each game counted twice (win+loss)
}

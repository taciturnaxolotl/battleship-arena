package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"

	"github.com/go-chi/chi/v5"

	"battleship-arena/internal/storage"
)

const leaderboardHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Battleship Arena</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="icon" href="data:image/svg+xml,<svg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 100 100%22><text y=%22.9em%22 font-size=%2290%22>⚓</text></svg>">
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
        
        tbody tr.pending {
            opacity: 0.5;
            color: #64748b;
        }
        
        tbody tr.pending .player-name {
            color: #64748b;
        }
        
        tbody tr.pending .rank {
            color: #64748b !important;
        }
        
        tbody tr.broken {
            opacity: 0.6;
            color: #ef4444;
        }
        
        tbody tr.broken .player-name {
            color: #f87171;
        }
        
        tbody tr.broken .rank {
            color: #ef4444 !important;
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
        
        .player-name a:hover {
            color: #60a5fa !important;
            text-decoration: underline !important;
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
            padding: 0.5rem 0.875rem;
            border-radius: 0.5rem;
            font-family: 'Monaco', 'Courier New', monospace;
            font-size: 0.875rem;
            color: #60a5fa;
            border: 1px solid #1e3a8a;
            display: inline-block;
            line-height: 1.5;
        }
        
        .code-block {
            position: relative;
            background: #0f172a;
            border: 1px solid #1e3a8a;
            border-radius: 0.5rem;
            margin: 1rem 0;
            overflow: hidden;
        }
        
        .code-block-header {
            background: #1e3a8a;
            padding: 0.5rem 1rem;
            display: flex;
            justify-content: space-between;
            align-items: center;
            border-bottom: 1px solid #1e3a8a;
        }
        
        .code-block-lang {
            color: #94a3b8;
            font-size: 0.75rem;
            font-weight: 600;
            text-transform: uppercase;
            letter-spacing: 0.05em;
        }
        
        .code-block-copy {
            background: #3b82f6;
            color: white;
            border: none;
            padding: 0.25rem 0.75rem;
            border-radius: 0.25rem;
            font-size: 0.75rem;
            cursor: pointer;
            transition: background 0.2s;
        }
        
        .code-block-copy:hover {
            background: #2563eb;
        }
        
        .code-block-copy.copied {
            background: #10b981;
        }
        
        .code-block pre {
            margin: 0;
            padding: 1rem;
            overflow-x: auto;
        }
        
        .code-block code {
            background: transparent;
            border: none;
            padding: 0;
            display: block;
            color: #e2e8f0;
        }
        
        .code-block .token-command {
            color: #60a5fa;
        }
        
        .code-block .token-flag {
            color: #a78bfa;
        }
        
        .code-block .token-string {
            color: #34d399;
        }
        
        .code-block .token-comment {
            color: #64748b;
            font-style: italic;
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
        
        .progress-indicator {
            position: fixed;
            bottom: 2rem;
            right: 2rem;
            background: #1e293b;
            border: 2px solid #3b82f6;
            border-radius: 12px;
            padding: 1.5rem;
            box-shadow: 0 10px 25px rgba(0, 0, 0, 0.5);
            min-width: 300px;
            z-index: 1000;
            animation: slideIn 0.3s ease-out;
        }
        
        @keyframes slideIn {
            from {
                transform: translateX(400px);
                opacity: 0;
            }
            to {
                transform: translateX(0);
                opacity: 1;
            }
        }
        
        .progress-indicator.hidden {
            display: none;
        }
        
        .progress-header {
            display: flex;
            align-items: center;
            margin-bottom: 1rem;
        }
        
        .progress-spinner {
            width: 20px;
            height: 20px;
            border: 3px solid #334155;
            border-top-color: #3b82f6;
            border-radius: 50%;
            animation: spin 0.8s linear infinite;
            margin-right: 0.75rem;
        }
        
        @keyframes spin {
            to { transform: rotate(360deg); }
        }
        
        .progress-title {
            font-weight: 600;
            color: #e2e8f0;
            font-size: 1rem;
        }
        
        .progress-player {
            color: #3b82f6;
            font-weight: 700;
            margin-bottom: 0.5rem;
        }
        
        .progress-stats {
            font-size: 0.875rem;
            color: #94a3b8;
            margin-bottom: 0.75rem;
        }
        
        .progress-bar-container {
            background: #0f172a;
            border-radius: 4px;
            height: 8px;
            overflow: hidden;
            margin-bottom: 0.5rem;
        }
        
        .progress-bar {
            background: linear-gradient(90deg, #3b82f6, #8b5cf6);
            height: 100%;
            transition: width 0.5s ease;
            border-radius: 4px;
        }
        
        .progress-time {
            font-size: 0.75rem;
            color: #64748b;
            text-align: right;
        }
        
        .progress-queue {
            margin-top: 1rem;
            padding-top: 1rem;
            border-top: 1px solid #334155;
        }
        
        .progress-queue-title {
            font-size: 0.75rem;
            color: #64748b;
            margin-bottom: 0.5rem;
        }
        
        .progress-queue-list {
            font-size: 0.875rem;
            color: #94a3b8;
            max-height: 100px;
            overflow-y: auto;
        }
        
        .progress-queue-item {
            padding: 0.25rem 0;
        }
        
        .hidden {
            display: none;
        }
        
        .tooltip {
            position: relative;
            display: inline-block;
            cursor: help;
        }
        
        .tooltip:hover::after {
            content: attr(data-tooltip);
            position: absolute;
            bottom: 100%;
            left: 50%;
            transform: translateX(-50%);
            background: #1e293b;
            border: 1px solid #3b82f6;
            color: #e2e8f0;
            padding: 0.5rem 0.75rem;
            border-radius: 0.375rem;
            font-size: 0.75rem;
            white-space: nowrap;
            z-index: 1000;
            margin-bottom: 0.5rem;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.3);
        }
        
        .info-card ul {
            list-style: none;
            margin: 1rem 0;
        }
        
        .info-card li {
            padding: 0.5rem 0;
            color: #94a3b8;
            display: flex;
            align-items: start;
            gap: 0.5rem;
        }
        
        .info-card li::before {
            content: "→";
            color: #3b82f6;
            font-weight: bold;
            flex-shrink: 0;
        }
        
        @media (max-width: 768px) {
            h1 { font-size: 2rem; }
            .subtitle { font-size: 1rem; }
            th, td { padding: 0.75rem 1rem; font-size: 0.875rem; }
            .stat-value { font-size: 2rem; }
            .progress-indicator {
                bottom: 1rem;
                right: 1rem;
                left: 1rem;
                min-width: unset;
            }
        }
    </style>
    <script>
        let eventSource;
        
        function connectSSE() {
            console.log('Connecting to SSE...');
            eventSource = new EventSource('/events/updates');
            
            eventSource.onopen = () => {
                console.log('SSE connection established');
                document.querySelector('.status-bar').style.borderColor = 'rgba(16, 185, 129, 0.4)';
            };
            
            eventSource.onmessage = (event) => {
                console.log('SSE raw event:', event);
                console.log('SSE event.data:', event.data);
                try {
                    const data = JSON.parse(event.data);
                    console.log('SSE message received:', data);
                    
                    // Check if it's a progress update or leaderboard update
                    if (data.type === 'progress') {
                        console.log('Progress update:', data);
                        updateProgress(data);
                    } else if (data.type === 'complete') {
                        console.log('Progress complete');
                        hideProgress();
                    } else if (Array.isArray(data)) {
                        // Leaderboard update
                        console.log('Updating leaderboard with', data.length, 'entries');
                        updateLeaderboard(data);
                    } else {
                        console.log('Unknown message type:', data);
                    }
                } catch (error) {
                    console.error('Failed to parse SSE data:', error, 'Raw data:', event.data);
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
                const isPending = e.IsPending || false;
                const rowClass = isPending ? ' class="pending"' : '';
                
                let rankDisplay;
                if (isPending) {
                    rankDisplay = '⏳';
                } else {
                    const medals = ['🥇', '🥈', '🥉'];
                    rankDisplay = medals[i] || rank;
                }
                
                const winRate = e.WinPct.toFixed(1);
                const winRateClass = e.WinPct >= 60 ? 'win-rate-high' : e.WinPct >= 40 ? 'win-rate-med' : 'win-rate-low';
                const lastPlayed = isPending ? 'Waiting...' : new Date(e.LastPlayed).toLocaleString('en-US', { 
                    month: 'short', 
                    day: 'numeric',
                    hour: 'numeric',
                    minute: '2-digit'
                });
                
                const nameDisplay = e.Username + (isPending ? ' <span style="font-size: 0.8em;">(pending)</span>' : '');
                const ratingDisplay = isPending ? '-' : '<strong>' + e.Rating + '</strong> <span style="color: #94a3b8; font-size: 0.85em;">±' + e.RD + '</span>';
                const winsDisplay = isPending ? '-' : e.Wins.toLocaleString();
                const lossesDisplay = isPending ? '-' : e.Losses.toLocaleString();
                const winRateDisplay = isPending ? '-' : '<span class="win-rate ' + winRateClass + '">' + winRate + '%</span>';
                const avgMovesDisplay = isPending ? '-' : e.AvgMoves.toFixed(1);
                
                return '<tr' + rowClass + '>' +
                    '<td class="rank rank-' + rank + '">' + rankDisplay + '</td>' +
                    '<td class="player-name"><a href="/user/' + e.Username + '" style="color: inherit; text-decoration: none;">' + nameDisplay + '</a></td>' +
                    '<td>' + ratingDisplay + '</td>' +
                    '<td>' + winsDisplay + '</td>' +
                    '<td>' + lossesDisplay + '</td>' +
                    '<td>' + winRateDisplay + '</td>' +
                    '<td>' + avgMovesDisplay + '</td>' +
                    '<td style="color: #64748b;">' + lastPlayed + '</td>' +
                    '</tr>';
            }).join('');
            
            // Update stats
            const statValues = document.querySelectorAll('.stat-value');
            statValues[0].textContent = entries.length;
            const totalGames = entries.reduce((sum, e) => sum + e.Wins + e.Losses, 0);
            statValues[1].textContent = totalGames.toLocaleString();
        }
        
        function updateProgress(data) {
            const indicator = document.getElementById('progress-indicator');
            
            if (!indicator) {
                console.error('Progress indicator element not found!');
                return;
            }
            
            console.log('Updating progress indicator:', data);
            
            // Show indicator
            indicator.classList.remove('hidden');
            
            // Update content
            document.getElementById('progress-player').textContent = data.player;
            document.getElementById('progress-current').textContent = data.current_match;
            document.getElementById('progress-total').textContent = data.total_matches;
            document.getElementById('progress-time').textContent = data.estimated_time_left;
            document.getElementById('progress-bar').style.width = data.percent_complete + '%';
            
            // Update queue
            const queueContainer = document.getElementById('progress-queue-container');
            if (data.queued_players && data.queued_players.length > 0) {
                queueContainer.style.display = 'block';
                const queueList = document.getElementById('progress-queue-list');
                queueList.innerHTML = data.queued_players.map(p => 
                    '<div class="progress-queue-item">⏳ ' + p + '</div>'
                ).join('');
            } else {
                queueContainer.style.display = 'none';
            }
        }
        
        function hideProgress() {
            const indicator = document.getElementById('progress-indicator');
            if (indicator) {
                indicator.classList.add('hidden');
            }
        }
        
        window.addEventListener('DOMContentLoaded', () => {
            connectSSE();
        });
        
        function copyCode(button, text) {
            // Decode HTML entities in template variables
            const tempDiv = document.createElement('div');
            tempDiv.innerHTML = text;
            const decodedText = tempDiv.textContent || tempDiv.innerText;
            
            navigator.clipboard.writeText(decodedText).then(() => {
                const originalText = button.textContent;
                button.textContent = 'Copied!';
                button.classList.add('copied');
                setTimeout(() => {
                    button.textContent = originalText;
                    button.classList.remove('copied');
                }, 2000);
            }).catch(err => {
                console.error('Failed to copy:', err);
            });
        }
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
                        <th><span class="tooltip" data-tooltip="Glicko-2 rating: higher is better">Rating</span></th>
                        <th>Wins</th>
                        <th>Losses</th>
                        <th>Win Rate</th>
                        <th><span class="tooltip" data-tooltip="Average moves to win (lower is better)">Avg Moves</span></th>
                        <th>Last Active</th>
                    </tr>
                </thead>
                <tbody>
                    {{if .Entries}}
                    {{range $i, $e := .Entries}}
                    <tr{{if $e.IsPending}} class="pending"{{else if $e.IsBroken}} class="broken"{{end}}>
                        <td class="rank rank-{{add $i 1}}">{{if $e.IsBroken}}💥{{else if $e.IsPending}}⏳{{else if lt $i 3}}{{medal $i}}{{else}}{{add $i 1}}{{end}}</td>
                        <td class="player-name"><a href="/user/{{$e.Username}}" style="color: inherit; text-decoration: none;">{{$e.Username}}{{if $e.IsPending}} <span style="font-size: 0.8em;">(pending)</span>{{else if $e.IsBroken}} <span style="font-size: 0.8em; color: #ef4444;">(compilation failed)</span>{{end}}</a></td>
                        <td>{{if or $e.IsPending $e.IsBroken}}-{{else}}<strong>{{$e.Rating}}</strong> <span style="color: #94a3b8; font-size: 0.85em;">±{{$e.RD}}</span>{{end}}</td>
                        <td>{{if or $e.IsPending $e.IsBroken}}-{{else}}{{$e.Wins}}{{end}}</td>
                        <td>{{if or $e.IsPending $e.IsBroken}}-{{else}}{{$e.Losses}}{{end}}</td>
                        <td>{{if or $e.IsPending $e.IsBroken}}-{{else}}<span class="win-rate {{winRateClass $e}}">{{winRate $e}}%</span>{{end}}</td>
                        <td>{{if or $e.IsPending $e.IsBroken}}-{{else}}{{printf "%.1f" $e.AvgMoves}}{{end}}</td>
                        <td style="color: #64748b;">{{if $e.IsPending}}Waiting...{{else if $e.IsBroken}}Failed{{else}}{{$e.LastPlayed.Format "Jan 2, 3:04 PM"}}{{end}}</td>
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
            <p><strong>First time?</strong> Connect via SSH to create your account:</p>
            
            <div class="code-block">
                <div class="code-block-header">
                    <span class="code-block-lang">bash</span>
                    <button class="code-block-copy" onclick="copyCode(this, 'ssh -p 2222 {{.ServerURL}}')">Copy</button>
                </div>
                <pre><code><span class="token-command">ssh</span> <span class="token-flag">-p</span> <span class="token-string">2222</span> <span class="token-string">{{.ServerURL}}</span></code></pre>
            </div>
            
            <p style="margin-top: 0.5rem; color: #94a3b8;">You'll be prompted for your name, bio, and link. Your SSH key will be registered.</p>
            
            <p style="margin-top: 1rem;"><strong>Upload your AI:</strong></p>
            
            <div class="code-block">
                <div class="code-block-header">
                    <span class="code-block-lang">bash</span>
                    <button class="code-block-copy" onclick="copyCode(this, 'scp -P 2222 memory_functions_yourname.cpp {{.ServerURL}}:~/')">Copy</button>
                </div>
                <pre><code><span class="token-command">scp</span> <span class="token-flag">-P</span> <span class="token-string">2222</span> <span class="token-string">memory_functions_yourname.cpp</span> <span class="token-string">{{.ServerURL}}:~/</span></code></pre>
            </div>
            
            <p style="margin-top: 1.5rem;"><strong>How it works:</strong></p>
            <ul>
                <li>Your AI plays 1000 games against each opponent</li>
                <li>Rankings use Glicko-2 rating system (like chess)</li>
                <li>Lower average moves = more efficient strategy</li>
                <li>Live updates as matches complete</li>
            </ul>
            
            <p style="margin-top: 1rem; color: #94a3b8;">
                <a href="/users" style="color: #60a5fa; text-decoration: none;">View all players →</a>
            </p>
        </div>
    </div>
    
    <!-- Progress Indicator -->
    <div id="progress-indicator" class="progress-indicator hidden">
        <div class="progress-header">
            <div class="progress-spinner"></div>
            <div class="progress-title">Computing Ratings</div>
        </div>
        <div class="progress-player" id="progress-player">-</div>
        <div class="progress-stats">
            Match <span id="progress-current">0</span> of <span id="progress-total">0</span>
        </div>
        <div class="progress-bar-container">
            <div class="progress-bar" id="progress-bar" style="width: 0%"></div>
        </div>
        <div class="progress-time">
            Est. <span id="progress-time">-</span> remaining
        </div>
        <div id="progress-queue-container" class="progress-queue" style="display: none;">
            <div class="progress-queue-title">Queued Players:</div>
            <div id="progress-queue-list" class="progress-queue-list"></div>
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
	"winRate": func(e storage.LeaderboardEntry) string {
		return formatFloat(e.WinPct, 1)
	},
	"winRateClass": func(e storage.LeaderboardEntry) string {
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

func HandleLeaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := storage.GetLeaderboard(50)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load leaderboard: %v", err), http.StatusInternalServerError)
		return
	}

	// Empty leaderboard is fine
	if entries == nil {
		entries = []storage.LeaderboardEntry{}
	}

	// Get matches for bracket
	matches, err := storage.GetAllMatches()
	if err != nil {
		matches = []storage.MatchResult{}
	}

	data := struct {
		Entries      []storage.LeaderboardEntry
		Matches      []storage.MatchResult
		TotalPlayers int
		TotalGames   int
		ServerURL    string
	}{
		Entries:      entries,
		Matches:      matches,
		TotalPlayers: len(entries),
		TotalGames:   calculateTotalGames(entries),
		ServerURL:    GetServerURL(),
	}

	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, fmt.Sprintf("Template error: %v", err), http.StatusInternalServerError)
	}
}

func HandleAPILeaderboard(w http.ResponseWriter, r *http.Request) {
	entries, err := storage.GetLeaderboard(50)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to load leaderboard: %v", err), http.StatusInternalServerError)
		return
	}

	// Empty leaderboard is fine
	if entries == nil {
		entries = []storage.LeaderboardEntry{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(entries)
}

func calculateTotalGames(entries []storage.LeaderboardEntry) int {
	total := 0
	for _, e := range entries {
		total += e.Wins + e.Losses
	}
	return total / 2 // Each game counted twice (win+loss)
}

func HandleRatingHistory(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "player")
	if username == "" {
		http.Error(w, "Username required", http.StatusBadRequest)
		return
	}

	// Get submission ID for this username
	var submissionID int
	err := storage.DB.QueryRow(
		"SELECT id FROM submissions WHERE username = ? AND is_active = 1",
		username,
	).Scan(&submissionID)

	if err != nil {
		http.Error(w, "Player not found", http.StatusNotFound)
		return
	}

	// Get rating history
	history, err := storage.GetRatingHistory(submissionID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get rating history: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func HandlePlayerPage(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "player")
	if username == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	tmpl := template.Must(template.New("player").Parse(playerPageHTML))
	tmpl.Execute(w, map[string]string{"Username": username})
}

const playerPageHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>{{.Username}} - Battleship Arena</title>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="icon" href="data:image/svg+xml,<svg xmlns=%22http://www.w3.org/2000/svg%22 viewBox=%220 0 100 100%22><text y=%22.9em%22 font-size=%2290%22>⚓</text></svg>">
    <script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.0/dist/chart.umd.min.js"></script>
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
            max-width: 1200px;
            margin: 0 auto;
        }
        
        h1 {
            font-size: 2.5rem;
            font-weight: 700;
            margin-bottom: 0.5rem;
            background: linear-gradient(135deg, #60a5fa 0%, #a78bfa 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        
        .back-link {
            display: inline-block;
            margin-bottom: 2rem;
            color: #60a5fa;
            text-decoration: none;
            font-size: 0.9rem;
        }
        
        .back-link:hover {
            text-decoration: underline;
        }
        
        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-bottom: 2rem;
        }
        
        .stat-card {
            background: #1e293b;
            border: 1px solid #334155;
            border-radius: 12px;
            padding: 1.5rem;
        }
        
        .stat-label {
            font-size: 0.875rem;
            color: #94a3b8;
            margin-bottom: 0.5rem;
        }
        
        .stat-value {
            font-size: 2rem;
            font-weight: 700;
            color: #60a5fa;
        }
        
        .chart-container {
            background: #1e293b;
            border: 1px solid #334155;
            border-radius: 12px;
            padding: 2rem;
            margin-bottom: 2rem;
        }
        
        .chart-title {
            font-size: 1.25rem;
            font-weight: 600;
            margin-bottom: 1.5rem;
            color: #e2e8f0;
        }
        
        canvas {
            max-height: 400px;
        }
    </style>
</head>
<body>
    <div class="container">
        <a href="/" class="back-link">← Back to Leaderboard</a>
        <h1>{{.Username}}</h1>
        <p style="color: #94a3b8; margin-bottom: 2rem;">Player Statistics</p>
        
        <div class="stats-grid" id="stats-grid">
            <div class="stat-card">
                <div class="stat-label">Current Rating</div>
                <div class="stat-value" id="current-rating">-</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Rating Deviation</div>
                <div class="stat-value" id="current-rd">-</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Win Rate</div>
                <div class="stat-value" id="win-rate">-</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Total Matches</div>
                <div class="stat-value" id="total-matches">-</div>
            </div>
        </div>
        
        <div class="chart-container">
            <h2 class="chart-title">Rating History</h2>
            <canvas id="rating-chart"></canvas>
        </div>
        
        <div class="chart-container">
            <h2 class="chart-title">Rating Deviation Over Time</h2>
            <canvas id="rd-chart"></canvas>
        </div>
    </div>
    
    <script>
        const username = "{{.Username}}";
        
        async function loadData() {
            try {
                // Load rating history
                const historyRes = await fetch('/api/rating-history/' + username);
                const history = await historyRes.json();
                
                // Load current stats from leaderboard
                const leaderboardRes = await fetch('/api/leaderboard');
                const leaderboard = await leaderboardRes.json();
                const player = leaderboard.find(p => p.Username === username);
                
                if (player) {
                    document.getElementById('current-rating').textContent = player.Rating + ' ±' + player.RD;
                    document.getElementById('current-rd').textContent = player.RD;
                    document.getElementById('win-rate').textContent = player.WinPct.toFixed(1) + '%';
                    const total = player.Wins + player.Losses;
                    document.getElementById('total-matches').textContent = Math.floor(total / 1000);
                }
                
                // Create rating chart
                const ratingCtx = document.getElementById('rating-chart').getContext('2d');
                new Chart(ratingCtx, {
                    type: 'line',
                    data: {
                        labels: history.map((h, i) => 'Match ' + (i + 1)),
                        datasets: [{
                            label: 'Rating',
                            data: history.map(h => h.Rating),
                            borderColor: '#60a5fa',
                            backgroundColor: 'rgba(96, 165, 250, 0.1)',
                            tension: 0.1,
                            fill: true
                        }]
                    },
                    options: {
                        responsive: true,
                        maintainAspectRatio: true,
                        plugins: {
                            legend: {
                                display: false
                            }
                        },
                        scales: {
                            y: {
                                beginAtZero: false,
                                grid: {
                                    color: '#334155'
                                },
                                ticks: {
                                    color: '#94a3b8'
                                }
                            },
                            x: {
                                grid: {
                                    color: '#334155'
                                },
                                ticks: {
                                    color: '#94a3b8',
                                    maxTicksLimit: 10
                                }
                            }
                        }
                    }
                });
                
                // Create RD chart
                const rdCtx = document.getElementById('rd-chart').getContext('2d');
                new Chart(rdCtx, {
                    type: 'line',
                    data: {
                        labels: history.map((h, i) => 'Match ' + (i + 1)),
                        datasets: [{
                            label: 'Rating Deviation',
                            data: history.map(h => h.RD),
                            borderColor: '#a78bfa',
                            backgroundColor: 'rgba(167, 139, 250, 0.1)',
                            tension: 0.1,
                            fill: true
                        }]
                    },
                    options: {
                        responsive: true,
                        maintainAspectRatio: true,
                        plugins: {
                            legend: {
                                display: false
                            }
                        },
                        scales: {
                            y: {
                                beginAtZero: false,
                                grid: {
                                    color: '#334155'
                                },
                                ticks: {
                                    color: '#94a3b8'
                                }
                            },
                            x: {
                                grid: {
                                    color: '#334155'
                                },
                                ticks: {
                                    color: '#94a3b8',
                                    maxTicksLimit: 10
                                }
                            }
                        }
                    }
                });
                
            } catch (err) {
                console.error('Failed to load data:', err);
            }
        }
        
        loadData();
    </script>
</body>
</html>
`

package server

import (
	"encoding/json"
	"html/template"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"

	"battleship-arena/internal/storage"
)

func HandlePlayPage(w http.ResponseWriter, r *http.Request) {
	aiName := chi.URLParam(r, "aiName")
	
	tmpl := template.Must(template.New("play").Parse(playPageHTML))
	tmpl.Execute(w, map[string]string{
		"AIName":    aiName,
		"ServerURL": GetServerURL(),
	})
}

func HandleAvailableAIs(w http.ResponseWriter, r *http.Request) {
	entries, err := storage.GetLeaderboard(50)
	if err != nil {
		http.Error(w, "Failed to load AIs", http.StatusInternalServerError)
		return
	}

	type AIInfo struct {
		Name   string `json:"name"`
		Rating int    `json:"rating"`
	}

	var ais []AIInfo
	re := regexp.MustCompile(`memory_functions_(\w+)\.cpp`)
	
	for _, e := range entries {
		if e.IsBroken {
			continue
		}
		// Get submission to extract AI name from filename
		subs, err := storage.GetUserSubmissions(e.Username)
		if err != nil || len(subs) == 0 {
			continue
		}
		
		// Find active submission
		for _, sub := range subs {
			if sub.Status == "completed" {
				matches := re.FindStringSubmatch(sub.Filename)
				if len(matches) >= 2 {
					ais = append(ais, AIInfo{
						Name:   matches[1],
						Rating: e.Rating,
					})
					break
				}
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ais)
}

const playPageHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Play Battleship - Battleship Arena</title>
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
        
        .back-link {
            display: inline-block;
            margin-bottom: 1.5rem;
            color: #60a5fa;
            text-decoration: none;
            font-size: 0.875rem;
            transition: color 0.2s;
        }
        
        .back-link:hover {
            color: #93c5fd;
        }
        
        .game-container {
            display: flex;
            gap: 2rem;
            justify-content: center;
            flex-wrap: wrap;
        }
        
        .board-section {
            background: #1e293b;
            border: 1px solid #334155;
            border-radius: 0.75rem;
            padding: 1.5rem;
        }
        
        .board-title {
            font-size: 1.125rem;
            font-weight: 700;
            margin-bottom: 1.25rem;
            color: #e2e8f0;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            font-size: 0.875rem;
        }
        
        .board {
            display: grid;
            grid-template-columns: 30px repeat(10, 40px);
            grid-template-rows: 30px repeat(10, 40px);
            gap: 1px;
            background: #0f172a;
            padding: 0.5rem;
            border-radius: 0.5rem;
            margin: 0 auto;
            width: fit-content;
        }
        
        .board.enemy {
            cursor: crosshair;
        }
        
        .board.enemy.disabled {
            cursor: not-allowed;
            opacity: 0.6;
        }
        
        .header-cell {
            display: flex;
            align-items: center;
            justify-content: center;
            font-weight: 600;
            color: #64748b;
            font-size: 0.75rem;
            text-transform: uppercase;
        }
        
        .cell {
            width: 40px;
            height: 40px;
            background: #1e293b;
            border: 1px solid #334155;
            display: flex;
            align-items: center;
            justify-content: center;
            font-size: 1.25rem;
            transition: all 0.15s;
            border-radius: 2px;
        }
        
        .board.enemy .cell:not(.hit):not(.miss):not(.sunk):hover {
            background: rgba(59, 130, 246, 0.2);
            border-color: #3b82f6;
            transform: scale(1.05);
            cursor: pointer;
        }
        
        .cell.ship {
            background: #334155;
        }
        
        .cell.hit {
            background: rgba(220, 38, 38, 0.2);
            border-color: #dc2626;
            color: #fca5a5;
        }
        
        .cell.miss {
            background: rgba(30, 64, 175, 0.15);
            border-color: #1e40af;
            color: #60a5fa;
        }
        
        .cell.sunk {
            background: rgba(127, 29, 29, 0.3);
            border-color: #7f1d1d;
            color: #fca5a5;
        }
        
        .ships-panel {
            background: #0f172a;
            border: 1px solid #334155;
            border-radius: 0.5rem;
            padding: 1.25rem;
            margin-top: 1.25rem;
        }
        
        .ships-title {
            font-size: 0.75rem;
            color: #94a3b8;
            margin-bottom: 0.75rem;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            font-weight: 600;
        }
        
        .ship-row {
            display: flex;
            align-items: center;
            gap: 0.5rem;
            padding: 0.5rem;
            font-size: 0.875rem;
            border-radius: 0.25rem;
            transition: background 0.15s;
        }
        
        .ship-row:hover {
            background: rgba(59, 130, 246, 0.05);
        }
        
        .ship-row.sunk {
            text-decoration: line-through;
            opacity: 0.4;
        }
        
        .ship-icon {
            font-size: 1rem;
        }
        
        .ship-name {
            font-size: 0.875rem;
            color: #e2e8f0;
        }
        
        .status-bar {
            text-align: center;
            padding: 1.25rem;
            margin: 2rem auto;
            max-width: 600px;
            border-radius: 0.75rem;
            font-size: 1.125rem;
            font-weight: 600;
            border: 1px solid;
        }
        
        .status-bar.your-turn {
            background: rgba(16, 185, 129, 0.1);
            border-color: rgba(16, 185, 129, 0.3);
            color: #10b981;
        }
        
        .status-bar.ai-turn {
            background: rgba(245, 158, 11, 0.1);
            border-color: rgba(245, 158, 11, 0.3);
            color: #f59e0b;
        }
        
        .status-bar.game-over {
            background: rgba(139, 92, 246, 0.1);
            border-color: rgba(139, 92, 246, 0.3);
            color: #8b5cf6;
        }
        
        .ai-selector {
            background: #1e293b;
            border: 1px solid #334155;
            border-radius: 0.75rem;
            padding: 2.5rem;
            text-align: center;
            max-width: 800px;
            margin: 0 auto;
        }
        
        .ai-selector h2 {
            font-size: 1.5rem;
            margin-bottom: 0.5rem;
        }
        
        .ai-selector p {
            color: #94a3b8;
            margin-bottom: 2rem;
        }
        
        .ai-list {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
            gap: 1rem;
        }
        
        .ai-card {
            background: #0f172a;
            border: 1px solid #334155;
            border-radius: 0.5rem;
            padding: 1.25rem;
            cursor: pointer;
            transition: all 0.15s;
        }
        
        .ai-card:hover {
            border-color: #3b82f6;
            background: rgba(59, 130, 246, 0.05);
            transform: translateY(-2px);
        }
        
        .ai-name {
            font-weight: 600;
            margin-bottom: 0.5rem;
            color: #e2e8f0;
        }
        
        .ai-rating {
            font-size: 0.875rem;
            color: #94a3b8;
        }
        
        .btn {
            background: linear-gradient(135deg, #3b82f6, #8b5cf6);
            color: white;
            border: none;
            padding: 0.875rem 2rem;
            border-radius: 0.5rem;
            font-size: 0.9375rem;
            font-weight: 600;
            cursor: pointer;
            transition: all 0.15s;
        }
        
        .btn:hover {
            transform: translateY(-1px);
            box-shadow: 0 4px 12px rgba(59, 130, 246, 0.4);
        }
        
        .btn:disabled {
            opacity: 0.5;
            cursor: not-allowed;
            transform: none;
        }
        
        .move-log {
            background: #0f172a;
            border: 1px solid #334155;
            border-radius: 0.5rem;
            padding: 1.25rem;
            margin-top: 2rem;
            max-width: 800px;
            margin-left: auto;
            margin-right: auto;
        }
        
        .move-log-title {
            font-size: 0.75rem;
            color: #94a3b8;
            margin-bottom: 0.75rem;
            text-transform: uppercase;
            letter-spacing: 0.05em;
            font-weight: 600;
        }
        
        .move-log-content {
            max-height: 200px;
            overflow-y: auto;
        }
        
        .move-entry {
            font-size: 0.875rem;
            padding: 0.5rem;
            margin-bottom: 0.25rem;
            border-radius: 0.25rem;
            background: #1e293b;
        }
        
        .move-entry:last-child {
            margin-bottom: 0;
        }
        
        .move-entry.hit {
            color: #fca5a5;
            border-left: 3px solid #dc2626;
        }
        
        .move-entry.miss {
            color: #93c5fd;
            border-left: 3px solid #3b82f6;
        }
        
        .move-entry.sunk {
            color: #fbbf24;
            border-left: 3px solid #f59e0b;
            font-weight: 600;
        }

        #loading {
            text-align: center;
            padding: 2rem;
            color: #64748b;
        }
        
        .error-toast {
            position: fixed;
            bottom: 2rem;
            left: 50%;
            transform: translateX(-50%);
            background: #7f1d1d;
            border: 1px solid #dc2626;
            color: #fca5a5;
            padding: 1rem 2rem;
            border-radius: 8px;
            font-size: 0.9rem;
            z-index: 1000;
            animation: slideUp 0.3s ease-out;
        }
        
        .error-toast.fade-out {
            animation: fadeOut 0.3s ease-out forwards;
        }
        
        @keyframes slideUp {
            from { transform: translateX(-50%) translateY(100%); opacity: 0; }
            to { transform: translateX(-50%) translateY(0); opacity: 1; }
        }
        
        @keyframes fadeOut {
            to { opacity: 0; transform: translateX(-50%) translateY(20px); }
        }
    </style>
</head>
<body>
    <div class="container">
        <a href="/" class="back-link">← Back to Leaderboard</a>
        
        <header>
            <h1>⚓ Battleship Arena</h1>
            <p style="color: #94a3b8; font-size: 1.125rem;">Challenge an AI opponent</p>
        </header>
        
        <div id="game-area">
            <div id="loading">Loading available AIs...</div>
        </div>
    </div>
    
    <script>
        let ws = null;
        let gameState = null;
        let selectedAI = "{{.AIName}}";
        
        const CELL_EMPTY = 32;  // ' '
        const CELL_SHIP = 83;   // 'S'
        const CELL_HIT = 88;    // 'X'
        const CELL_MISS = 79;   // 'O'
        const CELL_SUNK = 35;   // '#'
        
        async function init() {
            if (selectedAI) {
                startGame(selectedAI);
            } else {
                await showAISelector();
            }
        }
        
        async function showAISelector() {
            const res = await fetch('/api/available-ais');
            const ais = await res.json();
            
            const area = document.getElementById('game-area');
            
            if (!ais || ais.length === 0) {
                area.innerHTML = '<div class="ai-selector"><h2>No AIs Available</h2><p>No AI submissions are ready to play against yet.</p></div>';
                return;
            }
            
            let html = '<div class="ai-selector"><h2>Choose an Opponent</h2><div class="ai-list">';
            
            for (const ai of ais) {
                html += '<div class="ai-card" onclick="startGame(\'' + ai.name + '\')">' +
                        '<div class="ai-name">' + ai.name + '</div>' +
                        '<div class="ai-rating">Rating: ' + ai.rating + '</div></div>';
            }
            
            html += '</div></div>';
            area.innerHTML = html;
        }
        
        function startGame(aiName) {
            selectedAI = aiName;
            
            // Connect WebSocket
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            ws = new WebSocket(protocol + '//' + window.location.host + '/ws/game');
            
            ws.onopen = () => {
                ws.send(JSON.stringify({
                    type: 'start_game',
                    payload: { aiName: aiName }
                }));
            };
            
            ws.onmessage = (event) => {
                const msg = JSON.parse(event.data);
                handleMessage(msg);
            };
            
            ws.onclose = () => {
                console.log('WebSocket closed');
            };
            
            ws.onerror = (err) => {
                console.error('WebSocket error:', err);
                document.getElementById('game-area').innerHTML = 
                    '<div class="ai-selector"><h2>Connection Error</h2><p>Failed to connect to game server.</p>' +
                    '<button class="btn" onclick="location.reload()">Retry</button></div>';
            };
            
            document.getElementById('game-area').innerHTML = '<div id="loading">Starting game against ' + aiName + '...</div>';
        }
        
        function handleMessage(msg) {
            switch (msg.type) {
                case 'game_state':
                    gameState = msg.payload;
                    renderGame();
                    break;
                    
                case 'human_move_result':
                case 'ai_move_result':
                    addMoveLog(msg.type === 'human_move_result' ? 'You' : 'AI', msg.payload);
                    break;
                    
                case 'error':
                    showError(msg.payload.error);
                    break;
                    
                case 'game_ended':
                    showAISelector();
                    break;
            }
        }
        
        let moveLog = [];
        
        function showError(message) {
            // Remove existing toast if any
            const existing = document.querySelector('.error-toast');
            if (existing) existing.remove();
            
            const toast = document.createElement('div');
            toast.className = 'error-toast';
            toast.textContent = message;
            document.body.appendChild(toast);
            
            setTimeout(() => {
                toast.classList.add('fade-out');
                setTimeout(() => toast.remove(), 300);
            }, 4000);
        }
        
        function addMoveLog(player, result) {
            let text = player + ' fired at ' + result.move + ': ';
            let cls = 'miss';
            
            if (result.sunk) {
                text += 'Sunk ' + result.shipName + '!';
                cls = 'sunk';
            } else if (result.hit) {
                text += 'Hit!';
                cls = 'hit';
            } else {
                text += 'Miss';
            }
            
            moveLog.unshift({ text, cls });
            if (moveLog.length > 20) moveLog.pop();
        }
        
        function renderGame() {
            if (!gameState) return;
            
            let statusClass = 'your-turn';
            let statusText = "Your turn - click enemy board to fire!";
            
            if (gameState.gameOver) {
                statusClass = 'game-over';
                statusText = gameState.winner === 'human' ? '🎉 You Won!' : '💥 AI Wins!';
            } else if (gameState.currentTurn === 'ai') {
                statusClass = 'ai-turn';
                statusText = "AI is thinking...";
            }
            
            let html = '<div class="status-bar ' + statusClass + '">' + statusText + '</div>';
            
            html += '<div class="game-container">';
            
            // Enemy board (for attacking)
            html += '<div class="board-section">';
            html += '<div class="board-title">Enemy Fleet (' + gameState.aiName + ')</div>';
            html += renderBoard(gameState.enemyBoard, true);
            html += renderShips(gameState.enemyShips, true);
            html += '</div>';
            
            // Own board
            html += '<div class="board-section">';
            html += '<div class="board-title">Your Fleet</div>';
            html += renderBoard(gameState.ownBoard, false);
            html += renderShips(gameState.ownShips, false);
            html += '</div>';
            
            html += '</div>';
            
            // Move log
            html += '<div class="move-log"><div class="move-log-title">Battle Log</div><div class="move-log-content">';
            for (const entry of moveLog) {
                html += '<div class="move-entry ' + entry.cls + '">' + entry.text + '</div>';
            }
            html += '</div></div>';
            
            // Play again button
            if (gameState.gameOver) {
                html += '<div style="text-align:center;margin-top:2rem;">' +
                        '<button class="btn" onclick="location.reload()">Play Again</button></div>';
            }
            
            document.getElementById('game-area').innerHTML = html;
        }
        
        function renderBoard(board, isEnemy) {
            const disabled = gameState.currentTurn !== 'human' || gameState.gameOver;
            let html = '<div class="board ' + (isEnemy ? 'enemy' : '') + (disabled ? ' disabled' : '') + '">';
            
            // Header row
            html += '<div class="header-cell"></div>';
            for (let c = 1; c <= 10; c++) {
                html += '<div class="header-cell">' + c + '</div>';
            }
            
            // Board cells
            for (let r = 0; r < 10; r++) {
                html += '<div class="header-cell">' + String.fromCharCode(65 + r) + '</div>';
                
                for (let c = 0; c < 10; c++) {
                    const cell = board[r][c];
                    let cls = 'cell';
                    let content = '';
                    
                    if (cell === CELL_HIT) {
                        cls += ' hit';
                        content = '💥';
                    } else if (cell === CELL_MISS) {
                        cls += ' miss';
                        content = '•';
                    } else if (cell === CELL_SUNK) {
                        cls += ' sunk';
                        content = '☠️';
                    } else if (cell === CELL_SHIP && !isEnemy) {
                        cls += ' ship';
                        content = '🚢';
                    }
                    
                    const onclick = isEnemy && !disabled ? ' onclick="fireAt(' + r + ',' + c + ')"' : '';
                    html += '<div class="' + cls + '"' + onclick + '>' + content + '</div>';
                }
            }
            
            html += '</div>';
            return html;
        }
        
        function renderShips(ships, isEnemy) {
            let html = '<div class="ships-panel"><div class="ships-title">' + 
                       (isEnemy ? 'Enemy Ships' : 'Your Ships') + '</div>';
            
            for (const ship of ships) {
                const sunkClass = ship.sunk ? ' sunk' : '';
                const icon = ship.sunk ? '💀' : '🚢';
                html += '<div class="ship-row' + sunkClass + '">' +
                        '<span class="ship-icon">' + icon + '</span>' +
                        '<span class="ship-name">' + ship.name + ' (' + ship.size + ')</span>' +
                        '</div>';
            }
            
            html += '</div>';
            return html;
        }
        
        function fireAt(row, col) {
            if (!ws || !gameState || gameState.currentTurn !== 'human' || gameState.gameOver) {
                return;
            }
            
            const move = String.fromCharCode(65 + row) + (col + 1);
            
            ws.send(JSON.stringify({
                type: 'move',
                payload: { move: move }
            }));
        }
        
        init();
    </script>
</body>
</html>
`

package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	
	"github.com/go-chi/chi/v5"
	gossh "golang.org/x/crypto/ssh"
	
	"battleship-arena/internal/storage"
)

func HandleUserProfile(w http.ResponseWriter, r *http.Request) {
	username := chi.URLParam(r, "username")
	if username == "" {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	
	user, err := storage.GetUserByUsername(username)
	if err != nil {
		http.Error(w, "Error loading user", http.StatusInternalServerError)
		return
	}
	if user == nil {
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	
	// Get user's submission stats
	entries, _ := storage.GetLeaderboard(100)
	var userEntry *storage.LeaderboardEntry
	for _, e := range entries {
		if e.Username == username {
			userEntry = &e
			break
		}
	}
	
	// Parse public key for display
	publicKeyDisplay := formatPublicKey(user.PublicKey)
	
	tmpl := template.Must(template.New("user").Parse(userProfileHTML))
	data := struct {
		User            *storage.User
		Entry           *storage.LeaderboardEntry
		PublicKeyDisplay string
	}{
		User:             user,
		Entry:            userEntry,
		PublicKeyDisplay: publicKeyDisplay,
	}
	tmpl.Execute(w, data)
}

func HandleUsers(w http.ResponseWriter, r *http.Request) {
	users, err := storage.GetAllUsers()
	if err != nil {
		http.Error(w, "Error loading users", http.StatusInternalServerError)
		return
	}
	
	tmpl := template.Must(template.New("users").Parse(usersListHTML))
	tmpl.Execute(w, users)
}

func formatPublicKey(key string) string {
	key = strings.TrimSpace(key)
	parts := strings.Fields(key)
	if len(parts) < 2 {
		return key
	}
	
	// Parse the key to get fingerprint
	pubKey, _, _, _, err := gossh.ParseAuthorizedKey([]byte(key))
	if err != nil {
		return key
	}
	
	fingerprint := gossh.FingerprintSHA256(pubKey)
	return fmt.Sprintf("%s %s", parts[0], fingerprint)
}

const userProfileHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>{{.User.Name}} (@{{.User.Username}}) - Battleship Arena</title>
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
            max-width: 900px;
            margin: 0 auto;
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
        
        .profile-header {
            background: #1e293b;
            border: 1px solid #334155;
            border-radius: 12px;
            padding: 2rem;
            margin-bottom: 2rem;
        }
        
        .username {
            font-size: 2rem;
            font-weight: 700;
            color: #e2e8f0;
            margin-bottom: 0.5rem;
        }
        
        .handle {
            font-size: 1.2rem;
            color: #94a3b8;
            margin-bottom: 1rem;
        }
        
        .bio {
            color: #cbd5e1;
            margin-bottom: 1rem;
            line-height: 1.6;
        }
        
        .link {
            color: #60a5fa;
            text-decoration: none;
        }
        
        .link:hover {
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
        
        .key-section {
            background: #1e293b;
            border: 1px solid #334155;
            border-radius: 12px;
            padding: 2rem;
        }
        
        .section-title {
            font-size: 1.25rem;
            font-weight: 600;
            margin-bottom: 1rem;
            color: #e2e8f0;
        }
        
        .key-display {
            background: #0f172a;
            padding: 1rem;
            border-radius: 8px;
            font-family: 'Monaco', 'Courier New', monospace;
            font-size: 0.875rem;
            color: #94a3b8;
            word-break: break-all;
        }
        
        .metadata {
            display: grid;
            grid-template-columns: repeat(2, 1fr);
            gap: 1rem;
            margin-top: 1rem;
            font-size: 0.875rem;
            color: #64748b;
        }
    </style>
</head>
<body>
    <div class="container">
        <a href="/" class="back-link">← Back to Leaderboard</a>
        
        <div class="profile-header">
            <div class="username">{{.User.Name}}</div>
            <div class="handle">@{{.User.Username}}</div>
            {{if .User.Bio}}
            <div class="bio">{{.User.Bio}}</div>
            {{end}}
            {{if .User.Link}}
            <a href="{{.User.Link}}" class="link" target="_blank">🔗 {{.User.Link}}</a>
            {{end}}
        </div>
        
        {{if .Entry}}
        <div class="stats-grid">
            <div class="stat-card">
                <div class="stat-label">Rating</div>
                <div class="stat-value">{{.Entry.Rating}}</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Wins</div>
                <div class="stat-value">{{.Entry.Wins}}</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Losses</div>
                <div class="stat-value">{{.Entry.Losses}}</div>
            </div>
            <div class="stat-card">
                <div class="stat-label">Win Rate</div>
                <div class="stat-value">{{printf "%.1f" .Entry.WinPct}}%</div>
            </div>
        </div>
        {{end}}
        
        <div class="key-section">
            <h2 class="section-title">SSH Public Key</h2>
            <div class="key-display">{{.PublicKeyDisplay}}</div>
            <div class="metadata">
                <div>Member since: {{.User.CreatedAt.Format "Jan 2, 2006"}}</div>
                <div>Last login: {{.User.LastLoginAt.Format "Jan 2, 3:04 PM"}}</div>
            </div>
        </div>
    </div>
</body>
</html>
`

const usersListHTML = `
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Users - Battleship Arena</title>
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
        
        .users-grid {
            display: grid;
            grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
            gap: 1.5rem;
        }
        
        .user-card {
            background: #1e293b;
            border: 1px solid #334155;
            border-radius: 12px;
            padding: 1.5rem;
            transition: transform 0.2s, border-color 0.2s;
            text-decoration: none;
            color: inherit;
            display: block;
        }
        
        .user-card:hover {
            transform: translateY(-2px);
            border-color: #60a5fa;
        }
        
        .user-name {
            font-size: 1.25rem;
            font-weight: 600;
            color: #e2e8f0;
            margin-bottom: 0.25rem;
        }
        
        .user-handle {
            font-size: 0.9rem;
            color: #94a3b8;
            margin-bottom: 0.75rem;
        }
        
        .user-bio {
            font-size: 0.875rem;
            color: #cbd5e1;
            line-height: 1.5;
        }
    </style>
</head>
<body>
    <div class="container">
        <a href="/" class="back-link">← Back to Leaderboard</a>
        <h1>Players</h1>
        <p style="color: #94a3b8; margin-bottom: 2rem;">{{len .}} registered users</p>
        
        <div class="users-grid">
            {{range .}}
            <a href="/user/{{.Username}}" class="user-card">
                <div class="user-name">{{.Name}}</div>
                <div class="user-handle">@{{.Username}}</div>
                {{if .Bio}}
                <div class="user-bio">{{.Bio}}</div>
                {{end}}
            </a>
            {{end}}
        </div>
    </div>
</body>
</html>
`

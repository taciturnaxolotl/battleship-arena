package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	
	"battleship-arena/internal/storage"
)

type viewMode int

const (
	viewHome viewMode = iota
	viewLeaderboard
	viewProfile
)

var titleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("205")).
	MarginTop(1).
	MarginBottom(1)

type model struct {
	username     string
	width        int
	height       int
	submissions  []storage.Submission
	leaderboard  []storage.LeaderboardEntry
	matches      []storage.MatchResult
	externalURL  string
	sshPort      string
	currentView  viewMode
}

func InitialModel(username string, width, height int) model {
	externalURL := os.Getenv("BATTLESHIP_EXTERNAL_URL")
	if externalURL == "" {
		externalURL = "localhost"
	}
	// Strip http:// or https:// prefix to get just the hostname
	externalURL = strings.TrimPrefix(externalURL, "http://")
	externalURL = strings.TrimPrefix(externalURL, "https://")
	// Strip port if present
	if idx := strings.Index(externalURL, ":"); idx != -1 {
		externalURL = externalURL[:idx]
	}
	
	sshPort := os.Getenv("BATTLESHIP_SSH_PORT")
	if sshPort == "" {
		sshPort = "2222"
	}
	
	return model{
		username:    username,
		width:       width,
		height:      height,
		submissions: []storage.Submission{},
		leaderboard: []storage.LeaderboardEntry{},
		externalURL: externalURL,
		sshPort:     sshPort,
		currentView: viewHome,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadLeaderboard, loadSubmissions(m.username), tickCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "h", "1":
			m.currentView = viewHome
		case "l", "2":
			m.currentView = viewLeaderboard
		case "p", "3":
			m.currentView = viewProfile
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case leaderboardMsg:
		m.leaderboard = msg.entries
	case submissionsMsg:
		m.submissions = msg.submissions
	case matchesMsg:
		m.matches = msg.matches
	case tickMsg:
		return m, tea.Batch(loadLeaderboard, loadSubmissions(m.username), loadMatches, tickCmd())
	}
	return m, nil
}



func (m model) View() string {
	var b strings.Builder

	title := titleStyle.Render("🚢 Battleship Arena")
	b.WriteString(title + "\n")
	
	// Navigation tabs
	tabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	activeTabStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86")).Bold(true)
	
	tabs := []string{"[h] Home", "[l] Leaderboard", "[p] Profile"}
	for i, tab := range tabs {
		if viewMode(i) == m.currentView {
			b.WriteString(activeTabStyle.Render(tab))
		} else {
			b.WriteString(tabStyle.Render(tab))
		}
		if i < len(tabs)-1 {
			b.WriteString("  ")
		}
	}
	b.WriteString("\n\n")

	// Render content based on current view
	switch m.currentView {
	case viewHome:
		b.WriteString(m.renderHome())
	case viewLeaderboard:
		b.WriteString(m.renderLeaderboardView())
	case viewProfile:
		b.WriteString(m.renderProfile())
	}

	b.WriteString("\n\nPress q to quit")

	return b.String()
}

func (m model) renderHome() string {
	var b strings.Builder
	
	b.WriteString(fmt.Sprintf("User: %s\n\n", m.username))

	// Upload instructions
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	b.WriteString(infoStyle.Render(fmt.Sprintf("Upload via: scp -P %s memory_functions_yourname.cpp %s@%s:~/", m.sshPort, m.username, m.externalURL)))
	b.WriteString("\n\n")

	// Show submissions
	if len(m.submissions) > 0 {
		b.WriteString(renderSubmissions(m.submissions))
	} else {
		b.WriteString("No submissions yet. Upload your first AI!\n")
	}

	return b.String()
}

func (m model) renderLeaderboardView() string {
	if len(m.leaderboard) > 0 {
		return renderLeaderboard(m.leaderboard)
	}
	return "Loading leaderboard..."
}

func (m model) renderProfile() string {
	var b strings.Builder
	
	b.WriteString(fmt.Sprintf("Profile: %s\n\n", m.username))
	
	// Show user stats from submissions
	if len(m.submissions) > 0 {
		b.WriteString(renderSubmissions(m.submissions))
		b.WriteString("\n")
	}
	
	// Show recent matches involving this user
	if len(m.matches) > 0 {
		b.WriteString("\nRecent Matches:\n")
		b.WriteString(renderMatches(m.matches, m.username))
	}
	
	return b.String()
}



type leaderboardMsg struct {
	entries []storage.LeaderboardEntry
}

func loadLeaderboard() tea.Msg {
	entries, err := storage.GetLeaderboard(20)
	if err != nil {
		return leaderboardMsg{entries: nil}
	}
	return leaderboardMsg{entries: entries}
}

type submissionsMsg struct {
	submissions []storage.Submission
}

func loadSubmissions(username string) tea.Cmd {
	return func() tea.Msg {
		submissions, err := storage.GetUserSubmissions(username)
		if err != nil {
			return submissionsMsg{submissions: nil}
		}
		return submissionsMsg{submissions: submissions}
	}
}

type matchesMsg struct {
	matches []storage.MatchResult
}

func loadMatches() tea.Msg {
	matches, err := storage.GetAllMatches()
	if err != nil {
		return matchesMsg{matches: nil}
	}
	return matchesMsg{matches: matches}
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func renderSubmissions(submissions []storage.Submission) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("📤 Your Submissions") + "\n\n")

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240"))
	b.WriteString(headerStyle.Render(fmt.Sprintf("%-35s %-15s %s",
		"Filename", "Uploaded", "Status")) + "\n")

	for _, sub := range submissions {
		var statusColor string
		switch sub.Status {
		case "pending":
			statusColor = "yellow"
		case "testing":
			statusColor = "blue"
		case "completed":
			statusColor = "green"
		case "failed":
			statusColor = "red"
		default:
			statusColor = "white"
		}

		relTime := formatRelativeTime(sub.UploadTime)
		
		// Build the line manually to avoid formatting issues with ANSI codes
		line := fmt.Sprintf("%-35s %-15s ", sub.Filename, relTime)
		statusStyled := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(sub.Status)
		b.WriteString(line + statusStyled + "\n")
	}

	return b.String()
}

func formatRelativeTime(t time.Time) string {
	duration := time.Since(t)
	if duration < time.Minute {
		return "just now"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh ago", hours)
	}
	days := int(duration.Hours() / 24)
	return fmt.Sprintf("%dd ago", days)
}

func renderLeaderboard(entries []storage.LeaderboardEntry) string {
	if len(entries) == 0 {
		return "No entries yet"
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("🏆 Leaderboard") + "\n\n")

	// Header without styling on the whole line
	b.WriteString(fmt.Sprintf("%-4s %-20s %11s %8s %8s %10s %10s\n", 
		"Rank", "User", "Rating", "Wins", "Losses", "Win Rate", "Avg Moves"))

	for i, entry := range entries {
		rank := fmt.Sprintf("#%d", i+1)
		
		// Apply color only to the rank and pad manually
		var displayRank string
		if i == 0 {
			displayRank = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(rank) + "  " // Gold
		} else if i == 1 {
			displayRank = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render(rank) + "  " // Silver
		} else if i == 2 {
			displayRank = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render(rank) + "  " // Bronze
		} else {
			displayRank = fmt.Sprintf("%-4s", rank)
		}
		
		// Format line with Glicko-2 rating ± RD
		ratingStr := fmt.Sprintf("%d±%d", entry.Rating, entry.RD)
		b.WriteString(fmt.Sprintf("%s %-20s %11s %8d %8d %9.2f%% %9.1f\n",
			displayRank, entry.Username, ratingStr, entry.Wins, entry.Losses, entry.WinPct, entry.AvgMoves))
	}

	return b.String()
}

func renderMatches(matches []storage.MatchResult, username string) string {
	var b strings.Builder
	
	// Filter to show only matches involving this user
	userMatches := []storage.MatchResult{}
	for _, match := range matches {
		if match.Player1Username == username || match.Player2Username == username {
			userMatches = append(userMatches, match)
		}
	}
	
	if len(userMatches) == 0 {
		return "No matches yet"
	}
	
	// Limit to last 10 matches
	limit := 10
	if len(userMatches) > limit {
		userMatches = userMatches[:limit]
	}
	
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240"))
	b.WriteString(headerStyle.Render(fmt.Sprintf("%-20s %-20s %-20s %10s\n",
		"Player 1", "Player 2", "Winner", "Avg Moves")))
	b.WriteString("\n")
	
	for _, match := range userMatches {
		line := fmt.Sprintf("%-20s vs %-20s → %-20s (%d moves)\n",
			match.Player1Username, match.Player2Username, match.WinnerUsername, match.AvgMoves)
		
		// Highlight wins in green, losses in red
		if match.WinnerUsername == username {
			b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("green")).Render(line))
		} else {
			b.WriteString(line)
		}
	}
	
	return b.String()
}

func renderBracket(matches []storage.MatchResult) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("⚔️  Tournament Bracket") + "\n\n")

	if len(matches) == 0 {
		return b.String()
	}

	// Group matches by matchup pairs
	matchups := make(map[string]storage.MatchResult)
	for _, match := range matches {
		// Create a consistent key regardless of order
		key := match.Player1Username + " vs " + match.Player2Username
		reverseKey := match.Player2Username + " vs " + match.Player1Username
		
		// Check if we already have this matchup
		if _, exists := matchups[reverseKey]; !exists {
			matchups[key] = match
		}
	}

	// Display up to 8 matchups in bracket format
	count := 0
	for _, match := range matchups {
		if count >= 8 {
			break
		}
		
		// Determine winner styling
		player1Style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		player2Style := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
		winnerBox := lipgloss.NewStyle().
			Foreground(lipgloss.Color("green")).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)
		
		var winner string
		if match.WinnerUsername == match.Player1Username {
			player1Style = player1Style.Foreground(lipgloss.Color("green")).Bold(true)
			winner = match.Player1Username
		} else {
			player2Style = player2Style.Foreground(lipgloss.Color("green")).Bold(true)
			winner = match.Player2Username
		}
		
		// Format bracket style
		// Player1  ┐
		//          ├── Winner
		// Player2  ┘
		
		player1Box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			Width(15)
		
		player2Box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1).
			Width(15)
		
		p1 := player1Box.Render(player1Style.Render(match.Player1Username))
		connector1 := "  ┐"
		middle := "   ├──"
		connector2 := "  ┘"
		p2 := player2Box.Render(player2Style.Render(match.Player2Username))
		winnerStr := winnerBox.Render(fmt.Sprintf("%s wins", winner))
		
		b.WriteString(p1 + connector1 + "\n")
		b.WriteString(strings.Repeat(" ", 17) + middle + " " + winnerStr + "\n")
		b.WriteString(p2 + connector2 + "\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(
			fmt.Sprintf("                        (avg %d moves)\n", match.AvgMoves)))
		b.WriteString("\n")
		
		count++
	}

	if len(matchups) > 8 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(
			fmt.Sprintf("... and %d more matches\n", len(matchups)-8)))
	}

	return b.String()
}

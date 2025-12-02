package main

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	username     string
	width        int
	height       int
	submissions  []Submission
	leaderboard  []LeaderboardEntry
	matches      []MatchResult
}

func initialModel(username string, width, height int) model {
	return model{
		username:    username,
		width:       width,
		height:      height,
		submissions: []Submission{},
		leaderboard: []LeaderboardEntry{},
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(loadLeaderboard, loadSubmissions(m.username), loadMatches, tickCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
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
	b.WriteString(title + "\n\n")
	
	b.WriteString(fmt.Sprintf("User: %s\n\n", m.username))

	// Upload instructions
	infoStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("86"))
	b.WriteString(infoStyle.Render(fmt.Sprintf("Upload via: scp -P %s memory_functions_yourname.cpp %s@%s:~/", sshPort, m.username, host)))
	b.WriteString("\n\n")

	// Show submissions
	if len(m.submissions) > 0 {
		b.WriteString(renderSubmissions(m.submissions))
		b.WriteString("\n")
	}

	// Show bracket-style matches
	if len(m.matches) > 0 {
		b.WriteString(renderBracket(m.matches))
		b.WriteString("\n")
	}

	// Show leaderboard if loaded
	if len(m.leaderboard) > 0 {
		b.WriteString(renderLeaderboard(m.leaderboard))
	}

	b.WriteString("\n\nPress q to quit")

	return b.String()
}



type leaderboardMsg struct {
	entries []LeaderboardEntry
}

func loadLeaderboard() tea.Msg {
	entries, err := getLeaderboard(20)
	if err != nil {
		return leaderboardMsg{entries: nil}
	}
	return leaderboardMsg{entries: entries}
}

type submissionsMsg struct {
	submissions []Submission
}

func loadSubmissions(username string) tea.Cmd {
	return func() tea.Msg {
		submissions, err := getUserSubmissions(username)
		if err != nil {
			return submissionsMsg{submissions: nil}
		}
		return submissionsMsg{submissions: submissions}
	}
}

type matchesMsg struct {
	matches []MatchResult
}

func loadMatches() tea.Msg {
	matches, err := getAllMatches()
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

func renderSubmissions(submissions []Submission) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("📤 Your Submissions") + "\n\n")

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240"))
	b.WriteString(headerStyle.Render(fmt.Sprintf("%-35s %-15s %s\n",
		"Filename", "Uploaded", "Status")))

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
		
		// Format the line without styles first for proper alignment
		statusStyled := lipgloss.NewStyle().Foreground(lipgloss.Color(statusColor)).Render(sub.Status)
		b.WriteString(fmt.Sprintf("%-35s %-15s %s\n",
			sub.Filename, relTime, statusStyled))
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

func renderLeaderboard(entries []LeaderboardEntry) string {
	if len(entries) == 0 {
		return "No entries yet"
	}

	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("🏆 Leaderboard") + "\n\n")

	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("240"))
	b.WriteString(headerStyle.Render(fmt.Sprintf("%-4s %-20s %8s %8s %10s\n", 
		"Rank", "User", "Wins", "Losses", "Win Rate")))

	for i, entry := range entries {
		winRate := 0.0
		total := entry.Wins + entry.Losses
		if total > 0 {
			winRate = float64(entry.Wins) / float64(total) * 100
		}

		rank := fmt.Sprintf("#%d", i+1)
		line := fmt.Sprintf("%-4s %-20s %8d %8d %9.2f%%\n",
			rank, entry.Username, entry.Wins, entry.Losses, winRate)

		style := lipgloss.NewStyle()
		if i == 0 {
			style = style.Foreground(lipgloss.Color("220")) // Gold
		} else if i == 1 {
			style = style.Foreground(lipgloss.Color("250")) // Silver
		} else if i == 2 {
			style = style.Foreground(lipgloss.Color("208")) // Bronze
		}

		b.WriteString(style.Render(line))
	}

	return b.String()
}

func renderBracket(matches []MatchResult) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("⚔️  Recent Matches") + "\n\n")

	if len(matches) == 0 {
		return b.String()
	}

	// Show most recent matches (up to 10)
	displayCount := len(matches)
	if displayCount > 10 {
		displayCount = 10
	}

	for i := 0; i < displayCount; i++ {
		match := matches[i]
		
		// Determine styling based on winner
		player1Style := lipgloss.NewStyle()
		player2Style := lipgloss.NewStyle()
		
		if match.WinnerUsername == match.Player1Username {
			player1Style = player1Style.Foreground(lipgloss.Color("green")).Bold(true)
			player2Style = player2Style.Foreground(lipgloss.Color("240"))
		} else {
			player2Style = player2Style.Foreground(lipgloss.Color("green")).Bold(true)
			player1Style = player1Style.Foreground(lipgloss.Color("240"))
		}
		
		// Format: [Player1] ──vs── [Player2]  →  Winner (avg moves)
		player1Str := player1Style.Render(fmt.Sprintf("%-15s", match.Player1Username))
		vsStr := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" ──vs── ")
		player2Str := player2Style.Render(fmt.Sprintf("%-15s", match.Player2Username))
		
		winnerMark := "→"
		winnerStr := lipgloss.NewStyle().Foreground(lipgloss.Color("green")).Render(
			fmt.Sprintf("%s %s wins (avg %d moves)", winnerMark, match.WinnerUsername, match.AvgMoves))
		
		b.WriteString(fmt.Sprintf("%s%s%s  %s\n", player1Str, vsStr, player2Str, winnerStr))
	}

	return b.String()
}

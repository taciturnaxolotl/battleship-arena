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
	return tea.Batch(loadLeaderboard, loadSubmissions(m.username), tickCmd())
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
	case tickMsg:
		return m, tea.Batch(loadLeaderboard, loadSubmissions(m.username), tickCmd())
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

	// Header without styling on the whole line
	b.WriteString(fmt.Sprintf("%-4s %-20s %6s %8s %8s %10s %10s\n", 
		"Rank", "User", "ELO", "Wins", "Losses", "Win Rate", "Avg Moves"))

	for i, entry := range entries {
		rank := fmt.Sprintf("#%d", i+1)
		
		// Apply color only to the rank
		var coloredRank string
		if i == 0 {
			coloredRank = lipgloss.NewStyle().Foreground(lipgloss.Color("220")).Render(rank) // Gold
		} else if i == 1 {
			coloredRank = lipgloss.NewStyle().Foreground(lipgloss.Color("250")).Render(rank) // Silver
		} else if i == 2 {
			coloredRank = lipgloss.NewStyle().Foreground(lipgloss.Color("208")).Render(rank) // Bronze
		} else {
			coloredRank = rank
		}
		
		// Format line with proper spacing
		b.WriteString(fmt.Sprintf("%-4s %-20s %6d %8d %8d %9.2f%% %9.1f\n",
			coloredRank, entry.Username, entry.Elo, entry.Wins, entry.Losses, entry.WinPct, entry.AvgMoves))
	}

	return b.String()
}

func renderBracket(matches []MatchResult) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("⚔️  Tournament Bracket") + "\n\n")

	if len(matches) == 0 {
		return b.String()
	}

	// Group matches by matchup pairs
	matchups := make(map[string]MatchResult)
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

package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type menuChoice int

const (
	menuUpload menuChoice = iota
	menuLeaderboard
	menuSubmit
	menuHelp
	menuQuit
)

type model struct {
	username     string
	width        int
	height       int
	choice       menuChoice
	submitting   bool
	filename     string
	fileContent  []byte
	message      string
	leaderboard  []LeaderboardEntry
}

func initialModel(username string, width, height int) model {
	return model{
		username: username,
		width:    width,
		height:   height,
		choice:   menuUpload,
	}
}

func (m model) Init() tea.Cmd {
	return loadLeaderboard
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.choice > 0 {
				m.choice--
			}
		case "down", "j":
			if m.choice < menuQuit {
				m.choice++
			}
		case "enter":
			return m.handleSelection()
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case leaderboardMsg:
		m.leaderboard = msg.entries
	}
	return m, nil
}

func (m model) handleSelection() (tea.Model, tea.Cmd) {
	switch m.choice {
	case menuUpload:
		m.message = fmt.Sprintf("Upload via SCP:\nscp -P %s memory_functions_yourname.cpp %s@%s:~/", sshPort, m.username, host)
		return m, nil
	case menuLeaderboard:
		return m, loadLeaderboard
	case menuSubmit:
		m.message = "Submission queued for testing..."
		return m, submitForTesting(m.username)
	case menuHelp:
		helpText := `Battleship Arena - How to Compete

1. Create your AI implementation (memory_functions_*.cpp)
2. Upload via SCP from your terminal:
   scp -P ` + sshPort + ` memory_functions_yourname.cpp ` + m.username + `@` + host + `:~/
3. Select "Test Submission" to queue your AI for testing
4. Check the leaderboard to see your ranking!

Your AI will be tested against the random AI baseline.
Win rate and average moves determine your ranking.`
		m.message = helpText
		return m, nil
	case menuQuit:
		return m, tea.Quit
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	title := titleStyle.Render("🚢 Battleship Arena")
	b.WriteString(title + "\n\n")
	
	b.WriteString(fmt.Sprintf("User: %s\n\n", m.username))

	// Menu
	menuStyle := lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("170")).
		Bold(true).
		PaddingLeft(1)
	
	for i := menuChoice(0); i <= menuQuit; i++ {
		cursor := " "
		style := menuStyle
		if i == m.choice {
			cursor = ">"
			style = selectedStyle
		}
		b.WriteString(style.Render(fmt.Sprintf("%s %s\n", cursor, menuText(i))))
	}

	if m.message != "" {
		b.WriteString("\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Render(m.message) + "\n")
	}

	// Show leaderboard if loaded
	if len(m.leaderboard) > 0 {
		b.WriteString("\n" + renderLeaderboard(m.leaderboard))
	}

	b.WriteString("\n\nPress q to quit, ↑/↓ to navigate, enter to select")

	return b.String()
}

func menuText(c menuChoice) string {
	switch c {
	case menuUpload:
		return "Upload Submission"
	case menuLeaderboard:
		return "View Leaderboard"
	case menuSubmit:
		return "Test Submission"
	case menuHelp:
		return "Help"
	case menuQuit:
		return "Quit"
	default:
		return "Unknown"
	}
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

type submitMsg struct {
	success bool
	message string
}

func submitForTesting(username string) tea.Cmd {
	return func() tea.Msg {
		// Queue submission for testing
		if err := queueSubmission(username); err != nil {
			return submitMsg{success: false, message: err.Error()}
		}
		return submitMsg{success: true, message: "Submitted successfully!"}
	}
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

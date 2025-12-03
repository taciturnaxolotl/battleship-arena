package tui

import (
	"fmt"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	
	"battleship-arena/internal/storage"
)

type OnboardingModel struct {
	username  string
	publicKey string
	step      int // 0=name, 1=bio, 2=link, 3=done
	name      string
	bio       string
	link      string
	input     string
	err       error
	width     int
	height    int
	completed bool
}

type onboardingCompleteMsg struct {
	username string
}

func NewOnboardingModel(username, publicKey string, width, height int) OnboardingModel {
	return OnboardingModel{
		username:  username,
		publicKey: publicKey,
		step:      0,
		width:     width,
		height:    height,
		completed: false,
	}
}

func (m OnboardingModel) Init() tea.Cmd {
	return nil
}

func (m OnboardingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			switch m.step {
			case 0: // Name
				if strings.TrimSpace(m.input) == "" {
					m.err = fmt.Errorf("name is required")
					return m, nil
				}
				m.name = strings.TrimSpace(m.input)
				m.input = ""
				m.err = nil
				m.step = 1
			case 1: // Bio
				m.bio = strings.TrimSpace(m.input)
				m.input = ""
				m.err = nil
				m.step = 2
			case 2: // Link
				m.link = strings.TrimSpace(m.input)
				m.step = 3
				
				// Create user in database
				_, err := storage.CreateUser(m.username, m.name, m.bio, m.link, m.publicKey)
				if err != nil {
					log.Printf("Failed to create user: %v", err)
					m.err = fmt.Errorf("failed to create account")
					m.step = 2
					return m, nil
				}
				
				m.completed = true
				return m, func() tea.Msg {
					return onboardingCompleteMsg{username: m.username}
				}
			}
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.input += msg.String()
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case onboardingCompleteMsg:
		// Transition to main model
		mainModel := InitialModel(m.username, m.width, m.height)
		return mainModel, mainModel.Init()
	}
	return m, nil
}

func (m OnboardingModel) View() string {
	if m.completed {
		successStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("green")).
			Bold(true)
		return successStyle.Render("\n✅ Account created successfully!\n\nLoading dashboard...\n")
	}
	
	var b strings.Builder
	
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginTop(1).
		MarginBottom(1)
	
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("86"))
	
	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("212")).
		Bold(true)
	
	errorStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196"))
	
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))
	
	b.WriteString(titleStyle.Render("🚢 Welcome to Battleship Arena!"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Setting up account for: %s\n\n", m.username))
	
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("❌ %s\n\n", m.err.Error())))
	}
	
	switch m.step {
	case 0:
		b.WriteString(promptStyle.Render("What's your full name?") + " (required)\n")
		b.WriteString(inputStyle.Render(m.input + "█") + "\n\n")
		b.WriteString(helpStyle.Render("Press Enter to continue"))
	case 1:
		b.WriteString(promptStyle.Render("Bio:") + " (optional, press Enter to skip)\n")
		b.WriteString(inputStyle.Render(m.input + "█") + "\n\n")
		b.WriteString(helpStyle.Render("A short description about yourself"))
	case 2:
		b.WriteString(promptStyle.Render("Link:") + " (optional, press Enter to skip)\n")
		b.WriteString(inputStyle.Render(m.input + "█") + "\n\n")
		b.WriteString(helpStyle.Render("Website, GitHub, or social media link"))
	}
	
	return b.String()
}

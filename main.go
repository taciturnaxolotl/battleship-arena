package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/charmbracelet/wish/scp"
)

const (
	host      = "0.0.0.0"
	sshPort   = "2222"
	webPort   = "8080"
	uploadDir = "./submissions"
	resultsDB = "./results.db"
)

func main() {
	// Initialize storage
	if err := initStorage(); err != nil {
		log.Fatal(err)
	}

	// Start background worker
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go startWorker(workerCtx)

	// Start web server
	go startWebServer()

	// Start SSH server with TUI, SCP, and SFTP
	toClient, fromClient := newSCPHandlers()
	s, err := wish.NewServer(
		wish.WithAddress(host + ":" + sshPort),
		wish.WithHostKeyPath(".ssh/battleship_arena"),
		wish.WithSubsystem("sftp", sftpHandler),
		wish.WithMiddleware(
			scp.Middleware(toClient, fromClient),
			bubbletea.Middleware(teaHandler),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("SSH server listening on %s:%s", host, sshPort)
	log.Printf("Web leaderboard at http://%s:%s", host, webPort)

	go func() {
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	<-done
	log.Println("Shutting down servers...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		log.Fatal(err)
	}
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	// Don't handle non-interactive sessions (SCP/SFTP have commands)
	if len(s.Command()) > 0 {
		return nil, nil
	}
	
	pty, _, active := s.Pty()
	if !active {
		wish.Fatalln(s, "no active terminal")
		return nil, nil
	}

	m := initialModel(s.User(), pty.Window.Width, pty.Window.Height)
	return m, []tea.ProgramOption{tea.WithAltScreen()}
}

func initStorage() error {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return err
	}
	
	db, err := initDB(resultsDB)
	if err != nil {
		return err
	}
	globalDB = db
	
	return nil
}

func startWebServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleLeaderboard)
	mux.HandleFunc("/api/leaderboard", handleAPILeaderboard)
	mux.HandleFunc("/api/bracket", handleBracketData)
	
	// Serve static files
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	
	log.Printf("Web server starting on :%s", webPort)
	if err := http.ListenAndServe(":"+webPort, mux); err != nil {
		log.Fatal(err)
	}
}

var titleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("205")).
	MarginTop(1).
	MarginBottom(1)

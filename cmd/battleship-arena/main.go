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
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	
	"battleship-arena/internal/runner"
	"battleship-arena/internal/server"
	"battleship-arena/internal/storage"
	"battleship-arena/internal/tui"
)

type Config struct {
	Host             string
	SSHPort          string
	WebPort          string
	UploadDir        string
	ResultsDB        string
	AdminPasscode    string
	ExternalURL      string
}

func loadConfig() Config {
	cfg := Config{
		Host:             getEnv("BATTLESHIP_HOST", "0.0.0.0"),
		SSHPort:          getEnv("BATTLESHIP_SSH_PORT", "2222"),
		WebPort:          getEnv("BATTLESHIP_WEB_PORT", "8081"),
		UploadDir:        getEnv("BATTLESHIP_UPLOAD_DIR", "./submissions"),
		ResultsDB:        getEnv("BATTLESHIP_RESULTS_DB", "./results.db"),
		AdminPasscode:    getEnv("BATTLESHIP_ADMIN_PASSCODE", "battleship-admin-override"),
		ExternalURL:      getEnv("BATTLESHIP_EXTERNAL_URL", "http://localhost:8081"),
	}
	return cfg
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func main() {
	cfg := loadConfig()
	
	if err := initStorage(cfg); err != nil {
		log.Fatal(err)
	}

	server.InitSSE()
	server.SetConfig(cfg.AdminPasscode, cfg.ExternalURL)

	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go runner.StartWorker(workerCtx, cfg.UploadDir, server.BroadcastProgress, server.NotifyLeaderboardUpdate, server.BroadcastProgressComplete)

	toClient, fromClient := server.NewSCPHandlers(cfg.UploadDir)
	sshServer, err := wish.NewServer(
		wish.WithAddress(cfg.Host + ":" + cfg.SSHPort),
		wish.WithHostKeyPath(".ssh/battleship_arena"),
		wish.WithPublicKeyAuth(server.PublicKeyAuthHandler),
		wish.WithPasswordAuth(server.PasswordAuthHandler),
		wish.WithSubsystem("sftp", server.SFTPHandler(cfg.UploadDir)),
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

	log.Printf("SSH server listening on %s:%s", cfg.Host, cfg.SSHPort)
	log.Printf("Web leaderboard at %s", cfg.ExternalURL)

	go func() {
		if err := sshServer.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	go func() {
		<-done
		log.Println("Shutting down servers...")
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := sshServer.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Mount("/events/", server.SSEServer)
	r.Get("/api/leaderboard", server.HandleAPILeaderboard)
	r.Get("/api/rating-history/{player}", server.HandleRatingHistory)
	r.Get("/player/{player}", server.HandlePlayerPage)
	r.Get("/user/{username}", server.HandleUserProfile)
	r.Get("/users", server.HandleUsers)
	r.Get("/", server.HandleLeaderboard)

	log.Println("Server running at " + cfg.ExternalURL)
	http.ListenAndServe(":"+cfg.WebPort, r)
}

func teaHandler(s ssh.Session) (tea.Model, []tea.ProgramOption) {
	if len(s.Command()) > 0 {
		return nil, nil
	}
	
	// Check if user needs onboarding
	needsOnboarding := false
	if val := s.Context().Value("needs_onboarding"); val != nil {
		needsOnboarding = val.(bool)
	}
	
	pty, _, active := s.Pty()
	if !active {
		wish.Fatalln(s, "no active terminal")
		return nil, nil
	}
	
	// Get proper terminal options for color support
	opts := bubbletea.MakeOptions(s)
	
	if needsOnboarding {
		// Run onboarding first
		publicKey := ""
		if val := s.Context().Value("public_key"); val != nil {
			publicKey = val.(string)
		}
		
		m := tui.NewOnboardingModel(s.User(), publicKey, pty.Window.Width, pty.Window.Height)
		return m, opts
	}

	m := tui.InitialModel(s.User(), pty.Window.Width, pty.Window.Height)
	return m, opts
}

func initStorage(cfg Config) error {
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		return err
	}
	
	db, err := storage.InitDB(cfg.ResultsDB)
	if err != nil {
		return err
	}
	storage.DB = db
	
	return nil
}

var titleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("205")).
	MarginTop(1).
	MarginBottom(1)

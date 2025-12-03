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
	"github.com/alexandrevicenzi/go-sse"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	host      = "0.0.0.0"
	sshPort   = "2222"
	webPort   = "8081"
	uploadDir = "./submissions"
	resultsDB = "./results.db"
)

func main() {
	// Initialize storage
	if err := initStorage(); err != nil {
		log.Fatal(err)
	}

	// Initialize SSE server EXACTLY like test
	s := sse.NewServer(nil)
	defer s.Shutdown()
	sseServer = s

	// Start background worker
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	go startWorker(workerCtx)

	// Start SSH server in background
	toClient, fromClient := newSCPHandlers()
	sshServer, err := wish.NewServer(
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
		if err := sshServer.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	// Graceful shutdown handler
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

	// Start web server EXACTLY like test
	r := chi.NewRouter()
	
	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	
	// SSE endpoint - mounted directly to router
	r.Mount("/events/", s)
	
	// API routes
	r.Get("/api/leaderboard", handleAPILeaderboard)
	r.Get("/api/rating-history/{player}", handleRatingHistory)
	
	// Player pages
	r.Get("/player/{player}", handlePlayerPage)
	
	// Home page
	r.Get("/", handleLeaderboard)
	
	// Static files
	fs := http.FileServer(http.Dir("./static"))
	r.Handle("/static/*", http.StripPrefix("/static/", fs))

	log.Println("Server running at http://localhost:" + webPort)
	http.ListenAndServe(":"+webPort, r)
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
	
	// SSE endpoint with explicit logging
	mux.HandleFunc("/events/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("SSE request received: %s", r.URL.Path)
		
		// Try to manually write headers and flush BEFORE SSE library
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("X-Accel-Buffering", "no")
		
		log.Printf("Headers set, writing header...")
		w.WriteHeader(http.StatusOK)
		
		if flusher, ok := w.(http.Flusher); ok {
			log.Printf("Flushing headers manually...")
			flusher.Flush()
			log.Printf("Headers flushed!")
		} else {
			log.Printf("NO FLUSHER!")
		}
		
		log.Printf("Calling SSE ServeHTTP...")
		sseServer.ServeHTTP(w, r)
		log.Printf("SSE ServeHTTP returned")
	})
	
	// Web routes (no Chi)
	mux.HandleFunc("/", handleLeaderboard)
	mux.HandleFunc("/api/leaderboard", handleAPILeaderboard)
	mux.HandleFunc("/api/rating-history/", handleRatingHistory)
	mux.HandleFunc("/player/", handlePlayerPage)
	
	// Static files
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))
	
	log.Printf("Web server starting on :%s", webPort)
	http.ListenAndServe(":"+webPort, mux)
}



var titleStyle = lipgloss.NewStyle().
	Bold(true).
	Foreground(lipgloss.Color("205")).
	MarginTop(1).
	MarginBottom(1)

package server

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish/scp"
	
	"battleship-arena/internal/storage"
)

func NewSCPHandlers(uploadDir string) (scp.CopyToClientHandler, scp.CopyFromClientHandler) {
	baseHandler := scp.NewFileSystemHandler(uploadDir)
	
	uploadHandler := &validatingHandler{
		baseHandler: baseHandler,
		uploadDir:   uploadDir,
	}
	
	return nil, uploadHandler
}

type validatingHandler struct {
	baseHandler scp.CopyFromClientHandler
	uploadDir   string
}

func (h *validatingHandler) Write(s ssh.Session, entry *scp.FileEntry) (int64, error) {
	filename := filepath.Base(entry.Name)
	log.Printf("SCP Write called: entry.Name=%s, filename=%s, size=%d", entry.Name, filename, entry.Size)
	
	// Skip validation for directory markers (SCP protocol negotiation)
	if filename == "~" || filename == "." || filename == ".." {
		log.Printf("Skipping directory marker: %s", filename)
		return 0, nil
	}
	
	// Validate filename (must be memory_functions_*.cpp)
	if !strings.HasPrefix(filename, "memory_functions_") || !strings.HasSuffix(filename, ".cpp") {
		log.Printf("Invalid filename from %s: %s", s.User(), filename)
		return 0, fmt.Errorf("only memory_functions_*.cpp files are accepted")
	}

	userDir := filepath.Join(h.uploadDir, s.User())
	if err := os.MkdirAll(userDir, 0755); err != nil {
		log.Printf("Failed to create user directory: %v", err)
		return 0, err
	}

	// Remove old file if it exists to ensure clean overwrite
	targetPath := filepath.Join(userDir, filename)
	if _, err := os.Stat(targetPath); err == nil {
		log.Printf("Removing old file: %s", targetPath)
		os.Remove(targetPath)
	}

	// Modify the entry to write to user's subdirectory
	userEntry := &scp.FileEntry{
		Name:   filepath.Join(s.User(), filename),
		Mode:   entry.Mode,
		Size:   entry.Size,
		Reader: entry.Reader,
	}
	
	log.Printf("Writing to: %s", filepath.Join(h.uploadDir, userEntry.Name))

	n, err := h.baseHandler.Write(s, userEntry)
	if err != nil {
		log.Printf("Write error: %v", err)
		return n, err
	}

	log.Printf("Uploaded %s from %s (%d bytes)", filename, s.User(), n)
	
	// Add submission and trigger testing
	submissionID, err := storage.AddSubmission(s.User(), filename)
	if err != nil {
		log.Printf("Failed to add submission: %v", err)
	} else {
		log.Printf("Queued submission %d for testing", submissionID)
		// The worker will pick it up automatically
	}
	
	return n, nil
}

func (h *validatingHandler) Mkdir(s ssh.Session, entry *scp.DirEntry) error {
	// Allow mkdir but namespace it to user directory
	userEntry := &scp.DirEntry{
		Name: filepath.Join(s.User(), entry.Name),
		Mode: entry.Mode,
	}
	return h.baseHandler.Mkdir(s, userEntry)
}

package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
)

// Add SCP support as a custom middleware
func scpMiddleware() wish.Middleware {
	return func(sh ssh.Handler) ssh.Handler {
		return func(s ssh.Session) {
			cmd := s.Command()
			if len(cmd) > 0 && cmd[0] == "scp" {
				handleSCP(s, cmd)
				return
			}
			sh(s)
		}
	}
}

func handleSCP(s ssh.Session, cmd []string) {
	// Parse SCP command
	target := false
	filename := ""
	
	for i, arg := range cmd {
		if arg == "-t" {
			target = true
		} else if i == len(cmd)-1 {
			filename = filepath.Base(arg)
		}
	}

	if !target {
		log.Printf("SCP source mode not supported from %s", s.User())
		fmt.Fprintf(s, "SCP source mode not supported\n")
		s.Exit(1)
		return
	}

	// Validate filename
	matched, _ := filepath.Match("memory_functions_*.cpp", filename)
	if !matched {
		log.Printf("Invalid filename from %s: %s", s.User(), filename)
		fmt.Fprintf(s, "Only memory_functions_*.cpp files are accepted\n")
		s.Exit(1)
		return
	}

	// Create user directory
	userDir := filepath.Join(uploadDir, s.User())
	if err := os.MkdirAll(userDir, 0755); err != nil {
		log.Printf("Failed to create user directory: %v", err)
		s.Exit(1)
		return
	}

	// SCP protocol: send 0 byte to indicate ready
	fmt.Fprintf(s, "\x00")

	// Read SCP header (C0644 size filename)
	buf := make([]byte, 1024)
	n, err := s.Read(buf)
	if err != nil {
		log.Printf("Failed to read SCP header: %v", err)
		s.Exit(1)
		return
	}

	// Acknowledge header
	fmt.Fprintf(s, "\x00")

	// Save file
	dstPath := filepath.Join(userDir, filename)
	file, err := os.Create(dstPath)
	if err != nil {
		log.Printf("Failed to create file: %v", err)
		s.Exit(1)
		return
	}
	defer file.Close()

	// Read file content
	_, err = io.Copy(file, io.LimitReader(s, int64(n)))
	if err != nil && err != io.EOF {
		log.Printf("Failed to write file: %v", err)
		s.Exit(1)
		return
	}

	// Final acknowledgment
	fmt.Fprintf(s, "\x00")

	log.Printf("Uploaded %s from %s", filename, s.User())
	addSubmission(s.User(), filename)
	
	s.Exit(0)
}

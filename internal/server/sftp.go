package server

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/pkg/sftp"
	
	"battleship-arena/internal/storage"
)

func SFTPHandler(uploadDir string) func(ssh.Session) {
	return func(s ssh.Session) {
		userDir := filepath.Join(uploadDir, s.User())
		
		if err := os.MkdirAll(userDir, 0755); err != nil {
			log.Printf("Failed to create user directory: %v", err)
			return
		}
		
		handler := &sftpFileHandler{
			baseDir:  userDir,
			username: s.User(),
		}
		
		server := sftp.NewRequestServer(s, sftp.Handlers{
			FileGet:  handler,
			FilePut:  handler,
			FileCmd:  handler,
			FileList: handler,
		})
		
		if err := server.Serve(); err == io.EOF {
			server.Close()
		} else if err != nil {
			log.Printf("sftp server error: %v", err)
			wish.Fatalln(s, err)
		}
	}
}

type sftpFileHandler struct {
	baseDir  string
	username string
}

// Fileread for downloads (disabled)
func (h *sftpFileHandler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	return nil, fmt.Errorf("downloads not supported")
}

// Filewrite for uploads
func (h *sftpFileHandler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	filename := filepath.Base(r.Filepath)
	
	// Validate filename
	if !strings.HasPrefix(filename, "memory_functions_") || !strings.HasSuffix(filename, ".cpp") {
		log.Printf("Invalid filename from %s: %s", h.username, filename)
		return nil, fmt.Errorf("only memory_functions_*.cpp files are accepted")
	}
	
	dstPath := filepath.Join(h.baseDir, filename)
	log.Printf("SFTP: Creating file %s for user %s", dstPath, h.username)
	
	// Remove old file if it exists to ensure clean overwrite
	if _, err := os.Stat(dstPath); err == nil {
		log.Printf("SFTP: Removing old file: %s", dstPath)
		os.Remove(dstPath)
	}
	
	flags := r.Pflags()
	var osFlags int
	if flags.Creat {
		osFlags |= os.O_CREATE
	}
	if flags.Trunc {
		osFlags |= os.O_TRUNC
	}
	if flags.Write {
		osFlags |= os.O_WRONLY
	}
	
	file, err := os.OpenFile(dstPath, osFlags, 0644)
	if err != nil {
		log.Printf("Failed to create file: %v", err)
		return nil, err
	}
	
	return &fileWriterAt{
		file:     file,
		filename: filename,
		username: h.username,
	}, nil
}

// Filecmd handles file operations
func (h *sftpFileHandler) Filecmd(r *sftp.Request) error {
	switch r.Method {
	case "Setstat", "Rename", "Remove", "Mkdir", "Rmdir":
		// Allow these operations within user directory
		return nil
	default:
		return sftp.ErrSSHFxOpUnsupported
	}
}

// Filelist for directory listings
func (h *sftpFileHandler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	switch r.Method {
	case "List":
		entries, err := os.ReadDir(h.baseDir)
		if err != nil {
			return nil, err
		}
		infos := make([]fs.FileInfo, 0, len(entries))
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}
			infos = append(infos, info)
		}
		return listerAt(infos), nil
	case "Stat":
		info, err := os.Stat(filepath.Join(h.baseDir, r.Filepath))
		if err != nil {
			return nil, err
		}
		return listerAt{info}, nil
	default:
		return nil, sftp.ErrSSHFxOpUnsupported
	}
}

type listerAt []fs.FileInfo

func (l listerAt) ListAt(ls []fs.FileInfo, offset int64) (int, error) {
	if offset >= int64(len(l)) {
		return 0, io.EOF
	}
	n := copy(ls, l[offset:])
	if n < len(ls) {
		return n, io.EOF
	}
	return n, nil
}

type fileWriterAt struct {
	file     *os.File
	filename string
	username string
}

func (f *fileWriterAt) WriteAt(p []byte, off int64) (int, error) {
	return f.file.WriteAt(p, off)
}

func (f *fileWriterAt) Close() error {
	err := f.file.Close()
	if err == nil {
		log.Printf("SFTP: Uploaded %s from %s", f.filename, f.username)
		
		// Add submission and trigger testing
		submissionID, err := storage.AddSubmission(f.username, f.filename)
		if err != nil {
			log.Printf("Failed to add submission: %v", err)
		} else {
			log.Printf("Queued submission %d for testing", submissionID)
			// The worker will pick it up automatically
		}
	}
	return err
}

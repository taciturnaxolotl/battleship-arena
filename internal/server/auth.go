package server

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	gossh "golang.org/x/crypto/ssh"
	
	"battleship-arena/internal/storage"
)

var (
	adminPasscode string
	externalURL   string
)

func GetServerURL() string {
	// Strip protocol (http://, https://) from URL for SSH commands
	url := externalURL
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	return url
}

func SetConfig(passcode, url string) {
	adminPasscode = passcode
	externalURL = url
	log.Printf("✓ Config loaded: url=%s\n", url)
}

func PublicKeyAuthHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	publicKeyStr := strings.TrimSpace(string(gossh.MarshalAuthorizedKey(key)))
	
	log.Printf("Auth attempt: user=%s, key_fingerprint=%s", ctx.User(), gossh.FingerprintSHA256(key))
	
	// Try to find user by public key
	user, err := storage.GetUserByPublicKey(publicKeyStr)
	if err != nil {
		log.Printf("Error looking up user by public key: %v", err)
		return false
	}
	
	if user != nil {
		// Existing user - verify username matches
		log.Printf("Found existing user: %s (trying to login as: %s)", user.Username, ctx.User())
		if user.Username == ctx.User() {
			ctx.SetValue("user_id", user.ID)
			ctx.SetValue("needs_onboarding", false)
			storage.UpdateUserLastLogin(user.Username)
			log.Printf("✓ Authenticated %s", user.Username)
			return true
		}
		// Public key registered to different username
		log.Printf("❌ Public key registered to %s, but trying to auth as %s", user.Username, ctx.User())
		return false
	}
	
	log.Printf("New user detected: %s", ctx.User())
	
	// New user - check if username is taken
	existingUser, err := storage.GetUserByUsername(ctx.User())
	if err != nil {
		log.Printf("Error looking up username: %v", err)
		return false
	}
	
	if existingUser != nil {
		// Username taken by someone else
		log.Printf("❌ Username %s already taken", ctx.User())
		return false
	}
	
	// New user with available username - allow and mark for onboarding
	log.Printf("✓ New user %s allowed for onboarding", ctx.User())
	ctx.SetValue("public_key", publicKeyStr)
	ctx.SetValue("needs_onboarding", true)
	return true
}

func PasswordAuthHandler(ctx ssh.Context, password string) bool {
	// Check for admin passcode override
	if password == adminPasscode {
		log.Printf("🔑 Admin passcode used for user: %s", ctx.User())
		
		// Check if user exists
		user, err := storage.GetUserByUsername(ctx.User())
		if err != nil {
			log.Printf("Error looking up username: %v", err)
			return false
		}
		
		if user != nil {
			// Existing user - allow login
			ctx.SetValue("user_id", user.ID)
			ctx.SetValue("needs_onboarding", false)
			ctx.SetValue("admin_override", true)
			log.Printf("✓ Admin authenticated as %s", user.Username)
			return true
		}
		
		// New user - create with dummy key
		log.Printf("✓ Admin creating new user: %s", ctx.User())
		dummyKey := fmt.Sprintf("admin-override-%s", ctx.User())
		newUser, err := storage.CreateUser(ctx.User(), ctx.User(), "Admin created user", "", dummyKey)
		if err != nil {
			log.Printf("Error creating user: %v", err)
			return false
		}
		
		ctx.SetValue("user_id", newUser.ID)
		ctx.SetValue("needs_onboarding", false)
		ctx.SetValue("admin_override", true)
		log.Printf("✓ Admin created and authenticated as %s", ctx.User())
		return true
	}
	
	// Regular password auth disabled
	return false
}

func SessionHandler(s ssh.Session) {
	needsOnboarding := false
	if val := s.Context().Value("needs_onboarding"); val != nil {
		needsOnboarding = val.(bool)
	}
	
	if needsOnboarding {
		// Run onboarding flow
		if err := runOnboarding(s); err != nil {
			wish.Errorln(s, fmt.Sprintf("Onboarding failed: %v", err))
			return
		}
	}
	
	// Normal session continues
	wish.Println(s, "Welcome to Battleship Arena!")
}

func runOnboarding(s ssh.Session) error {
	username := s.User()
	publicKeyStr := ""
	if val := s.Context().Value("public_key"); val != nil {
		publicKeyStr = val.(string)
	}
	
	if publicKeyStr == "" {
		return errors.New("no public key found")
	}
	
	wish.Println(s, "\n🚢 Welcome to Battleship Arena!")
	wish.Println(s, fmt.Sprintf("Setting up account for: %s\n", username))
	
	// Get name
	wish.Print(s, "What's your full name? (required): ")
	name, err := readLine(s)
	if err != nil {
		return err
	}
	if name == "" {
		return errors.New("name is required")
	}
	
	// Get bio
	wish.Print(s, "Bio (optional, press Enter to skip): ")
	bio, err := readLine(s)
	if err != nil {
		return err
	}
	
	// Get link
	wish.Print(s, "Link (optional, press Enter to skip): ")
	link, err := readLine(s)
	if err != nil {
		return err
	}
	
	// Create user
	_, err = storage.CreateUser(username, name, bio, link, publicKeyStr)
	if err != nil {
		return fmt.Errorf("failed to create user: %v", err)
	}
	
	wish.Println(s, "\n✅ Account created successfully!")
	wish.Println(s, "You can now upload your battleship AI and compete!\n")
	
	// Update context
	s.Context().SetValue("needs_onboarding", false)
	
	return nil
}

func readLine(s ssh.Session) (string, error) {
	var line []byte
	buf := make([]byte, 1)
	
	for {
		n, err := s.Read(buf)
		if err != nil {
			return "", err
		}
		if n == 0 {
			continue
		}
		
		b := buf[0]
		
		// Handle newline
		if b == '\n' || b == '\r' {
			return string(line), nil
		}
		
		// Handle backspace
		if b == 127 || b == 8 {
			if len(line) > 0 {
				line = line[:len(line)-1]
				s.Write([]byte("\b \b"))
			}
			continue
		}
		
		// Handle printable characters
		if b >= 32 && b < 127 {
			line = append(line, b)
			s.Write(buf[:1])
		}
	}
}

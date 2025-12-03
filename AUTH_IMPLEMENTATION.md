# SSH Public Key Authentication - Implementation Summary

## ✅ What Was Implemented

### 1. Database Schema
Added `users` table to track authenticated users:
- `username` - SSH username (unique)
- `name` - Full name (required during onboarding)
- `bio` - Optional description
- `link` - Optional website/social link
- `public_key` - SSH public key (unique, used for auth)
- `created_at` - Registration timestamp
- `last_login_at` - Last successful login

**Location:** `internal/storage/database.go` + new `internal/storage/users.go`

### 2. SSH Authentication Handler
Implements public key authentication flow:
- Checks if public key is registered
- If registered: verifies username matches and allows access
- If new: checks if username is available
- If username taken: rejects (prevents key reuse)
- If available: flags user for onboarding

**Location:** `internal/server/auth.go`

### 3. Onboarding Flow
Interactive terminal prompt for first-time users:
- Prompts for full name (required)
- Prompts for bio (optional, skip with Enter)
- Prompts for link (optional, skip with Enter)
- Creates user record with their public key
- Subsequent logins skip onboarding

**Location:** `internal/server/auth.go` + `internal/tui/onboarding.go`

### 4. User Profile Pages
Web interface to view user information:
- `/users` - List all registered users
- `/user/{username}` - Individual user profile showing:
  - Name, bio, and link
  - SSH public key fingerprint
  - Game statistics (rating, wins, losses)
  - Join date and last login

**Location:** `internal/server/users.go`

### 5. Leaderboard Integration
Updated leaderboard to link usernames to profiles:
- Clicking a username takes you to their profile
- Shows authentication info alongside game stats

**Location:** `internal/server/web.go` (updated player name links)

## 🔐 Security Features

1. **Public key only** - No password authentication accepted
2. **Username ownership** - One public key per username, cannot be changed
3. **Key uniqueness** - One public key cannot register multiple usernames  
4. **Automatic verification** - Every connection validates the key

## 📝 User Experience

### First Connection
```bash
ssh -p 2222 -i ~/.ssh/id_ed25519 alice@localhost
```

**Prompts:**
```
🚢 Welcome to Battleship Arena!
Setting up account for: alice

What's your full name? (required): Alice Johnson
Bio (optional, press Enter to skip): CS student and battleship enthusiast
Link (optional, press Enter to skip): https://github.com/alice

✅ Account created successfully!
You can now upload your battleship AI and compete!
```

### Subsequent Connections
```bash
ssh -p 2222 alice@localhost
# → Immediately shows TUI dashboard (no prompts)
```

### Uploading Files
```bash
scp -P 2222 memory_functions_alice.cpp alice@localhost:~/
# → Works with same key authentication
```

## 🌐 Web Interface

### User List (`/users`)
- Grid view of all registered users
- Shows name, username, bio
- Click to view full profile

### User Profile (`/user/alice`)
- Full name and username
- Bio and external link (if provided)
- SSH public key fingerprint (SHA256)
- Game statistics (if they've competed)
- Registration and last login timestamps

### Leaderboard (`/`)
- Usernames are now clickable links
- Lead to user profile pages
- Shows rating, wins, losses, etc.

## 📂 Files Modified/Created

### New Files
- `internal/storage/users.go` - User CRUD operations
- `internal/server/auth.go` - SSH authentication handlers
- `internal/server/users.go` - User profile web handlers
- `internal/tui/onboarding.go` - Onboarding TUI (Bubble Tea model)
- `SSH_AUTH.md` - User-facing documentation

### Modified Files
- `internal/storage/database.go` - Added users table to schema
- `cmd/battleship-arena/main.go` - Added auth handlers and user routes
- `internal/server/web.go` - Updated player name links to /user/

## 🚀 Testing

1. **Start server:**
   ```bash
   make run
   ```

2. **Connect with new user:**
   ```bash
   ssh -p 2222 newuser@localhost
   ```

3. **View users:**
   ```
   http://localhost:8081/users
   ```

4. **View profile:**
   ```
   http://localhost:8081/user/newuser
   ```

5. **Try duplicate username:**
   ```bash
   # With different SSH key, same username → should be rejected
   ```

## 💡 Design Decisions

1. **Onboarding in terminal** - Users are already in SSH, so keep it simple
2. **Public key as primary key** - Ensures one key = one account
3. **Optional bio/link** - Don't force users to provide info they don't want to share
4. **SHA256 fingerprint display** - More readable than full public key
5. **Separate /user/ route** - Distinguishes from game stats at /player/

## 🔄 Migration Path

Existing deployments will need to:
1. Run migration to add users table (happens automatically on next startup)
2. Existing SSH users will be prompted for onboarding on next login
3. No data loss - submission history remains intact

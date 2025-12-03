# SSH Public Key Authentication

The Battleship Arena now uses SSH public key authentication for secure, passwordless access.

## First-Time Setup

1. **Generate an SSH key** (if you don't have one):
   ```bash
   ssh-keygen -t ed25519 -f ~/.ssh/battleship_arena
   ```

2. **Connect for the first time**:
   ```bash
   ssh -p 2222 -i ~/.ssh/battleship_arena yourname@localhost
   ```

3. **Complete onboarding**:
   - Enter your full name (required)
   - Enter a bio (optional)
   - Enter a website/link (optional)

4. **Your public key is now registered!** Only you can access this username.

## Uploading Your AI

After registration, upload your battleship AI:

```bash
scp -P 2222 -i ~/.ssh/battleship_arena memory_functions_yourname.cpp yourname@localhost:~/
```

## User Profiles

- View your profile: `https://arena.example.com/user/yourname`
- View all users: `https://arena.example.com/users`
- Profiles display:
  - Name, bio, and link
  - SSH public key fingerprint
  - Game statistics (if you've competed)

## Security Features

- ✅ Public key authentication only (no passwords)
- ✅ Username ownership tied to SSH key
- ✅ Keys cannot be reused for different usernames
- ✅ Automatic key verification on every connection

## SSH Config

Add to `~/.ssh/config` for easy access:

```
Host battleship
    HostName localhost
    Port 2222
    User yourname
    IdentityFile ~/.ssh/battleship_arena
```

Then simply: `ssh battleship`

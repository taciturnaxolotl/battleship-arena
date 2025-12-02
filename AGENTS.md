# Development Notes

## Architecture

- **main.go** - SSH/HTTP server initialization with Wish and Bubble Tea
- **model.go** - Terminal UI (TUI) for SSH sessions
- **database.go** - SQLite storage for submissions and results
- **web.go** - HTTP leaderboard with HTML template
- **runner.go** - Compiles and tests C++ submissions against battleship library
- **scp.go** - SCP upload middleware for file submissions
- **worker.go** - Background processor (runs every 30s)

## File Upload

Students upload via SCP:
```bash
scp -P 2222 memory_functions_name.cpp username@host:~/
```

Files must match pattern `memory_functions_*.cpp`

## Testing Flow

1. Student uploads file via SCP → saved to `./submissions/username/`
2. Student SSH in and selects "Test Submission"
3. Worker picks up pending submission
4. Compiles with battleship library: `g++ battle_light.cpp battleship_light.cpp memory_functions_*.cpp`
5. Runs benchmark: `./battle --benchmark 100`
6. Parses results and updates database
7. Leaderboard shows updated rankings

## Configuration

Edit `runner.go` line 11:
```go
const battleshipRepoPath = "/path/to/cs1210-battleship"
```

## Building

```bash
make build    # Build binary
make run      # Build and run
make gen-key  # Generate SSH host key
```

## Deployment

See `Dockerfile`, `docker-compose.yml`, or `battleship-arena.service` for systemd.

Web runs on port 8080, SSH on port 2222.

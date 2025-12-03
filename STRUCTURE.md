# Battleship Arena - Code Structure

Refactored into a clean modular architecture with proper separation of concerns.

## Directory Structure

```
battleship-arena/
├── cmd/
│   └── battleship-arena/      # Main application entry point
│       └── main.go             # Server initialization and routing
├── internal/                   # Private application code
│   ├── runner/                 # Match execution and compilation
│   │   ├── runner.go           # AI compilation and match running
│   │   └── worker.go           # Background submission processor
│   ├── server/                 # HTTP/SSH server components
│   │   ├── scp.go              # SCP file upload handler
│   │   ├── sftp.go             # SFTP file upload handler
│   │   ├── sse.go              # Server-Sent Events for live updates
│   │   └── web.go              # HTTP handlers and HTML templates
│   ├── storage/                # Data persistence layer
│   │   ├── database.go         # SQLite schema and queries
│   │   └── tournament.go       # Tournament bracket management
│   └── tui/                    # Terminal User Interface
│       └── model.go            # Bubble Tea SSH interface
├── battleship-engine/          # C++ battleship game engine
├── static/                     # Static web assets
├── go.mod                      # Go module definition
└── Makefile                    # Build automation

```

## Module Responsibilities

### `cmd/battleship-arena`
- Application entry point
- Server initialization (SSH, HTTP, SSE)
- Dependency injection and configuration
- Graceful shutdown handling

### `internal/runner`
- **runner.go**: Compiles C++ submissions, generates match binaries, runs head-to-head games
- **worker.go**: Background worker that processes pending submissions in a queue

### `internal/server`
- **scp.go**: Validates and handles SCP file uploads from students
- **sftp.go**: SFTP subsystem for file uploads
- **sse.go**: Server-Sent Events for real-time leaderboard updates and progress tracking
- **web.go**: HTTP handlers for leaderboard, player pages, and API endpoints

### `internal/storage`
- **database.go**: SQLite schema, CRUD operations, Glicko-2 rating system implementation
- **tournament.go**: Bracket generation, seeding, match scheduling, winner advancement

### `internal/tui`
- **model.go**: Bubble Tea terminal interface shown when students SSH in

## Key Design Decisions

1. **Internal packages**: Use `internal/` to prevent external imports and keep APIs private
2. **Dependency injection**: Pass configuration (uploadDir, ports) through function parameters rather than globals
3. **Clean interfaces**: Each module exports only what's needed (capital letters for public functions)
4. **Separation of concerns**: Storage, presentation, business logic, and transport are cleanly separated
5. **No circular dependencies**: Dependencies flow downward (cmd → server/runner → storage)

## Building & Running

```bash
# Build binary
make build

# Run server
make run

# Generate SSH host key
make gen-key

# Clean artifacts
make clean
```

## Adding Features

- **New API endpoint**: Add handler to `internal/server/web.go`, register route in `cmd/battleship-arena/main.go`
- **New database table**: Update schema in `storage.InitDB()`, add query functions to `internal/storage/database.go`
- **New match logic**: Modify `internal/runner/runner.go`
- **New TUI screen**: Update model in `internal/tui/model.go`

## Testing

```bash
go test ./...
```

Currently no tests exist (all packages return `[no test files]`), but the modular structure makes it easy to add unit tests for each package.

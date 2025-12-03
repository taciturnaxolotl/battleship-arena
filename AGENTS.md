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

## Required Function Signatures

**IMPORTANT**: Your submission must implement exactly these three functions with these exact signatures:

```cpp
void initMemoryYOURNAME(ComputerMemory &memory);
string smartMoveYOURNAME(const ComputerMemory &memory);
void updateMemoryYOURNAME(int row, int col, int result, ComputerMemory &memory);
```

Replace `YOURNAME` with your chosen suffix (must match your filename `memory_functions_YOURNAME.cpp`).

**Example** for `memory_functions_alice.cpp`:
```cpp
#include "memory_functions_alice.h"
#include "battleship.h"
#include "kasbs.h"
#include "memory.h"
#include <string>

using namespace std;

inline string formatMove(int row, int col) {
    char letter = static_cast<char>('A' + row);
    return string(1, letter) + to_string(col + 1);
}

void initMemoryAlice(ComputerMemory &memory) {
    // Initialize your memory structure
    for (int i = 0; i < BOARDSIZE; i++) {
        for (int j = 0; j < BOARDSIZE; j++) {
            memory.grid[i][j] = '?';
        }
    }
}

void updateMemoryAlice(int row, int col, int result, ComputerMemory &memory) {
    // Update memory based on shot result
    // result constants: HIT, MISS, SUNK (from kasbs.h)
    if (result == HIT || result == SUNK) {
        memory.grid[row][col] = 'h';
    } else {
        memory.grid[row][col] = 'm';
    }
}

string smartMoveAlice(const ComputerMemory &memory) {
    // Return your next move as a string (e.g., "A1", "B5", "J10")
    int row = 0;  // your logic here
    int col = 0;  // your logic here
    return formatMove(row, col);
}
```

**Key Points**:
- Function names must match your filename suffix exactly (case-sensitive)
- Must return `string` from `smartMove`, not integer array
- Must use `ComputerMemory &` parameter, not custom structs
- Use `formatMove(row, col)` helper to convert coordinates to string format
- Result constants available: `HIT`, `MISS`, `SUNK` from `kasbs.h`
- Grid size constant: `BOARDSIZE` from `kasbs.h`

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

## Performance Stages

Submissions are categorized into stages based on average moves per game:

- **Expert** (<85 moves): Significantly better than random shooting
- **Advanced** (85-95 moves): Better than random shooting  
- **Intermediate** (95-99 moves): Around random shooting performance
- **Beginner** (≥99 moves): Worse than random shooting

*Benchmark: Pure random shooting averages 95.5 moves over 1000 games (range: 64-100)*

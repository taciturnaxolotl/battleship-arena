# Battleship Arena - Tournament System

## Overview
Tournament-style battleship AI competition with automatic header generation and local compilation.

## Architecture

### Battleship Engine
- Located in `./battleship-engine/`
- Contains the lightweight battleship game engine
- No external dependencies on the school repo
- Auto-generates header files for submissions

### Submission Flow
1. User uploads `memory_functions_<name>.cpp` via SCP/SFTP
2. System auto-generates `memory_functions_<name>.h` header
3. Compiles submission with the battleship engine
4. If successful, runs tournament matches against all active submissions
5. Updates leaderboard with results

### Tournament Matching
- Each match compiles both AIs into a single binary
- Runs 10 games per match
- Winner determined by total wins
- All results stored in database

## Test Submissions

Three AI implementations for testing:

1. **random** - Pure random shooter (baseline)
2. **hunter** - Checkerboard hunt + adjacent targeting  
3. **klukas** - Advanced probability-based AI

## Testing

```bash
# Upload test submissions
./scripts/test-upload.sh

# This uploads:
# - alice with random AI
# - bob with hunter AI
# - charlie with klukas AI
```

## Requirements

Submissions must:
- Be named `memory_functions_<name>.cpp`
- Implement three functions:
  - `void initMemory<Name>(ComputerMemory &memory)`
  - `std::string smartMove<Name>(const ComputerMemory &memory)`
  - `void updateMemory<Name>(int row, int col, int result, ComputerMemory &memory)`
- Use only standard includes and provided headers (battleship.h, kasbs.h, memory.h)

Headers are auto-generated - users only need to upload the `.cpp` file!

## Usage

Start server:
```bash
./battleship-arena
```

View results:
- SSH TUI: `ssh -p 2222 username@0.0.0.0`
- Web: http://0.0.0.0:8080

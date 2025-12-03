# Scripts

Helper scripts for testing and development.

## Batch Upload

### `batch-upload.sh`
Uploads all test AIs using admin passcode authentication.

```bash
./scripts/batch-upload.sh
```

**What it does:**
- Uses admin passcode to authenticate as different users
- Auto-creates users if they don't exist
- Uploads each test AI file
- Queues all submissions for testing

**Admin passcode:**
- Default: `battleship-admin-override`
- Override via: `BATTLESHIP_ADMIN_PASSCODE` env var

## Test Submissions

The `test-submissions/` directory contains sample AI implementations for testing:

- `memory_functions_random.cpp` - Random shooting
- `memory_functions_hunter.cpp` - Hunt mode after first hit
- `memory_functions_diagonal.cpp` - Diagonal scanning pattern
- `memory_functions_parity.cpp` - Checkerboard pattern
- `memory_functions_probability.cpp` - Probability density
- `memory_functions_cluster.cpp` - Clustered targeting
- `memory_functions_edge.cpp` - Edge-first strategy
- `memory_functions_spiral.cpp` - Spiral scanning
- `memory_functions_snake.cpp` - Snake pattern
- `memory_functions_klukas.cpp` - Advanced algorithm

## Benchmark Script

### `benchmark_random`
Runs pure random shooting baseline (compiled C++ binary).

```bash
./scripts/benchmark_random 1000
```

Outputs average moves over N games for comparison.

## Quick Start

**Upload test AIs:**
```bash
# Run batch upload with admin passcode
./scripts/batch-upload.sh
```

**Manual upload (with SSH key):**
```bash
# Upload as yourself
scp -P 2222 memory_functions_yourname.cpp username@localhost:~/
```

**View results:**
- Web UI: http://localhost:8081
- All users: http://localhost:8081/users

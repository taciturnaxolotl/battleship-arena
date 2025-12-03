#!/bin/bash

# Batch upload script - uploads all test submissions using admin passcode
# This bypasses normal SSH key authentication for testing/setup

HOST="localhost"
PORT="2222"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Admin passcode (set via environment variable or use default)
ADMIN_PASSCODE="${BATTLESHIP_ADMIN_PASSCODE:-battleship-admin-override}"

echo "🚢 Battleship Arena - Batch Upload Script (Admin Mode)"
echo "======================================================="
echo ""
echo "This script uses admin passcode authentication to:"
echo "  1. Auto-create users if they don't exist"
echo "  2. Upload AI files for each user"
echo "  3. Queue submissions for testing"
echo ""
echo "⚠️  Admin passcode: ${ADMIN_PASSCODE:0:10}..."
echo ""
echo "Press Enter to continue or Ctrl+C to cancel..."
read

# Define all submissions: username, filename
declare -a SUBMISSIONS=(
    "alice:memory_functions_random.cpp"
    "bob:memory_functions_hunter.cpp"
    "charlie:memory_functions_klukas.cpp"
    "dave:memory_functions_diagonal.cpp"
    "eve:memory_functions_edge.cpp"
    "frank:memory_functions_spiral.cpp"
    "grace:memory_functions_parity.cpp"
    "henry:memory_functions_probability.cpp"
    "iris:memory_functions_cluster.cpp"
    "jack:memory_functions_snake.cpp"
)

# Upload each submission using admin passcode
success_count=0
fail_count=0

for submission in "${SUBMISSIONS[@]}"; do
    IFS=':' read -r username filename <<< "$submission"
    
    echo "📤 Uploading for user: $username"
    echo "   File: test-submissions/$filename"
    
    if [ ! -f "$SCRIPT_DIR/test-submissions/$filename" ]; then
        echo "❌ Error: File not found"
        ((fail_count++))
        echo ""
        continue
    fi
    
    # Use sshpass to provide password authentication
    # If sshpass not available, use expect or manual password entry
    if command -v sshpass &> /dev/null; then
        sshpass -p "$ADMIN_PASSCODE" scp -P $PORT "$SCRIPT_DIR/test-submissions/$filename" "$username@$HOST:~/$filename" 2>&1 | grep -q "100%"
        result=$?
    else
        echo "   Using manual password authentication (enter passcode when prompted)"
        echo "   Password: $ADMIN_PASSCODE"
        scp -P $PORT "$SCRIPT_DIR/test-submissions/$filename" "$username@$HOST:~/$filename" 2>&1 | grep -q "100%"
        result=$?
    fi
    
    if [ $result -eq 0 ]; then
        echo "✅ Upload successful for $username"
        ((success_count++))
    else
        echo "❌ Upload failed for $username"
        ((fail_count++))
    fi
    echo ""
    
    # Small delay to avoid overwhelming the server
    sleep 0.5
done

echo "======================================================="
echo "✨ Batch upload complete!"
echo ""
echo "Results:"
echo "  ✅ Successful: $success_count"
echo "  ❌ Failed: $fail_count"
echo ""
echo "Next steps:"
echo "  - View web leaderboard: http://$HOST:8081"
echo "  - View all users: http://$HOST:8081/users"
echo "  - Check compilation logs in server output"
echo ""
echo "Note: Compilation and testing will happen automatically in the background."

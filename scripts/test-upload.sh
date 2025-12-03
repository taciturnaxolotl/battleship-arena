#!/bin/bash

# Test script to upload all AI submissions

HOST="0.0.0.0"
PORT="2222"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🚢 Battleship Arena - Test Submission Script"
echo "=============================================="
echo ""

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

# Upload each submission
for submission in "${SUBMISSIONS[@]}"; do
    IFS=':' read -r username filename <<< "$submission"
    
    echo "📤 Uploading for user: $username"
    echo "   File: test-submissions/$filename"
    
    if [ -f "$SCRIPT_DIR/test-submissions/$filename" ]; then
        # Capture SCP output to check for 100% completion
        scp_output=$(scp -P $PORT "$SCRIPT_DIR/test-submissions/$filename" "$username@$HOST:~/$filename" 2>&1)
        echo "$scp_output" | grep -q "100%"
        if [ $? -eq 0 ]; then
            echo "✅ Upload successful for $username"
        else
            echo "❌ Upload failed for $username"
            echo "$scp_output"
        fi
    else
        echo "❌ Error: File not found"
    fi
    echo ""
done

echo "=============================================="
echo "✨ All submissions uploaded!"
echo ""
echo "You can now:"
echo "  - SSH to view the TUI: ssh -p $PORT alice@$HOST"
echo "  - Check the web leaderboard: http://$HOST:8080"

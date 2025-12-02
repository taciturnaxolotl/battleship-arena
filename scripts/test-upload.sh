#!/bin/bash

# Test script to upload three different AI submissions

HOST="0.0.0.0"
PORT="2222"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "🚢 Battleship Arena - Test Submission Script"
echo "=============================================="
echo ""

# Copy klukas submission to test-submissions if it doesn't exist
if [ ! -f "$SCRIPT_DIR/test-submissions/memory_functions_klukas.cpp" ]; then
    echo "Copying klukas submission..."
    mkdir -p "$SCRIPT_DIR/test-submissions"
    cp /Users/kierank/code/school/cs1210-battleship/src/memory_functions_klukas.cpp "$SCRIPT_DIR/test-submissions/"
fi

# Upload for alice (random AI)
echo "📤 Uploading for user: alice"
echo "   File: test-submissions/memory_functions_random.cpp"
if [ -f "$SCRIPT_DIR/test-submissions/memory_functions_random.cpp" ]; then
    scp -P $PORT "$SCRIPT_DIR/test-submissions/memory_functions_random.cpp" "alice@$HOST:~/memory_functions_random.cpp"
    if [ $? -eq 0 ]; then
        echo "✅ Upload successful for alice"
    else
        echo "❌ Upload failed for alice"
    fi
else
    echo "❌ Error: File not found"
fi
echo ""

# Upload for bob (hunter AI)
echo "📤 Uploading for user: bob"
echo "   File: test-submissions/memory_functions_hunter.cpp"
if [ -f "$SCRIPT_DIR/test-submissions/memory_functions_hunter.cpp" ]; then
    scp -P $PORT "$SCRIPT_DIR/test-submissions/memory_functions_hunter.cpp" "bob@$HOST:~/memory_functions_hunter.cpp"
    if [ $? -eq 0 ]; then
        echo "✅ Upload successful for bob"
    else
        echo "❌ Upload failed for bob"
    fi
else
    echo "❌ Error: File not found"
fi
echo ""

# Upload for charlie (klukas AI)
echo "📤 Uploading for user: charlie"
echo "   File: test-submissions/memory_functions_klukas.cpp"
if [ -f "$SCRIPT_DIR/test-submissions/memory_functions_klukas.cpp" ]; then
    scp -P $PORT "$SCRIPT_DIR/test-submissions/memory_functions_klukas.cpp" "charlie@$HOST:~/memory_functions_klukas.cpp"
    if [ $? -eq 0 ]; then
        echo "✅ Upload successful for charlie"
    else
        echo "❌ Upload failed for charlie"
    fi
else
    echo "❌ Error: File not found"
fi
echo ""

echo "=============================================="
echo "✨ All submissions uploaded!"
echo ""
echo "You can now:"
echo "  - SSH to view the TUI: ssh -p $PORT alice@$HOST"
echo "  - Check the web leaderboard: http://$HOST:8080"

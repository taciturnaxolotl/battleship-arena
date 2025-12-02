#!/bin/bash
# Example test script for submitting and testing an AI

set -e

USER="testuser"
HOST="localhost"
PORT="2222"
FILE="memory_functions_test.cpp"

echo "🚢 Battleship Arena Test Script"
echo "================================"

# Check if submission file exists
if [ ! -f "$1" ]; then
    echo "Usage: $0 <memory_functions_*.cpp>"
    exit 1
fi

FILE=$(basename "$1")

echo "📤 Uploading $FILE..."
scp -P $PORT "$1" ${USER}@${HOST}:~/

echo "✅ Upload complete!"
echo ""
echo "Next steps:"
echo "1. SSH into the server: ssh -p $PORT ${USER}@${HOST}"
echo "2. Navigate to 'Test Submission' in the menu"
echo "3. View results on the leaderboard: http://localhost:8080"

#include <iostream>
#include <vector>
#include <algorithm>
#include <numeric>
#include <cstdlib>
#include <ctime>

using namespace std;

const int BOARDSIZE = 10;

// Simple coordinate parsing
pair<int, int> parseMove(const string& move) {
    int row = move[0] - 'A';
    int col = stoi(move.substr(1)) - 1;
    return {row, col};
}

int runSingleGame() {
    char board[BOARDSIZE][BOARDSIZE];
    
    // Initialize board
    for (int i = 0; i < BOARDSIZE; i++) {
        for (int j = 0; j < BOARDSIZE; j++) {
            board[i][j] = '.';
        }
    }
    
    // Place ships randomly
    int shipSizes[] = {5, 4, 3, 3, 2};
    for (int s = 0; s < 5; s++) {
        bool placed = false;
        while (!placed) {
            int row = rand() % BOARDSIZE;
            int col = rand() % BOARDSIZE;
            int orient = rand() % 2;
            
            bool canPlace = true;
            for (int i = 0; i < shipSizes[s]; i++) {
                int r = row + (orient == 1 ? i : 0);
                int c = col + (orient == 0 ? i : 0);
                if (r >= BOARDSIZE || c >= BOARDSIZE || board[r][c] == 'S') {
                    canPlace = false;
                    break;
                }
            }
            
            if (canPlace) {
                for (int i = 0; i < shipSizes[s]; i++) {
                    int r = row + (orient == 1 ? i : 0);
                    int c = col + (orient == 0 ? i : 0);
                    board[r][c] = 'S';
                }
                placed = true;
            }
        }
    }
    
    // Random shooting
    vector<pair<int,int>> allCells;
    for (int i = 0; i < BOARDSIZE; i++) {
        for (int j = 0; j < BOARDSIZE; j++) {
            allCells.push_back({i, j});
        }
    }
    
    // Shuffle cells
    for (int i = allCells.size() - 1; i > 0; i--) {
        int j = rand() % (i + 1);
        swap(allCells[i], allCells[j]);
    }
    
    // Shoot until all ships found
    int moves = 0;
    int shipsRemaining = 17; // 5+4+3+3+2
    
    for (size_t i = 0; i < allCells.size(); i++) {
        int row = allCells[i].first;
        int col = allCells[i].second;
        moves++;
        if (board[row][col] == 'S') {
            shipsRemaining--;
            if (shipsRemaining == 0) break;
        }
    }
    
    return moves;
}

int main() {
    const int numGames = 1000;
    vector<int> moveCounts;
    
    srand(time(NULL));
    
    cout << "Running " << numGames << " games with random AI..." << endl;
    
    for (int i = 0; i < numGames; i++) {
        int moves = runSingleGame();
        moveCounts.push_back(moves);
        if ((i + 1) % 100 == 0) {
            cout << "Completed " << (i + 1) << " games..." << endl;
        }
    }
    
    // Calculate statistics
    sort(moveCounts.begin(), moveCounts.end());
    int minMoves = moveCounts.front();
    int maxMoves = moveCounts.back();
    double avg = accumulate(moveCounts.begin(), moveCounts.end(), 0.0) / moveCounts.size();
    int median = moveCounts[moveCounts.size() / 2];
    int p25 = moveCounts[moveCounts.size() / 4];
    int p75 = moveCounts[3 * moveCounts.size() / 4];
    
    cout << "\n=== Random AI Statistics (1000 games) ===" << endl;
    cout << "Min moves: " << minMoves << endl;
    cout << "25th percentile: " << p25 << endl;
    cout << "Median moves: " << median << endl;
    cout << "Average moves: " << avg << endl;
    cout << "75th percentile: " << p75 << endl;
    cout << "Max moves: " << maxMoves << endl;
    
    cout << "\n=== Suggested Stage Thresholds ===" << endl;
    cout << "Stage 1 (Beginner): >" << p75 << " avg moves (worse than random)" << endl;
    cout << "Stage 2 (Intermediate): " << static_cast<int>(avg) << "-" << p75 << " avg moves (around random average)" << endl;
    cout << "Stage 3 (Advanced): " << p25 << "-" << static_cast<int>(avg) << " avg moves (better than random)" << endl;
    cout << "Stage 4 (Expert): <" << p25 << " avg moves (much better than random)" << endl;
    
    return 0;
}

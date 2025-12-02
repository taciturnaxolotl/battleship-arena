#include "battleship_light.h"
#include <sstream>
#include <cctype>
#include <vector>
#include <mutex>

// Global flag for debug output
static bool g_debugEnabled = false;
static bool g_guardTripped = false;
static vector<string> g_debugLog;
static mutex g_debugLogMutex;
static const size_t MAX_DEBUG_LOG_SIZE = 1000;

void setDebugMode(bool enabled) {
    g_debugEnabled = enabled;
}

bool getGuardTripped() {
    return g_guardTripped;
}

void resetGuardTripped() {
    g_guardTripped = false;
    lock_guard<mutex> lock(g_debugLogMutex);
    g_debugLog.clear();
}

vector<string> getDebugLog() {
    lock_guard<mutex> lock(g_debugLogMutex);
    return g_debugLog;
}

void welcome(bool debug) {
    clearTheScreen();
    cout << "========================================" << endl;
    cout << "      BATTLESHIP - Lightweight" << endl;
    cout << "========================================" << endl;
    if (debug) {
        cout << "Debug mode enabled" << endl;
    }
    cout << endl;
}

void clearTheScreen() {
    // Simple cross-platform clear
    cout << "\033[2J\033[1;1H";
}

void pauseForEnter() {
    cout << "Press Enter to continue...";
    cin.ignore();
    cin.get();
}

void writeMessage(int x, int y, string message) {
    cout << message << endl;
}

void writeResult(int x, int y, int result, int playerType) {
    string player = (playerType == HUMAN) ? "Player" : "Computer";
    
    if (isASunk(result)) {
        int shipNum = isShip(result);
        char shipName;
        switch(shipNum) {
            case AC: shipName = 'A'; break;
            case BS: shipName = 'B'; break;
            case CR: shipName = 'C'; break;
            case SB: shipName = 'S'; break;
            case DS: shipName = 'D'; break;
            default: shipName = '?'; break;
        }
        cout << player << " SUNK a ship (" << shipName << ")!" << endl;
    } else if (isAHit(result)) {
        cout << player << " HIT!" << endl;
    } else {
        cout << player << " MISS" << endl;
    }
}

void displayBoard(int x, int y, int playerType, const Board &gameBoard) {
    cout << "   ";
    for (int col = 0; col < BOARDSIZE; col++) {
        cout << (col + 1) << " ";
    }
    cout << endl;
    
    for (int row = 0; row < BOARDSIZE; row++) {
        cout << char('A' + row) << "  ";
        for (int col = 0; col < BOARDSIZE; col++) {
            char cell = gameBoard.grid[row][col];
            
            // Hide ships if showing to computer player
            if (playerType == COMPUTER) {
                if (cell != HIT_MARKER && cell != MISS_MARKER && cell != SUNK_MARKER) {
                    cell = EMPTY_MARKER;
                }
            }
            
            cout << cell << " ";
        }
        cout << endl;
    }
    cout << endl;
}

bool placeShip(Board &gameBoard, int shipNum, int row, int col, int orient) {
    Ship &ship = gameBoard.s[shipNum];
    
    // Check bounds
    if (orient == HORZ) {
        if (col + ship.size > BOARDSIZE) return false;
    } else {
        if (row + ship.size > BOARDSIZE) return false;
    }
    
    // Check for collisions
    for (int i = 0; i < ship.size; i++) {
        int r = (orient == VERT) ? row + i : row;
        int c = (orient == HORZ) ? col + i : col;
        if (gameBoard.grid[r][c] != EMPTY_MARKER) {
            return false;
        }
    }
    
    // Place the ship
    ship.pos.startRow = row;
    ship.pos.startCol = col;
    ship.pos.orient = orient;
    
    for (int i = 0; i < ship.size; i++) {
        int r = (orient == VERT) ? row + i : row;
        int c = (orient == HORZ) ? col + i : col;
        gameBoard.grid[r][c] = ship.marker;
    }
    
    return true;
}

void initializeBoard(Board &gameBoard, bool file) {
    // Initialize grid
    for (int i = 0; i < BOARDSIZE; i++) {
        for (int j = 0; j < BOARDSIZE; j++) {
            gameBoard.grid[i][j] = EMPTY_MARKER;
        }
    }
    
    // Initialize ships
    gameBoard.s[AC].size = AC_SIZE;
    gameBoard.s[AC].hitsToSink = AC_SIZE;
    gameBoard.s[AC].marker = AC_MARKER;
    
    gameBoard.s[BS].size = BS_SIZE;
    gameBoard.s[BS].hitsToSink = BS_SIZE;
    gameBoard.s[BS].marker = BS_MARKER;
    
    gameBoard.s[CR].size = CR_SIZE;
    gameBoard.s[CR].hitsToSink = CR_SIZE;
    gameBoard.s[CR].marker = CR_MARKER;
    
    gameBoard.s[SB].size = SB_SIZE;
    gameBoard.s[SB].hitsToSink = SB_SIZE;
    gameBoard.s[SB].marker = SB_MARKER;
    
    gameBoard.s[DS].size = DS_SIZE;
    gameBoard.s[DS].hitsToSink = DS_SIZE;
    gameBoard.s[DS].marker = DS_MARKER;
    
    // Place ships randomly
    for (int shipNum = AC; shipNum <= DS; shipNum++) {
        bool placed = false;
        while (!placed) {
            int row = rand() % BOARDSIZE;
            int col = rand() % BOARDSIZE;
            int orient = rand() % 2;
            placed = placeShip(gameBoard, shipNum, row, col, orient);
        }
    }
}

int playMove(int row, int col, Board &gameBoard) {
    char cell = gameBoard.grid[row][col];
    
    // Already hit
    if (cell == HIT_MARKER || cell == MISS_MARKER || cell == SUNK_MARKER) {
        return MISS;
    }
    
    // Miss
    if (cell == EMPTY_MARKER) {
        gameBoard.grid[row][col] = MISS_MARKER;
        return MISS;
    }
    
    // Hit a ship
    int shipNum = 0;
    if (cell == AC_MARKER) shipNum = AC;
    else if (cell == BS_MARKER) shipNum = BS;
    else if (cell == CR_MARKER) shipNum = CR;
    else if (cell == SB_MARKER) shipNum = SB;
    else if (cell == DS_MARKER) shipNum = DS;
    
    if (shipNum == 0) {
        gameBoard.grid[row][col] = MISS_MARKER;
        return MISS;
    }
    
    Ship &ship = gameBoard.s[shipNum];
    ship.hitsToSink--;
    
    gameBoard.grid[row][col] = HIT_MARKER;
    
    if (ship.hitsToSink == 0) {
        // Mark all parts as sunk
        for (int i = 0; i < ship.size; i++) {
            int r = (ship.pos.orient == VERT) ? ship.pos.startRow + i : ship.pos.startRow;
            int c = (ship.pos.orient == HORZ) ? ship.pos.startCol + i : ship.pos.startCol;
            gameBoard.grid[r][c] = SUNK_MARKER;
        }
        return SUNK | shipNum;
    }
    
    return HIT | shipNum;
}

bool isAMiss(int playMoveResult) {
    return !(playMoveResult & HIT);
}

bool isAHit(int playMoveResult) {
    return (playMoveResult & HIT) != 0;
}

bool isASunk(int playMoveResult) {
    return (playMoveResult & SUNK) != 0;
}

int isShip(int playMoveResult) {
    return playMoveResult & SHIP;
}

string randomMove() {
    int row = rand() % BOARDSIZE;
    int col = rand() % BOARDSIZE;
    
    char letter = 'A' + row;
    return string(1, letter) + " " + to_string(col + 1);
}

int checkMove(string move, const Board &gameBoard, int &row, int &col) {
    // Trim whitespace
    move.erase(0, move.find_first_not_of(" \t\n\r"));
    move.erase(move.find_last_not_of(" \t\n\r") + 1);
    
    if (move.empty()) {
        return ILLEGAL_FORMAT;
    }
    
    // Parse format: "A 5" or "A5"
    char letter = toupper(move[0]);
    if (letter < 'A' || letter > 'J') {
        return ILLEGAL_FORMAT;
    }
    
    row = letter - 'A';
    
    // Extract number
    string numStr = move.substr(1);
    numStr.erase(0, numStr.find_first_not_of(" \t"));
    
    if (numStr.empty()) {
        return ILLEGAL_FORMAT;
    }
    
    try {
        col = stoi(numStr) - 1;
    } catch (...) {
        return ILLEGAL_FORMAT;
    }
    
    if (col < 0 || col >= BOARDSIZE) {
        return ILLEGAL_FORMAT;
    }
    
    // Check if already used
    char cell = gameBoard.grid[row][col];
    if (cell == HIT_MARKER || cell == MISS_MARKER || cell == SUNK_MARKER) {
        return REUSED_MOVE;
    }
    
    return VALID_MOVE;
}

void debug(string s, int x, int y) {
    // Only accumulate logs if debug mode is enabled or we need to track guards
    // This prevents memory bloat during benchmarks
    if (g_debugEnabled || s.find("*** GUARD TRIPPED ***") != string::npos) {
        lock_guard<mutex> lock(g_debugLogMutex);
        g_debugLog.push_back(s);
        
        // Limit log size to prevent unbounded growth
        if (g_debugLog.size() > MAX_DEBUG_LOG_SIZE) {
            g_debugLog.erase(g_debugLog.begin(), 
                           g_debugLog.begin() + (g_debugLog.size() - MAX_DEBUG_LOG_SIZE));
        }
    }
    
    if (g_debugEnabled) {
        cout << "[DEBUG] " << s << endl;
    }
}

string numToString(int x) {
    stringstream ss;
    ss << x;
    return ss.str();
}

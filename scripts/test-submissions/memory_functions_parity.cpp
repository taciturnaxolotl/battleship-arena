// Parity AI - uses checkerboard with hunt/target mode
// Strategy: Efficient coverage with smart follow-up on hits

#include "memory_functions_parity.h"
#include "battleship.h"
#include "kasbs.h"
#include "memory.h"
#include <string>
#include <vector>
#include <cstdlib>
#include <ctime>

using namespace std;

struct Cell {
    int row;
    int col;
};

static vector<Cell> targetStack;
static bool huntMode = true;
static int currentRow = 0;
static int currentCol = 0;

inline string formatMove(int row, int col) {
    char letter = static_cast<char>('A' + row);
    return string(1, letter) + to_string(col + 1);
}

void initMemoryParity(ComputerMemory &memory) {
    srand(time(NULL));
    
    for (int i = 0; i < BOARDSIZE; i++) {
        for (int j = 0; j < BOARDSIZE; j++) {
            memory.grid[i][j] = '?';
        }
    }
    
    targetStack.clear();
    huntMode = true;
    currentRow = 0;
    currentCol = 0;
}

void updateMemoryParity(int row, int col, int result, ComputerMemory &memory) {
    if (result == HIT || result == SUNK) {
        memory.grid[row][col] = 'h';
        huntMode = false;
        
        // Add adjacent cells to target stack
        int directions[4][2] = {{-1,0}, {1,0}, {0,-1}, {0,1}};
        for (int i = 0; i < 4; i++) {
            int newRow = row + directions[i][0];
            int newCol = col + directions[i][1];
            
            if (newRow >= 0 && newRow < BOARDSIZE && 
                newCol >= 0 && newCol < BOARDSIZE && 
                memory.grid[newRow][newCol] == '?') {
                
                Cell cell = {newRow, newCol};
                targetStack.push_back(cell);
            }
        }
    } else {
        memory.grid[row][col] = 'm';
    }
    
    if (targetStack.empty()) {
        huntMode = true;
    }
}

string smartMoveParity(const ComputerMemory &memory) {
    // Target mode - shoot from stack
    if (!huntMode && !targetStack.empty()) {
        Cell target = targetStack.back();
        targetStack.pop_back();
        return formatMove(target.row, target.col);
    }
    
    // Hunt mode - checkerboard pattern
    while (true) {
        if (currentRow >= BOARDSIZE) {
            // Try any unknown cell
            for (int i = 0; i < BOARDSIZE; i++) {
                for (int j = 0; j < BOARDSIZE; j++) {
                    if (memory.grid[i][j] == '?') {
                        return formatMove(i, j);
                    }
                }
            }
            return formatMove(rand() % BOARDSIZE, rand() % BOARDSIZE);
        }
        
        // Check parity and unknown
        if ((currentRow + currentCol) % 2 == 0 && memory.grid[currentRow][currentCol] == '?') {
            int row = currentRow;
            int col = currentCol;
            
            currentCol++;
            if (currentCol >= BOARDSIZE) {
                currentCol = 0;
                currentRow++;
            }
            return formatMove(row, col);
        }
        
        currentCol++;
        if (currentCol >= BOARDSIZE) {
            currentCol = 0;
            currentRow++;
        }
    }
}

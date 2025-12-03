// Snake AI - follows a snake/zigzag pattern across the board
// Strategy: Systematic coverage with no gaps, spacing based on smallest ship

#include "memory_functions_snake.h"
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
static int currentRow = 0;
static int currentCol = 0;
static bool movingRight = true;
static const int spacing = 2;

inline string formatMove(int row, int col) {
    char letter = static_cast<char>('A' + row);
    return string(1, letter) + to_string(col + 1);
}

void initMemorySnake(ComputerMemory &memory) {
    srand(time(NULL));
    
    for (int i = 0; i < BOARDSIZE; i++) {
        for (int j = 0; j < BOARDSIZE; j++) {
            memory.grid[i][j] = '?';
        }
    }
    
    targetStack.clear();
    currentRow = 0;
    currentCol = 0;
    movingRight = true;
}

void updateMemorySnake(int row, int col, int result, ComputerMemory &memory) {
    if (result == HIT || result == SUNK) {
        memory.grid[row][col] = 'h';
        
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
}

void getNextSnakePosition(int* row, int* col, const ComputerMemory &memory) {
    while (true) {
        // Off board, move to next row
        if (currentCol < 0 || currentCol >= BOARDSIZE) {
            currentRow += spacing;
            movingRight = !movingRight;
            
            if (movingRight) {
                currentCol = 0;
            } else {
                currentCol = BOARDSIZE - 1;
            }
            
            // Covered board, start filling gaps
            if (currentRow >= BOARDSIZE) {
                for (int i = 0; i < BOARDSIZE; i++) {
                    for (int j = 0; j < BOARDSIZE; j++) {
                        if (memory.grid[i][j] == '?') {
                            *row = i;
                            *col = j;
                            return;
                        }
                    }
                }
                *row = rand() % BOARDSIZE;
                *col = rand() % BOARDSIZE;
                return;
            }
            continue;
        }
        
        // Current position valid
        if (memory.grid[currentRow][currentCol] == '?') {
            *row = currentRow;
            *col = currentCol;
            
            if (movingRight) {
                currentCol += spacing;
            } else {
                currentCol -= spacing;
            }
            return;
        }
        
        // Already shot, move on
        if (movingRight) {
            currentCol += spacing;
        } else {
            currentCol -= spacing;
        }
    }
}

string smartMoveSnake(const ComputerMemory &memory) {
    // Shoot targets from hits first
    if (!targetStack.empty()) {
        Cell target = targetStack.back();
        targetStack.pop_back();
        return formatMove(target.row, target.col);
    }
    
    // Follow snake pattern
    int row, col;
    getNextSnakePosition(&row, &col, memory);
    return formatMove(row, col);
}

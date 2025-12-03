// Spiral AI - shoots in an inward spiral pattern
// Strategy: Methodical coverage starting from edges, spiraling toward center

#include "memory_functions_spiral.h"
#include "battleship.h"
#include "kasbs.h"
#include "memory.h"
#include <string>
#include <cstdlib>
#include <ctime>

using namespace std;

struct SpiralState {
    int nextRow;
    int nextCol;
    int spiralDepth;
    int currentDirection; // 0=RIGHT, 1=DOWN, 2=LEFT, 3=UP
    int stepsInCurrentDirection;
    int stepsToTake;
    bool initialized;
};

static SpiralState state;

inline string formatMove(int row, int col) {
    char letter = static_cast<char>('A' + row);
    return string(1, letter) + to_string(col + 1);
}

void initMemorySpiral(ComputerMemory &memory) {
    srand(time(NULL));
    
    for (int i = 0; i < BOARDSIZE; i++) {
        for (int j = 0; j < BOARDSIZE; j++) {
            memory.grid[i][j] = '?';
        }
    }
    
    state.nextRow = 0;
    state.nextCol = 0;
    state.currentDirection = 0; // RIGHT
    state.spiralDepth = 0;
    state.stepsInCurrentDirection = 0;
    state.stepsToTake = BOARDSIZE - 1;
    state.initialized = true;
}

void updateMemorySpiral(int row, int col, int result, ComputerMemory &memory) {
    if (result == HIT || result == SUNK) {
        memory.grid[row][col] = 'h';
    } else {
        memory.grid[row][col] = 'm';
    }
}

string smartMoveSpiral(const ComputerMemory &memory) {
    if (!state.initialized) {
        return formatMove(rand() % BOARDSIZE, rand() % BOARDSIZE);
    }
    
    int row = state.nextRow;
    int col = state.nextCol;
    
    // Advance to next position in spiral
    state.stepsInCurrentDirection++;
    
    // Move in current direction
    switch (state.currentDirection) {
        case 0: // RIGHT
            state.nextCol++;
            break;
        case 1: // DOWN
            state.nextRow++;
            break;
        case 2: // LEFT
            state.nextCol--;
            break;
        case 3: // UP
            state.nextRow--;
            break;
    }
    
    // Check if we need to turn
    if (state.stepsInCurrentDirection >= state.stepsToTake) {
        state.stepsInCurrentDirection = 0;
        state.currentDirection = (state.currentDirection + 1) % 4;
        
        // After turning twice, reduce steps
        if (state.currentDirection == 0 || state.currentDirection == 2) {
            state.stepsToTake--;
            state.spiralDepth++;
        }
    }
    
    // If spiraled to center, use fallback
    if (state.spiralDepth >= BOARDSIZE / 2) {
        for (int i = 0; i < BOARDSIZE; i++) {
            for (int j = 0; j < BOARDSIZE; j++) {
                if (memory.grid[i][j] == '?') {
                    return formatMove(i, j);
                }
            }
        }
        return formatMove(rand() % BOARDSIZE, rand() % BOARDSIZE);
    }
    
    return formatMove(row, col);
}

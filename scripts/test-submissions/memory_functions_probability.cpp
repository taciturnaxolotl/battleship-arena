// Probability AI - calculates ship placement likelihood for each cell
// Strategy: Smart probability-based targeting with density analysis

#include "memory_functions_probability.h"
#include "battleship.h"
#include "kasbs.h"
#include "memory.h"
#include <string>
#include <cstdlib>
#include <ctime>

using namespace std;

static int shipSizes[] = {5, 4, 3, 3, 2};
static int numShips = 5;

inline string formatMove(int row, int col) {
    char letter = static_cast<char>('A' + row);
    return string(1, letter) + to_string(col + 1);
}

void initMemoryProbability(ComputerMemory &memory) {
    srand(time(NULL));
    
    for (int i = 0; i < BOARDSIZE; i++) {
        for (int j = 0; j < BOARDSIZE; j++) {
            memory.grid[i][j] = '?';
        }
    }
}

void updateMemoryProbability(int row, int col, int result, ComputerMemory &memory) {
    if (result == HIT || result == SUNK) {
        memory.grid[row][col] = 'h';
    } else {
        memory.grid[row][col] = 'm';
    }
}

int calculateProbability(int row, int col, const ComputerMemory &memory) {
    if (memory.grid[row][col] != '?') {
        return 0;
    }
    
    int probability = 0;
    
    // For each ship size
    for (int ship = 0; ship < numShips; ship++) {
        int size = shipSizes[ship];
        
        // Check horizontal placements
        for (int startCol = col - size + 1; startCol <= col; startCol++) {
            if (startCol < 0 || startCol + size > BOARDSIZE) continue;
            
            bool valid = true;
            for (int c = startCol; c < startCol + size; c++) {
                if (memory.grid[row][c] == 'm' || memory.grid[row][c] == 's') {
                    valid = false;
                    break;
                }
            }
            if (valid) probability++;
        }
        
        // Check vertical placements
        for (int startRow = row - size + 1; startRow <= row; startRow++) {
            if (startRow < 0 || startRow + size > BOARDSIZE) continue;
            
            bool valid = true;
            for (int r = startRow; r < startRow + size; r++) {
                if (memory.grid[r][col] == 'm' || memory.grid[r][col] == 's') {
                    valid = false;
                    break;
                }
            }
            if (valid) probability++;
        }
    }
    
    // Bonus for cells adjacent to hits
    int directions[4][2] = {{-1,0}, {1,0}, {0,-1}, {0,1}};
    for (int i = 0; i < 4; i++) {
        int newRow = row + directions[i][0];
        int newCol = col + directions[i][1];
        
        if (newRow >= 0 && newRow < BOARDSIZE && 
            newCol >= 0 && newCol < BOARDSIZE && 
            memory.grid[newRow][newCol] == 'h') {
            probability += 50;
        }
    }
    
    return probability;
}

string smartMoveProbability(const ComputerMemory &memory) {
    int maxProb = -1;
    int bestRow = 0, bestCol = 0;
    
    // Calculate probability for each cell
    for (int i = 0; i < BOARDSIZE; i++) {
        for (int j = 0; j < BOARDSIZE; j++) {
            int prob = calculateProbability(i, j, memory);
            if (prob > maxProb) {
                maxProb = prob;
                bestRow = i;
                bestCol = j;
            }
        }
    }
    
    // Fallback to random if needed
    if (maxProb <= 0) {
        for (int i = 0; i < BOARDSIZE; i++) {
            for (int j = 0; j < BOARDSIZE; j++) {
                if (memory.grid[i][j] == '?') {
                    return formatMove(i, j);
                }
            }
        }
        bestRow = rand() % BOARDSIZE;
        bestCol = rand() % BOARDSIZE;
    }
    
    return formatMove(bestRow, bestCol);
}

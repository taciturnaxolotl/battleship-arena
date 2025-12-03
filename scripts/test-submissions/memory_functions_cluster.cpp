// Cluster AI - targets high-density regions where ships are likely clustered
// Strategy: Focus fire on promising areas before moving to next cluster

#include "memory_functions_cluster.h"
#include "battleship.h"
#include "kasbs.h"
#include "memory.h"
#include <string>
#include <cstdlib>
#include <ctime>

using namespace std;

static int densityMap[BOARDSIZE][BOARDSIZE];
static int clusterCenterRow = -1;
static int clusterCenterCol = -1;
static int shotsInCurrentCluster = 0;
static const int maxShotsPerCluster = 12;

inline string formatMove(int row, int col) {
    char letter = static_cast<char>('A' + row);
    return string(1, letter) + to_string(col + 1);
}

void initMemoryCluster(ComputerMemory &memory) {
    srand(time(NULL));
    
    for (int i = 0; i < BOARDSIZE; i++) {
        for (int j = 0; j < BOARDSIZE; j++) {
            memory.grid[i][j] = '?';
            densityMap[i][j] = 0;
        }
    }
    
    clusterCenterRow = -1;
    clusterCenterCol = -1;
    shotsInCurrentCluster = 0;
}

void updateMemoryCluster(int row, int col, int result, ComputerMemory &memory) {
    if (result == HIT || result == SUNK) {
        memory.grid[row][col] = 'h';
        
        // Increase density around hits
        for (int i = row - 2; i <= row + 2; i++) {
            for (int j = col - 2; j <= col + 2; j++) {
                if (i >= 0 && i < BOARDSIZE && j >= 0 && j < BOARDSIZE) {
                    densityMap[i][j] += 10;
                }
            }
        }
    } else {
        memory.grid[row][col] = 'm';
        
        // Decrease density around misses
        for (int i = row - 1; i <= row + 1; i++) {
            for (int j = col - 1; j <= col + 1; j++) {
                if (i >= 0 && i < BOARDSIZE && j >= 0 && j < BOARDSIZE) {
                    densityMap[i][j] -= 2;
                    if (densityMap[i][j] < 0) densityMap[i][j] = 0;
                }
            }
        }
    }
}

void findBestCluster(int* centerRow, int* centerCol, const ComputerMemory &memory) {
    int maxDensity = -1;
    *centerRow = -1;
    *centerCol = -1;
    
    for (int i = 1; i < BOARDSIZE - 1; i++) {
        for (int j = 1; j < BOARDSIZE - 1; j++) {
            if (memory.grid[i][j] != '?') continue;
            
            // Calculate cluster density (3x3 area)
            int clusterDensity = 0;
            for (int di = -1; di <= 1; di++) {
                for (int dj = -1; dj <= 1; dj++) {
                    clusterDensity += densityMap[i + di][j + dj];
                }
            }
            
            if (clusterDensity > maxDensity) {
                maxDensity = clusterDensity;
                *centerRow = i;
                *centerCol = j;
            }
        }
    }
}

string smartMoveCluster(const ComputerMemory &memory) {
    // Find new cluster if needed
    if (clusterCenterRow == -1 || shotsInCurrentCluster >= maxShotsPerCluster) {
        findBestCluster(&clusterCenterRow, &clusterCenterCol, memory);
        shotsInCurrentCluster = 0;
        
        // No cluster found, pick any unknown cell
        if (clusterCenterRow == -1) {
            for (int i = 0; i < BOARDSIZE; i++) {
                for (int j = 0; j < BOARDSIZE; j++) {
                    if (memory.grid[i][j] == '?') {
                        return formatMove(i, j);
                    }
                }
            }
            return formatMove(rand() % BOARDSIZE, rand() % BOARDSIZE);
        }
    }
    
    // Shoot within current cluster
    int attempts = 0;
    while (attempts < 100) {
        int offsetRow = (rand() % 5) - 2;
        int offsetCol = (rand() % 5) - 2;
        
        int targetRow = clusterCenterRow + offsetRow;
        int targetCol = clusterCenterCol + offsetCol;
        
        if (targetRow >= 0 && targetRow < BOARDSIZE && 
            targetCol >= 0 && targetCol < BOARDSIZE && 
            memory.grid[targetRow][targetCol] == '?') {
            
            shotsInCurrentCluster++;
            return formatMove(targetRow, targetCol);
        }
        attempts++;
    }
    
    // Fallback
    shotsInCurrentCluster++;
    return formatMove(rand() % BOARDSIZE, rand() % BOARDSIZE);
}

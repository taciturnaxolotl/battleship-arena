#include "battleship.h"
#include "kasbs.h"
#include "memory.h"
#include <string>

using namespace std;

// Hunter AI - uses hunt/target mode (simpler than klukas)
inline bool onBoard(int row, int col) {
    return row >= 0 && row < BOARDSIZE && col >= 0 && col < BOARDSIZE;
}

void initMemoryHunter(ComputerMemory &memory) {
   memory.mode        =  RANDOM;
   memory.hitRow      = -1;
   memory.hitCol      = -1;
   memory.hitShip     =  NONE;
   memory.fireDir     =  NONE;
   memory.fireDist    =  1;
   memory.lastResult  =  NONE;

   for (int i = 0; i < BOARDSIZE; i++) {
      for (int j = 0; j < BOARDSIZE; j++) {
         memory.grid[i][j] = EMPTY_MARKER;
      }
   }
}

string smartMoveHunter(const ComputerMemory &memory) {
   if (memory.mode == RANDOM) {
      // Use checkerboard pattern for hunting
      for (int i = 0; i < BOARDSIZE; i++) {
         for (int j = 0; j < BOARDSIZE; j++) {
            if ((i + j) % 2 == 0 && memory.grid[i][j] == EMPTY_MARKER) {
               char letter = static_cast<char>('A' + i);
               return string(1, letter) + to_string(j + 1);
            }
         }
      }
      
      // If no checkerboard cells left, use any empty cell
      for (int i = 0; i < BOARDSIZE; i++) {
         for (int j = 0; j < BOARDSIZE; j++) {
            if (memory.grid[i][j] == EMPTY_MARKER) {
               char letter = static_cast<char>('A' + i);
               return string(1, letter) + to_string(j + 1);
            }
         }
      }
   }
   
   // Target mode - try adjacent cells
   int directions[4][2] = {{-1, 0}, {0, 1}, {1, 0}, {0, -1}}; // N, E, S, W
   int dirIdx = memory.fireDir;
   
   if (dirIdx == NONE) dirIdx = 0;
   
   // Try current direction
   for (int tries = 0; tries < 4; tries++) {
      int idx = (dirIdx + tries) % 4;
      int dr = directions[idx][0];
      int dc = directions[idx][1];
      int newRow = memory.hitRow + dr;
      int newCol = memory.hitCol + dc;
      
      if (onBoard(newRow, newCol) && memory.grid[newRow][newCol] == EMPTY_MARKER) {
         char letter = static_cast<char>('A' + newRow);
         return string(1, letter) + to_string(newCol + 1);
      }
   }
   
   // Fallback to random
   for (int i = 0; i < BOARDSIZE; i++) {
      for (int j = 0; j < BOARDSIZE; j++) {
         if (memory.grid[i][j] == EMPTY_MARKER) {
            char letter = static_cast<char>('A' + i);
            return string(1, letter) + to_string(j + 1);
         }
      }
   }
   
   return "A1";
}

void updateMemoryHunter(int row, int col, int result, ComputerMemory &memory) {
   memory.lastResult = result;
   char marker;
   if (isAMiss(result)) {
      marker = MISS_MARKER;
   } else {
      marker = HIT_MARKER;
   }
   memory.grid[row][col] = marker;
   
   if (memory.mode == RANDOM) {
      if (!isAMiss(result)) {
         // Got a hit, switch to target mode
         memory.mode = SEARCH;
         memory.hitRow = row;
         memory.hitCol = col;
         memory.fireDir = NORTH; // Start trying north
      }
   } else {
      // In target mode
      if (isASunk(result)) {
         // Sunk the ship, back to hunt mode
         memory.mode = RANDOM;
         memory.hitRow = -1;
         memory.hitCol = -1;
         memory.fireDir = NONE;
      } else if (!isAMiss(result)) {
         // Another hit, keep current direction
         // (fireDir stays the same)
      } else {
         // Miss in target mode, try next direction
         if (memory.fireDir == NORTH) memory.fireDir = EAST;
         else if (memory.fireDir == EAST) memory.fireDir = SOUTH;
         else if (memory.fireDir == SOUTH) memory.fireDir = WEST;
         else {
            // Tried all directions, back to hunt
            memory.mode = RANDOM;
            memory.hitRow = -1;
            memory.hitCol = -1;
            memory.fireDir = NONE;
         }
      }
   }
}

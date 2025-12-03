#include "battleship.h"
#include "kasbs.h"
#include "memory.h"
#include <string>
#include <cstdlib>

using namespace std;

// Edge AI - prioritizes edges and corners first
void initMemoryEdge(ComputerMemory &memory) {
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

string smartMoveEdge(const ComputerMemory &memory) {
   if (memory.mode == RANDOM) {
      // First shoot corners
      int corners[4][2] = {{0, 0}, {0, BOARDSIZE-1}, {BOARDSIZE-1, 0}, {BOARDSIZE-1, BOARDSIZE-1}};
      for (int c = 0; c < 4; c++) {
         int r = corners[c][0];
         int col = corners[c][1];
         if (memory.grid[r][col] == EMPTY_MARKER) {
            char letter = static_cast<char>('A' + r);
            return string(1, letter) + to_string(col + 1);
         }
      }
      
      // Then shoot edges
      for (int i = 0; i < BOARDSIZE; i++) {
         // Top edge
         if (memory.grid[0][i] == EMPTY_MARKER) {
            return string(1, 'A') + to_string(i + 1);
         }
         // Bottom edge
         if (memory.grid[BOARDSIZE-1][i] == EMPTY_MARKER) {
            char letter = static_cast<char>('A' + BOARDSIZE - 1);
            return string(1, letter) + to_string(i + 1);
         }
         // Left edge
         if (memory.grid[i][0] == EMPTY_MARKER) {
            char letter = static_cast<char>('A' + i);
            return string(1, letter) + to_string(1);
         }
         // Right edge
         if (memory.grid[i][BOARDSIZE-1] == EMPTY_MARKER) {
            char letter = static_cast<char>('A' + i);
            return string(1, letter) + to_string(BOARDSIZE);
         }
      }
      
      // Then fill in the rest
      for (int i = 0; i < BOARDSIZE; i++) {
         for (int j = 0; j < BOARDSIZE; j++) {
            if (memory.grid[i][j] == EMPTY_MARKER) {
               char letter = static_cast<char>('A' + i);
               return string(1, letter) + to_string(j + 1);
            }
         }
      }
   }
   
   // Target mode
   int directions[4][2] = {{-1, 0}, {0, 1}, {1, 0}, {0, -1}};
   for (int d = 0; d < 4; d++) {
      int newRow = memory.hitRow + directions[d][0];
      int newCol = memory.hitCol + directions[d][1];
      
      if (newRow >= 0 && newRow < BOARDSIZE && newCol >= 0 && newCol < BOARDSIZE &&
          memory.grid[newRow][newCol] == EMPTY_MARKER) {
         char letter = static_cast<char>('A' + newRow);
         return string(1, letter) + to_string(newCol + 1);
      }
   }
   
   return "A1";
}

void updateMemoryEdge(int row, int col, int result, ComputerMemory &memory) {
   memory.lastResult = result;
   char marker;
   if (isAMiss(result)) {
      marker = MISS_MARKER;
   } else {
      marker = HIT_MARKER;
   }
   memory.grid[row][col] = marker;
   
   if (memory.mode == RANDOM && !isAMiss(result)) {
      memory.mode = SEARCH;
      memory.hitRow = row;
      memory.hitCol = col;
   } else if (isASunk(result)) {
      memory.mode = RANDOM;
      memory.hitRow = -1;
      memory.hitCol = -1;
   }
}

#include "battleship.h"
#include "kasbs.h"
#include "memory.h"
#include <string>

using namespace std;

// Diagonal AI - shoots in diagonal patterns
void initMemoryDiagonal(ComputerMemory &memory) {
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

string smartMoveDiagonal(const ComputerMemory &memory) {
   if (memory.mode == RANDOM) {
      // Shoot in diagonal pattern first
      for (int i = 0; i < BOARDSIZE; i++) {
         for (int j = 0; j < BOARDSIZE; j++) {
            if ((i + j) % 3 == 0 && memory.grid[i][j] == EMPTY_MARKER) {
               char letter = static_cast<char>('A' + i);
               return string(1, letter) + to_string(j + 1);
            }
         }
      }
      
      // Fallback to any empty cell
      for (int i = 0; i < BOARDSIZE; i++) {
         for (int j = 0; j < BOARDSIZE; j++) {
            if (memory.grid[i][j] == EMPTY_MARKER) {
               char letter = static_cast<char>('A' + i);
               return string(1, letter) + to_string(j + 1);
            }
         }
      }
   }
   
   // Target mode - shoot adjacent cells
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

void updateMemoryDiagonal(int row, int col, int result, ComputerMemory &memory) {
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

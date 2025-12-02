#include "battleship.h"
#include "kasbs.h"
#include "memory.h"
#include <string>

using namespace std;

// Random AI - just picks random valid moves
void initMemoryRandom(ComputerMemory &memory) {
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

string smartMoveRandom(const ComputerMemory &memory) {
   // Find all empty cells
   for (int attempts = 0; attempts < 100; attempts++) {
      int row = rand() % BOARDSIZE;
      int col = rand() % BOARDSIZE;
      
      if (memory.grid[row][col] == EMPTY_MARKER) {
         char letter = static_cast<char>('A' + row);
         return string(1, letter) + to_string(col + 1);
      }
   }
   
   // Fallback: find first empty cell
   for (int i = 0; i < BOARDSIZE; i++) {
      for (int j = 0; j < BOARDSIZE; j++) {
         if (memory.grid[i][j] == EMPTY_MARKER) {
            char letter = static_cast<char>('A' + i);
            return string(1, letter) + to_string(j + 1);
         }
      }
   }
   
   return "A1"; // Should never reach here
}

void updateMemoryRandom(int row, int col, int result, ComputerMemory &memory) {
   memory.lastResult = result;
   char marker;
   if (isAMiss(result)) {
      marker = MISS_MARKER;
   } else {
      marker = HIT_MARKER;
   }
   memory.grid[row][col] = marker;
}

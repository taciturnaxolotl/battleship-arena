#include "memory_functions_walther.h"

using namespace std;

int shipCurrent;

// initMemory initializes the memory; at the outset of the game the grid of
// shots taken is empty, we've not hit any ships, and our player can only apply
// a general, somewhat random firing strategy until we get a hit on some ship
void initMemorywalther(ComputerMemory &memory) {
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

// complete this function so it produces a "smart" move based on the information
// which appears in the computer's memory
string smartMovewalther(const ComputerMemory &memory) {
   string move;
   int tempRow = memory.hitRow;
   int tempCol = memory.hitCol;   

   if (memory.mode == RANDOM) {
      move = randomMove();
   }
   else if (memory.mode == SEARCH) {
      move = memory.grid[tempRow][tempCol];
   }
   else if (memory.mode == DESTROY) {
      move = memory.grid[tempRow][tempCol];
   }
   return move;
}

// complete this function so it updates the computer's memory based on the
// result of the last shot at location (row, col)
void updateMemorywalther(int row, int col, int result, ComputerMemory &memory) {
   if (isASunk(result)) {
      if (shipCurrent == isShip(result)) {
         memory.mode = RANDOM;
         memory.fireDir = NORTH;
         memory.fireDist = 1;   	 
      }
      else {
         memory.fireDir += 1;
      }
      if (memory.mode == DESTROY) {
         if (memory.fireDir == NORTH) {
            memory.fireDir = SOUTH;
         }
         if (memory.fireDir == SOUTH) {
            memory.fireDir = NORTH;
         }
         if (memory.fireDir == EAST) {
            memory.fireDir = WEST;
         }
         if (memory.fireDir == WEST) {
            memory.fireDir = EAST;
         }
      }
   }
   else if (isAHit(result)) {
      if (memory.mode == RANDOM) {
	 row = memory.hitRow;
         col = memory.hitCol;
         memory.fireDir = NORTH;
	 shipCurrent = isShip(result);
      }
      else if (memory.mode == SEARCH) {
         if (shipCurrent == isShip(result)) {
            row = memory.hitRow;
            col = memory.hitCol;
            memory.fireDist += 1;
	    memory.mode = DESTROY;
	 }
	 else {
            memory.fireDir += 1;
	 }
      }
      else if (memory.mode == DESTROY) {
         if (shipCurrent == isShip(result)) {
            row = memory.hitRow;
            col = memory.hitCol;
            memory.fireDist += 1;
	 }
	 else {
            if (memory.fireDir == NORTH) {
               memory.fireDir = SOUTH;
	    }
	    if (memory.fireDir == SOUTH) {
               memory.fireDir = NORTH;
            }
	    if (memory.fireDir == EAST) {
               memory.fireDir = WEST;
            }
	    if (memory.fireDir == WEST) {
               memory.fireDir = EAST;
            }
	 }
      }
   }
   else if (isAMiss(result)) {
      if (memory.mode == RANDOM) {
         memory.mode == RANDOM;
      }
      else if (memory.mode == SEARCH) {
         memory.fireDir += 1;
      }
      else if (memory.mode == DESTROY) {
         if (memory.fireDir == NORTH) {
            memory.fireDir = SOUTH;
         }
         if (memory.fireDir == SOUTH) {
            memory.fireDir = NORTH;
         }
         if (memory.fireDir == EAST) {
            memory.fireDir = WEST;
         }
         if (memory.fireDir == WEST) {
            memory.fireDir = EAST;
         }
      }
   }
}

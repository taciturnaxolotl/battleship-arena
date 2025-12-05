#include "memory_functions_premkum.h"

using namespace std;

//these take the move of the smart computer
int lastRow;
int lastCol;

// initMemory initializes the memory; at the outset of the game the grid of
// shots taken is empty, we've not hit any ships, and our player can only apply
// a general, somewhat random firing strategy until we get a hit on some ship
void initMemoryPremkum(ComputerMemory &memory) {
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

//function that returns the row of the alphabet based on the integer recieved
string getRow(int row){
        string returnRow = "A";

        if(row <= 1){
                returnRow = "A";
        }
        if(row == 2){
                returnRow = "B";
        }
        if(row == 3){
                returnRow = "C";
        }
        if(row == 4){
                returnRow = "D";
        }
        if(row == 5){
                returnRow = "E";
        }
        if(row == 6){
                returnRow = "F";
        }
        if(row == 7){
                returnRow = "G";
        }
        if(row == 8){
                returnRow = "H";
        }
        if(row == 9){
                returnRow = "I";
        }
        if(row >= 10){
                returnRow = "J";
        }

        return returnRow;
}

// complete this function so it produces a "smart" move based on the information
// which appears in the computer's memory
string smartMovePremkum(const ComputerMemory &memory) {
   string move;

   if(memory.mode == RANDOM){
           move = randomMove();
   }

   if(memory.mode == SEARCH||memory.mode == DESTROY){
           if(memory.fireDir == NONE || memory.fireDir == WEST){
                   move = getRow(memory.hitRow - 1) + to_string(memory.hitCol);
           }
	   if(memory.fireDir == NORTH){
		   move = getRow(memory.hitRow) + to_string(memory.hitCol + 1);
	   }
           if(memory.fireDir == EAST){
                   move = getRow(memory.hitRow + 1) + to_string(memory.hitCol);
           }
           if(memory.fireDir == SOUTH){
                   move = getRow(memory.hitRow) + to_string(memory.hitCol - 1);
           }
   }

   lastRow = int(move[0]) - 64;
   lastCol = stoi(move.substr(1, 2));

   return move;
}


// complete this function so it updates the computer's memory based on the
// result of the last shot at location (row, col)
void updateMemoryPremkum(int row, int col, int result, ComputerMemory &memory) {

        if (memory.mode == RANDOM){
                if(isAHit(result)) {
			memory.hitRow = lastRow;
			memory.hitCol = lastCol;
                        memory.mode = SEARCH;
			memory.fireDir = 0;
                }
        }
        else if (memory.mode == SEARCH){
                if(isAHit(result)){
                        memory.hitRow = lastRow;
                        memory.hitCol = lastCol;
			if(isASunk(result)){
                                memory.mode = RANDOM;
				memory.fireDir = 0;
                        }
                        else{
                                memory.mode = DESTROY;
                        }
                }
		else{
			if(memory.fireDir <= 3){
				memory.fireDir = memory.fireDir + 1;
			}
			else{
				memory.fireDir = 1;
			}
		}
        }
        else {
                if(isASunk(result)){
                        memory.mode = RANDOM;
			memory.fireDir = 0;
		}
		else{
			if(isAMiss(result)){
				if(memory.fireDir == 1){
					memory.fireDir = 2;
				}
				else if(memory.fireDir == 2){
                                        memory.fireDir = 1;
                                }
                                else if(memory.fireDir == 3){
                                        memory.fireDir = 4;
                                }
                                else if(memory.fireDir == 4){
                                        memory.fireDir = 3;
                                }

			}
			else{
				if(memory.hitCol == 10 && memory.fireDir == 3){
					memory.fireDir = 4;
				}
				if(memory.hitRow == 10 && memory.fireDir == 2){
					memory.fireDir = 1;
				}
				if(memory.hitCol == 1 && memory.fireDir == 4){
					memory.fireDir = 3;
				}
				if(memory.hitRow == 1 && memory.fireDir == 1){
					memory.fireDir = 2;
				}
			}
		}
	}



}

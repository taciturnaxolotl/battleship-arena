#include "memory_functions_comito.h"

using namespace std;

void createMove(int row, int col, string &move);
int moveCheck(int row, int col, const ComputerMemory &memory);
void directionChecks(int &nextRow, int &nextCol, int dir, int offset, const ComputerMemory &memory);
int DirFlip(int currDir);

// initMemory initializes the memory; at the outset of the game the grid of
// shots taken is empty, we've not hit any ships, and our player can only apply
// a general, somewhat random firing strategy until we get a hit on some ship
void initMemorycomito(ComputerMemory &memory) {
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
string smartMovecomito(const ComputerMemory &memory) {
   string move;
   int checkResult = -1;
   int nextRow = -1;
   int nextCol = -1;
   if (memory.mode == SEARCH)
   {
      int i = 0;
      while (i <= 4 && checkResult != 1) 
      {   
        
         switch (memory.fireDir)
         {
            case 1:
               nextRow = memory.hitRow - 1;
               nextCol = memory.hitCol;
               break;
            case 2:
               nextRow = memory.hitRow + 1;
               nextCol = memory.hitCol;
               break;
            case 3:
               nextCol = memory.hitCol - 1;
               nextRow = memory.hitRow;
               break;
            case 4:
               nextCol = memory.hitCol + 1;
               nextRow = memory.hitRow;
               break;
         }
         i++;
         checkResult = moveCheck(nextRow, nextCol, memory);
      }
   }  
   if (memory.mode == DESTROY)
   {
      switch (memory.fireDir)
      {
         case 1:
            nextRow = memory.hitRow - memory.fireDist;
            nextCol = memory.hitCol;
            break;
         case 2:
            nextRow = memory.hitRow + memory.fireDist;
            nextCol = memory.hitCol;
            break;
         case 3:
            nextCol = memory.hitCol - memory.fireDist;
            nextRow = memory.hitRow;
            break;
         case 4:
            nextCol = memory.hitCol + memory.fireDist;
            nextRow = memory.hitRow;
            break;
      }
   }
   createMove(nextRow, nextCol, move);
   debug(move);
   return move;
}

void updateMemorycomito(int row, int col, int result, ComputerMemory &memory) {
   
   int moveRes = result / 10;
   int hitShipId = result % 10;

   if (memory.mode == RANDOM) //if random
   {
      if (result == 0) //if missed
      {
         memory.fireDir = 0;
      }
      else //if hit
      {
      memory.hitShip = isShip(result);
      memory.hitRow = row;
      memory.hitCol = col;
      memory.mode = SEARCH;
      memory.fireDir++;
      }
   }
   else if (memory.mode == SEARCH) //if search
   {
      if (result == 0) //if missed
      {
         memory.fireDir++;
         if (memory.fireDir > 4)
         {
            memory.mode = RANDOM;
            memory.fireDist = 1;
         }
      }
      else
      {
         memory.mode = DESTROY;
         memory.fireDist++;
      }
   }else //if destroy
   {
      int i = BOARDSIZE;
      int nextRow = -1;
      int nextCol = -1;
      int checkResult = -1;
      while (checkResult != 1 && i > 0)
      {
         switch (memory.fireDir)
         {
            case 1:
               nextRow = memory.hitRow - memory.fireDist;
               nextCol = memory.hitCol;
               break;
            case 2:
               nextRow = memory.hitRow + memory.fireDist;
               nextCol = memory.hitCol;
               break;
            case 3:
               nextCol = memory.hitCol - memory.fireDist;
               nextRow = memory.hitRow;
               break;
            case 4:
               nextCol = memory.hitCol + memory.fireDist;
               nextRow = memory.hitRow;
               break;
         }
         checkResult = moveCheck(nextRow, nextCol, memory);
         
         switch (checkResult)
         {
            case 0:
               memory.fireDir = DirFlip(memory.fireDir);
               memory.fireDist = 1;
               break;
            case 1:
               //check if should fire here and if yes fire
               if (moveRes != 0)
               {
                  
               }
               break;
            case 2:
               memory.fireDist += 1;
               break;
            case 3:
               memory.fireDir = DirFlip(memory.fireDir);
               memory.fireDist = 1;
               break;
         }
         i--;
      }
      if (i < 1)
      {
         memory.mode = RANDOM;
      }
   }
   //update memory grid
   if (result == 0)
   {
      memory.grid[row][col] = MISS_MARKER;
   }
   if (result == 1)
   {
      memory.grid[row][col] = HIT_MARKER;
   }
}

//I added this function to call within the move so that 
//I can input the row number and just get the letter out
//I didn't know how best to do this so I did whatever this is :/
void createMove(int row, int col, string &move)
{
   char letter = 'A';
   letter += row;
   move.push_back(letter);
   string number = to_string(col + 1);
   move.append(number);
}

int moveCheck(int row, int col, const ComputerMemory &memory)
{
   int success = 0;
   bool isMovePlayedYet = false;

   if (row < 0 || row >= BOARDSIZE || col < 0 || col >= BOARDSIZE)
   {
      success = false;
   }
   else
   {
      if(memory.grid[row][col] == EMPTY_MARKER)
      {
         success = 1;
      }else if (memory.grid[row][col] == HIT_MARKER)
      {
         success = 2;
      }else
      {
         success = 3;
      }
   }
   return success;
}
int DirFlip(int currDir)
{
   if (currDir == 1)
   {
      return 2;
   }
   if (currDir == 3)
   {
      return 4;
   }
   return currDir;
}
/* attempted function graveyard
{
   switch (dir)
   {
      case 1:
         nextRow = memory.hitRow - offset;
         nextCol = memory.hitCol;
         break;
      case 2:
         nextRow = memory.hitRow + offset;
         nextCol = memory.hitCol;
         break;
      case 3:
         nextCol = memory.hitCol - offset;
         nextRow = memory.hitRow;
         break;
      case 4:
         nextCol = memory.hitCol + offset;
         nextRow = memory.hitRow;
         break;
   }
}
*/

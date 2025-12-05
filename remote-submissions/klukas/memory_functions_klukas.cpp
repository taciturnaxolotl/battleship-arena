#include "memory_functions_klukas.h"
#include "battleship.h"
#include "kasbs.h"
#include "memory.h"
#include <string>
#include <vector>

using namespace std;

// extra helpers
inline bool onBoard(int row, int col) {
    return row >= 0 && row < BOARDSIZE && col >= 0 && col < BOARDSIZE;
}

inline void nextDelta(int dir, int &deltaRow, int &deltaCol) {
    if (dir == NORTH) { deltaRow = -1; deltaCol = 0; }
    else if (dir == EAST) { deltaRow = 0; deltaCol = 1; }
    else if (dir == SOUTH) { deltaRow = 1; deltaCol = 0; }
    else if (dir == WEST) { deltaRow = 0; deltaCol = -1; }
    else if (dir == NONE) {deltaRow = 0; deltaCol = 0; }
}

inline int getOppositeDirection(int dir) {
   if (dir == NORTH) return SOUTH;
   if (dir == EAST) return WEST;
   if (dir == SOUTH) return NORTH;
   if (dir == WEST) return EAST;
   else return NONE;
}

inline bool canStepFrom(int row, int col, int dir) {
    int deltaRow, deltaCol;
    nextDelta(dir, deltaRow, deltaCol);
    int newRow = row + deltaRow;
    int newCol = col + deltaCol;
    return onBoard(newRow, newCol);
}

inline string formatMove(int row, int col) {
    char letter = static_cast<char>('A' + row); // 0 -> 'A', 1 -> 'B', ...
    return string(1, letter) + to_string(col + 1);
}

inline int nextValidDir(int row, int col, const ComputerMemory &memory, int currentDir) {
   int ordered[4] = { NORTH, EAST, SOUTH, WEST };
   int curIdx = -1;
   for (int i = 0; i < 4; ++i) {
      if (ordered[i] == currentDir) {
         curIdx = i; break;
      }
   }

   for (int step = 1; step <= 4; ++step) {
      int idx = (curIdx + step) % 4;
      int d = ordered[idx];
      int dr=0, dc=0; nextDelta(d, dr, dc);
      int newRow = row + dr;
      int newCol = col + dc;
      if (!onBoard(newRow, newCol)) continue;
      if (memory.grid[newRow][newCol] != EMPTY_MARKER) continue;
      return d;
   }

   return NONE;
}

// Calculate probability that a cell could contain a ship
// Based on how many different ship placements include this cell
inline int calculateCellProbability(int row, int col, const ComputerMemory &memory) {
   if (memory.grid[row][col] != EMPTY_MARKER) {
      return 0; // Already shot
   }

   int probability = 0;
   int shipSizes[] = {AC_SIZE, BS_SIZE, CR_SIZE, SB_SIZE, DS_SIZE};

   // For each ship size, count how many valid placements include this cell
   for (int s = 0; s < 5; s++) {
      int shipSize = shipSizes[s];

      // Check horizontal placements
      for (int startCol = col - shipSize + 1; startCol <= col; startCol++) {
         if (startCol < 0 || startCol + shipSize > BOARDSIZE) continue;

         bool valid = true;
         for (int c = startCol; c < startCol + shipSize; c++) {
            if (memory.grid[row][c] == MISS_MARKER || memory.grid[row][c] == SUNK_MARKER) {
               valid = false;
               break;
            }
         }
         if (valid) probability++;
      }

      // Check vertical placements
      for (int startRow = row - shipSize + 1; startRow <= row; startRow++) {
         if (startRow < 0 || startRow + shipSize > BOARDSIZE) continue;

         bool valid = true;
         for (int r = startRow; r < startRow + shipSize; r++) {
            if (memory.grid[r][col] == MISS_MARKER || memory.grid[r][col] == SUNK_MARKER) {
               valid = false;
               break;
            }
         }
         if (valid) probability++;
      }
   }

   return probability;
}

// We do the vector thing because we could otherwise run into an
// infinite loop that is more and more likely the closer we get to cells being full
inline string getSmartRandomMove(const ComputerMemory &memory) {
   vector<pair<int, int>> emptyCells;

   // First pass: checkerboard pattern (parity) - most efficient
   for (int i = 0; i < BOARDSIZE; i++) {
      for (int j = 0; j < BOARDSIZE; j++) {
         if (memory.grid[i][j] == EMPTY_MARKER && (i + j) % 2 == 0) {
            emptyCells.push_back({i, j});
         }
      }
   }

   // Second pass: if no parity cells, use any empty cell
   if (emptyCells.empty()) {
      for (int i = 0; i < BOARDSIZE; i++) {
         for (int j = 0; j < BOARDSIZE; j++) {
            if (memory.grid[i][j] == EMPTY_MARKER) {
               emptyCells.push_back({i, j});
            }
         }
      }
   }

   // If no empty cells, return empty string
   if (emptyCells.empty()) {
      return "";
   }

   // Use probability density to weight the choice
   // Calculate probability for each candidate cell
   vector<int> probabilities;
   int maxProb = 0;

   for (const auto& cell : emptyCells) {
      int prob = calculateCellProbability(cell.first, cell.second, memory);
      probabilities.push_back(prob);
      if (prob > maxProb) maxProb = prob;
   }

   // If we have probability data, prefer high-probability cells
   if (maxProb > 0) {
      // Collect only the highest probability cells
      vector<pair<int, int>> highProbCells;
      for (size_t i = 0; i < emptyCells.size(); i++) {
         if (probabilities[i] == maxProb) {
            highProbCells.push_back(emptyCells[i]);
         }
      }

      if (!highProbCells.empty()) {
         int randomIndex = rand() % highProbCells.size();
         return formatMove(highProbCells[randomIndex].first, highProbCells[randomIndex].second);
      }
   }

   // Fallback: pick random from all candidates
   int randomIndex = rand() % emptyCells.size();
   return formatMove(emptyCells[randomIndex].first, emptyCells[randomIndex].second);
}

// initMemory initializes the memory; at the outset of the game the grid of
// shots taken is empty, we've not hit any ships. Our player can only apply
// a general, somewhat random firing strategy until we get a hit on some ship.
void initMemoryKlukas(ComputerMemory &memory) {
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

// Complete this function so it produces a "smart" move based on the information
// which appears in the computer's memory
string smartMoveKlukas(const ComputerMemory &memory) {
   if (memory.mode == RANDOM) {
      return getSmartRandomMove(memory);
   }

   int deltaRow = 0, deltaCol = 0;
   nextDelta(memory.fireDir, deltaRow, deltaCol);

   int row = memory.hitRow + deltaRow * memory.fireDist;
   int col = memory.hitCol + deltaCol * memory.fireDist;

      // Guard against off-board and reused shots
   if (!onBoard(row, col)) {
      debug("*** GUARD TRIPPED *** OFFBOARD: row=" + to_string(row) + " col=" + to_string(col) +
            " | hitRow=" + to_string(memory.hitRow) + " hitCol=" + to_string(memory.hitCol) +
            " | dir=" + to_string(memory.fireDir) + " dist=" + to_string(memory.fireDist) +
            " | mode=" + to_string(memory.mode) + " | dR=" + to_string(deltaRow) + " dC=" + to_string(deltaCol));
      return getSmartRandomMove(memory);
   }
   if (memory.grid[row][col] != EMPTY_MARKER) {
      debug("*** GUARD TRIPPED *** ALREADY FIRED: " + formatMove(row, col) + " (marker='" + string(1, memory.grid[row][col]) + "')" +
            " | hitRow=" + to_string(memory.hitRow) + " hitCol=" + to_string(memory.hitCol) +
            " | dir=" + to_string(memory.fireDir) + " dist=" + to_string(memory.fireDist) +
            " | mode=" + to_string(memory.mode) + " lastResult=" + to_string(memory.lastResult) +
            " | dR=" + to_string(deltaRow) + " dC=" + to_string(deltaCol));
      return getSmartRandomMove(memory);
   }

   return formatMove(row, col);
}

// Complete this function so it updates the computer's memory based on the
// result of the last shot at location (row, col)
void updateMemoryKlukas(int row, int col, int result, ComputerMemory &memory) {
   memory.lastResult = result;
   char marker;
   if (isAMiss(result)) marker = MISS_MARKER;
   else marker = HIT_MARKER;
   memory.grid[row][col] = marker;

   if (memory.mode == RANDOM) {
      if (!isAMiss(result)) {
         // Pick first available direction that hasn't been tried and is in bounds
         int firstDir = nextValidDir(row, col, memory, NONE);

         if (firstDir != NONE) {
            memory.mode = SEARCH;
            memory.hitRow = row;
            memory.hitCol = col;

            memory.fireDir = firstDir;
            memory.fireDist = 1;
         } else {
            // This would only be possible if its one of the last pieces to play
            // it sinks a ship which is very unlikely
         }
      }

      return;
   }

   if (memory.mode == SEARCH) {
      if (isAMiss(result)) {
         int nextDir = nextValidDir(memory.hitRow, memory.hitCol, memory, memory.fireDir);

         if (nextDir != NONE) {
            memory.fireDir = nextDir;
            memory.fireDist = 1;
         } else {
            memory.mode = RANDOM;
            memory.fireDir = NONE;
            memory.fireDist = 1;
         }

         return;
      }

      if (isASunk(result)) {
         memory.mode = RANDOM;
         memory.fireDir = NONE;
         memory.hitRow = -1;
         memory.hitCol = -1;
         memory.fireDist = 1;

         return;
      }

      if (!isAMiss(result)) {
         debug("SEARCH->DESTROY: Got 2nd hit at " + formatMove(row, col) +
               " | anchor=(" + to_string(memory.hitRow) + "," + to_string(memory.hitCol) +
               ") dir=" + to_string(memory.fireDir));
         memory.mode = DESTROY;
         memory.fireDist = 2; // We want to check beyond the search

         int deltaRow = 0, deltaCol = 0;
         nextDelta(memory.fireDir, deltaRow, deltaCol);

         int nextRow = memory.hitRow + deltaRow * memory.fireDist;
         int nextCol = memory.hitCol + deltaCol * memory.fireDist;

         if (onBoard(nextRow, nextCol) && memory.grid[nextRow][nextCol] == MISS_MARKER) {
            int opposite = getOppositeDirection(memory.fireDir);

            // if we haven't checked this spot yet in search mode
            if (opposite > memory.fireDir) {
               memory.fireDist = 1;
            }

            memory.fireDir = opposite;

            // Skip over any hits in the opposite direction
            int oppDr = 0, oppDc = 0;
            nextDelta(opposite, oppDr, oppDc);
            int oppDist = memory.fireDist;
            int oppRow = memory.hitRow + oppDr * oppDist;
            int oppCol = memory.hitCol + oppDc * oppDist;

            while (onBoard(oppRow, oppCol) && memory.grid[oppRow][oppCol] == HIT_MARKER) {
               oppDist++;
               oppRow = memory.hitRow + oppDr * oppDist;
               oppCol = memory.hitCol + oppDc * oppDist;
            }

            // Validate the final position before committing to it
            if (opposite != NONE && onBoard(oppRow, oppCol) && memory.grid[oppRow][oppCol] == EMPTY_MARKER) {
               memory.fireDist = oppDist;
            } else {
               // Can't fire in opposite direction either, go back to random
               memory.mode = RANDOM;
               memory.fireDir = NONE;
               memory.fireDist = 1;
            }
         } else {
            // Continue in same direction - start at distance 2 (we just hit at distance 1)
            memory.fireDist = 2;

            // Skip over any additional hits we may have already made in this direction
            int checkRow = memory.hitRow + deltaRow * memory.fireDist;
            int checkCol = memory.hitCol + deltaCol * memory.fireDist;

            while (onBoard(checkRow, checkCol) && memory.grid[checkRow][checkCol] == HIT_MARKER) {
               memory.fireDist++;
               checkRow = memory.hitRow + deltaRow * memory.fireDist;
               checkCol = memory.hitCol + deltaCol * memory.fireDist;
            }

            // If we ended up offboard or at a non-empty cell, try opposite direction
            if (!onBoard(checkRow, checkCol) || memory.grid[checkRow][checkCol] != EMPTY_MARKER) {
               int opposite = getOppositeDirection(memory.fireDir);
               int oppDr = 0, oppDc = 0;
               nextDelta(opposite, oppDr, oppDc);

               // Find the first empty cell in the opposite direction
               int oppDist = 1;
               int oppRow = memory.hitRow + oppDr * oppDist;
               int oppCol = memory.hitCol + oppDc * oppDist;

               // Skip over any hits in the opposite direction
               while (onBoard(oppRow, oppCol) && memory.grid[oppRow][oppCol] == HIT_MARKER) {
                  oppDist++;
                  oppRow = memory.hitRow + oppDr * oppDist;
                  oppCol = memory.hitCol + oppDc * oppDist;
               }

               if (opposite != NONE && onBoard(oppRow, oppCol) && memory.grid[oppRow][oppCol] == EMPTY_MARKER) {
                  memory.fireDir = opposite;
                  memory.fireDist = oppDist;
               } else {
                  // Can't fire in either direction, go back to random
                  memory.mode = RANDOM;
                  memory.fireDir = NONE;
                  memory.fireDist = 1;
               }
            }
         }

         return;
      }
   }

   if (memory.mode == DESTROY) {
      if (isASunk(result)) {
         memory.mode = RANDOM;
         memory.fireDir = NONE;
         memory.hitRow = -1;
         memory.hitCol = -1;
         memory.fireDist = 1;

         return;
      }

      if (!isAMiss(result)) {
         debug("DESTROY: Got hit at " + formatMove(row, col) +
               " | fireDist " + to_string(memory.fireDist) + "->" + to_string(memory.fireDist + 1));
         memory.fireDist++;

         int deltaRow, deltaCol;
         nextDelta(memory.fireDir, deltaRow, deltaCol);
         int nextRow = memory.hitRow + deltaRow * memory.fireDist;
         int nextCol = memory.hitCol + deltaCol * memory.fireDist;

         // Skip over cells we've already hit
         while (onBoard(nextRow, nextCol) &&
                memory.grid[nextRow][nextCol] == HIT_MARKER) {
            memory.fireDist++;
            nextRow = memory.hitRow + deltaRow * memory.fireDist;
            nextCol = memory.hitCol + deltaCol * memory.fireDist;
         }

         // After skipping, check if we have a valid position
         if (!onBoard(nextRow, nextCol) ||
             memory.grid[nextRow][nextCol] != EMPTY_MARKER) {
            // Can't continue in this direction, try opposite
            int opposite = getOppositeDirection(memory.fireDir);
            int oppDr = 0, oppDc = 0;
            nextDelta(opposite, oppDr, oppDc);

            // Find the first empty cell in the opposite direction
            int oppDist = 1;
            int oppRow = memory.hitRow + oppDr * oppDist;
            int oppCol = memory.hitCol + oppDc * oppDist;

            // Skip over any hits in the opposite direction
            while (onBoard(oppRow, oppCol) && memory.grid[oppRow][oppCol] == HIT_MARKER) {
               oppDist++;
               oppRow = memory.hitRow + oppDr * oppDist;
               oppCol = memory.hitCol + oppDc * oppDist;
            }

            if (opposite != NONE && onBoard(oppRow, oppCol) &&
                memory.grid[oppRow][oppCol] == EMPTY_MARKER
            ) {
               debug("DESTROY: Switching dir from " + to_string(memory.fireDir) + " to " + to_string(opposite) +
                     " at dist=" + to_string(oppDist) + " to fire at " + formatMove(oppRow, oppCol));
               memory.fireDir = opposite;
               memory.fireDist = oppDist;
            } else {
               memory.mode = RANDOM;
               memory.fireDir = NONE;
               memory.fireDist = 1;
            }
         }

         return;
      }

      if (isAMiss(result)) {
         int opposite = getOppositeDirection(memory.fireDir);
         int oppDr = 0, oppDc = 0;
         nextDelta(opposite, oppDr, oppDc);

         // Find the first empty cell in the opposite direction
         int oppDist = 1;
         int firstOppRow = memory.hitRow + oppDr * oppDist;
         int firstOppCol = memory.hitCol + oppDc * oppDist;

         // Skip over any hits in the opposite direction
         while (onBoard(firstOppRow, firstOppCol) && memory.grid[firstOppRow][firstOppCol] == HIT_MARKER) {
            oppDist++;
            firstOppRow = memory.hitRow + oppDr * oppDist;
            firstOppCol = memory.hitCol + oppDc * oppDist;
         }

         if (opposite != NONE && onBoard(firstOppRow, firstOppCol) &&
             memory.grid[firstOppRow][firstOppCol] == EMPTY_MARKER
         ) {
            debug("DESTROY after MISS: Switching dir from " + to_string(memory.fireDir) + " to " + to_string(opposite) +
                  " at dist=" + to_string(oppDist) + " to fire at " + formatMove(firstOppRow, firstOppCol));
            memory.fireDir = opposite;
            memory.fireDist = oppDist;
         } else {
            memory.mode = RANDOM;
            memory.fireDir = NONE;
            memory.fireDist = 1;
         }

         return;
      }
   }
}


#ifndef BATTLESHIP_LIGHT_H
#define BATTLESHIP_LIGHT_H

#include <iostream>
#include <cstdlib>
#include <ctime>
#include <string>
#include <vector>

// Use the same constants and types as the normal version
#include "kasbs.h"

using namespace std;

// Player types (not in kasbs.h)
const int HUMAN = 0;
const int COMPUTER = 1;

// Move validation (not in kasbs.h)
const int VALID_MOVE = 0;
const int ILLEGAL_FORMAT = 1;
const int REUSED_MOVE = 2;

// Position, Ship, and Board are compatible with normal version
struct Position {
    int startRow;
    int startCol;
    int orient;
};

struct Ship {
    Position pos;
    int size;
    int hitsToSink;
    char marker;
};

struct Board {
    char grid[BOARDSIZE][BOARDSIZE];
    Ship s[6]; // index 0 unused
};

// Core functions - lightweight implementations
void setDebugMode(bool enabled);
bool getGuardTripped();
void resetGuardTripped();
vector<string> getDebugLog();
void welcome(bool debug = false);
void clearTheScreen();
void pauseForEnter();
void writeMessage(int x, int y, string message);
void writeResult(int x, int y, int result, int playerType);
void displayBoard(int x, int y, int playerType, const Board &gameBoard);
void initializeBoard(Board &gameBoard, bool file = false);
int playMove(int row, int col, Board &gameBoard);
bool isAMiss(int playMoveResult);
bool isAHit(int playMoveResult);
bool isASunk(int playMoveResult);
int isShip(int playMoveResult);
string randomMove();
int checkMove(string move, const Board &gameBoard, int &row, int &col);
void debug(string s, int x = 22, int y = 1);
string numToString(int x);

#endif // BATTLESHIP_LIGHT_H

// Tournament battle runner - runs matches between two AI implementations
// Outputs results in parseable format

#include "battleship_light.h"
#include "memory.h"
#include <iostream>
#include <cstdlib>
#include <ctime>

using namespace std;

// Function pointers for the two AIs
void (*initMemory1)(ComputerMemory&) = nullptr;
string (*smartMove1)(const ComputerMemory&) = nullptr;
void (*updateMemory1)(int, int, int, ComputerMemory&) = nullptr;

void (*initMemory2)(ComputerMemory&) = nullptr;
string (*smartMove2)(const ComputerMemory&) = nullptr;
void (*updateMemory2)(int, int, int, ComputerMemory&) = nullptr;

struct MatchResult {
    int player1Wins = 0;
    int player2Wins = 0;
    int ties = 0;
    int totalMoves = 0;
};

MatchResult runMatch(int numGames) {
    MatchResult result;
    srand(time(NULL));
    
    for (int game = 0; game < numGames; game++) {
        Board board1, board2;
        ComputerMemory memory1, memory2;
        
        initializeBoard(board1);
        initializeBoard(board2);
        initMemory1(memory1);
        initMemory2(memory2);
        
        int shipsSunk1 = 0;
        int shipsSunk2 = 0;
        int moveCount = 0;
        
        while (true) {
            moveCount++;
            
            // Player 1 move
            string move1 = smartMove1(memory1);
            int row1, col1;
            int check1 = checkMove(move1, board2, row1, col1);
            while (check1 != VALID_MOVE) {
                move1 = randomMove();
                check1 = checkMove(move1, board2, row1, col1);
            }
            
            // Player 2 move
            string move2 = smartMove2(memory2);
            int row2, col2;
            int check2 = checkMove(move2, board1, row2, col2);
            while (check2 != VALID_MOVE) {
                move2 = randomMove();
                check2 = checkMove(move2, board1, row2, col2);
            }
            
            // Execute moves
            int result1 = playMove(row1, col1, board2);
            int result2 = playMove(row2, col2, board1);
            
            updateMemory1(row1, col1, result1, memory1);
            updateMemory2(row2, col2, result2, memory2);
            
            if (isASunk(result1)) shipsSunk1++;
            if (isASunk(result2)) shipsSunk2++;
            
            if (shipsSunk1 == 5 || shipsSunk2 == 5) {
                break;
            }
        }
        
        result.totalMoves += moveCount;
        
        if (shipsSunk1 == 5 && shipsSunk2 == 5) {
            result.ties++;
        } else if (shipsSunk1 == 5) {
            result.player1Wins++;
        } else {
            result.player2Wins++;
        }
    }
    
    return result;
}

int main(int argc, char* argv[]) {
    if (argc < 2) {
        cerr << "Usage: " << argv[0] << " <num_games>" << endl;
        return 1;
    }
    
    int numGames = atoi(argv[1]);
    if (numGames <= 0) numGames = 10;
    
    setDebugMode(false);
    
    MatchResult result = runMatch(numGames);
    
    // Output in parseable format
    cout << "PLAYER1_WINS=" << result.player1Wins << endl;
    cout << "PLAYER2_WINS=" << result.player2Wins << endl;
    cout << "TIES=" << result.ties << endl;
    cout << "TOTAL_MOVES=" << result.totalMoves << endl;
    cout << "AVG_MOVES=" << (result.totalMoves / numGames) << endl;
    
    return 0;
}

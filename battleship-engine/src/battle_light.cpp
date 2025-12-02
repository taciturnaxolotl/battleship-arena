// Lightweight cross-platform battleship implementation
// Author: Kieran Klukas
// Date: November 2025
// Purpose: Test smart battleship AI with benchmarking on non-Linux systems

#include "battleship_light.h"
#include "memory.h"
#include "memory_functions_klukas.h"
#include <iostream>
#include <chrono>
#include <thread>
#include <vector>
#include <mutex>
#include <atomic>

using namespace std;

struct BenchmarkStats {
    atomic<int> wins{0};
    atomic<int> losses{0};
    atomic<int> ties{0};
    atomic<long long> totalMoves{0};
    atomic<long long> totalTimeNs{0};
    atomic<int> minMovesWin{999999};
    atomic<int> maxMovesWin{0};
    atomic<int> minMovesLoss{999999};
    atomic<int> maxMovesLoss{0};
    
    mutex updateMutex;  // For min/max updates
};

void printStats(const BenchmarkStats &stats, int gamesPlayed) {
    const int MAX_MOVES = 200; // Theoretical max (both players shoot all 100 squares)
    double avgMoves = (double)stats.totalMoves.load() / gamesPlayed;
    double movesPercent = (avgMoves / MAX_MOVES) * 100.0;
    
    cout << "\n========== BENCHMARK RESULTS ==========" << endl;
    cout << "Games played: " << gamesPlayed << endl;
    cout << "Smart AI wins: " << stats.wins.load() << " (" 
         << (100.0 * stats.wins.load() / gamesPlayed) << "%)" << endl;
    cout << "Dumb AI wins: " << stats.losses.load() << " (" 
         << (100.0 * stats.losses.load() / gamesPlayed) << "%)" << endl;
    cout << "Ties: " << stats.ties.load() << endl;
    cout << "Avg moves per game: " << (int)avgMoves 
         << " (" << movesPercent << "% of max)" << endl;
    
    if (stats.wins.load() > 0) {
        cout << "Win move range: " << stats.minMovesWin.load() << "-" << stats.maxMovesWin.load() << endl;
    }
    if (stats.losses.load() > 0) {
        cout << "Loss move range: " << stats.minMovesLoss.load() << "-" << stats.maxMovesLoss.load() << endl;
    }
    
    double avgTimeMs = (double)stats.totalTimeNs.load() / gamesPlayed / 1000000.0;
    cout << "Avg time per game: " << avgTimeMs << "ms" << endl;
    cout << "========================================\n" << endl;
}

// Thread-safe game runner function
void runGames(int startGame, int endGame, BenchmarkStats &stats, bool logLosses, 
              atomic<int> &gamesCompleted, int totalGames, unsigned int threadSeed) {
    // Each thread gets its own random seed
    srand(threadSeed);
    
    for (int game = startGame; game < endGame; game++) {
        auto startTime = chrono::high_resolution_clock::now();
        
        Board          dumbComputerBoard, smartComputerBoard;
        ComputerMemory smartComputerMemory;

        string         dumbComputerMove, smartComputerMove;
        int            numDumbComputerShipsSunk = 0;
        int            numSmartComputerShipsSunk = 0;
        int            dumbComputerRow, dumbComputerColumn;
        int            smartComputerRow, smartComputerColumn;
        int            checkValue, dumbComputerResult, smartComputerResult;
        int            moveCount = 0;

        initializeBoard(dumbComputerBoard);
        initializeBoard(smartComputerBoard);
        initMemoryKlukas(smartComputerMemory);

        while (true) {
            moveCount++;

            // Dumb computer move
            dumbComputerMove = randomMove();
            checkValue = checkMove(dumbComputerMove, smartComputerBoard,
                                   dumbComputerRow, dumbComputerColumn);

            while (checkValue != VALID_MOVE) {
                dumbComputerMove = randomMove();
                checkValue = checkMove(dumbComputerMove, smartComputerBoard,
                                       dumbComputerRow, dumbComputerColumn);
            }

            // Smart computer move
            smartComputerMove = smartMoveKlukas(smartComputerMemory);
            int checkResult = checkMove(smartComputerMove, dumbComputerBoard,
                                        smartComputerRow, smartComputerColumn);

            while (checkResult != VALID_MOVE) {
                smartComputerMove = randomMove();
                checkResult = checkMove(smartComputerMove, dumbComputerBoard,
                                        smartComputerRow, smartComputerColumn);
            }

            // Execute moves
            dumbComputerResult = playMove(dumbComputerRow, dumbComputerColumn,
                                          smartComputerBoard);
            smartComputerResult = playMove(smartComputerRow, smartComputerColumn,
                                           dumbComputerBoard);
            updateMemoryKlukas(smartComputerRow, smartComputerColumn,
                               smartComputerResult, smartComputerMemory);

            if (isASunk(dumbComputerResult)) {
                numDumbComputerShipsSunk++;
            }
            if (isASunk(smartComputerResult)) {
                numSmartComputerShipsSunk++;
            }

            if (numDumbComputerShipsSunk == 5 || numSmartComputerShipsSunk == 5) {
                break;
            }
        }

        auto endTime = chrono::high_resolution_clock::now();
        auto duration = chrono::duration_cast<chrono::nanoseconds>(endTime - startTime);

        // Update stats atomically
        stats.totalMoves += moveCount;
        stats.totalTimeNs += duration.count();

        if (numDumbComputerShipsSunk == 5 && numSmartComputerShipsSunk == 5) {
            stats.ties++;
        } else if (numSmartComputerShipsSunk == 5) {
            stats.wins++;
            
            // Update min/max with mutex protection
            lock_guard<mutex> lock(stats.updateMutex);
            int currentMin = stats.minMovesWin.load();
            if (moveCount < currentMin) stats.minMovesWin = moveCount;
            int currentMax = stats.maxMovesWin.load();
            if (moveCount > currentMax) stats.maxMovesWin = moveCount;
        } else {
            stats.losses++;
            
            lock_guard<mutex> lock(stats.updateMutex);
            int currentMin = stats.minMovesLoss.load();
            if (moveCount < currentMin) stats.minMovesLoss = moveCount;
            int currentMax = stats.maxMovesLoss.load();
            if (moveCount > currentMax) stats.maxMovesLoss = moveCount;
        }
        
        // Update progress counter
        gamesCompleted++;
    }
}

int main(int argc, char* argv[]) {
    bool benchmark = false;
    bool verbose = false;
    bool logLosses = false;
    bool catchGuards = false;
    int numGames = 1;

    // Parse command line args
    for (int i = 1; i < argc; i++) {
        string arg = argv[i];
        if (arg == "--benchmark" || arg == "-b") {
            benchmark = true;
            if (i + 1 < argc) {
                numGames = atoi(argv[++i]);
                if (numGames <= 0) numGames = 100;
            } else {
                numGames = 100;
            }
        } else if (arg == "--verbose" || arg == "-v") {
            verbose = true;
        } else if (arg == "--log-losses" || arg == "-l") {
            logLosses = true;
        } else if (arg == "--catch-guards" || arg == "-g") {
            catchGuards = true;
        }
    }

    BenchmarkStats stats;
    srand(time(NULL));
    
    // Catch-guards mode: run games until we hit a guard
    if (catchGuards) {
        cout << "Running games until guard is tripped..." << endl;
        int gamesRun = 0;
        
        while (true) {
            gamesRun++;
            resetGuardTripped();
            
            Board          dumbComputerBoard, smartComputerBoard;
            ComputerMemory smartComputerMemory;
            string         dumbComputerMove, smartComputerMove;
            int            numDumbComputerShipsSunk = 0;
            int            numSmartComputerShipsSunk = 0;
            int            dumbComputerRow, dumbComputerColumn;
            int            smartComputerRow, smartComputerColumn;
            int            checkValue, dumbComputerResult, smartComputerResult;
            int            moveCount = 0;

            initializeBoard(dumbComputerBoard);
            initializeBoard(smartComputerBoard);
            initMemoryKlukas(smartComputerMemory);

            bool guardTripped = false;
            while (true) {
                moveCount++;

                // Dumb computer move
                dumbComputerMove = randomMove();
                checkValue = checkMove(dumbComputerMove, smartComputerBoard,
                                       dumbComputerRow, dumbComputerColumn);
                while (checkValue != VALID_MOVE) {
                    dumbComputerMove = randomMove();
                    checkValue = checkMove(dumbComputerMove, smartComputerBoard,
                                           dumbComputerRow, dumbComputerColumn);
                }

                // Smart computer move
                smartComputerMove = smartMoveKlukas(smartComputerMemory);
                
                // Check if guard was tripped
                if (getGuardTripped()) {
                    guardTripped = true;
                    break;
                }
                
                int checkResult = checkMove(smartComputerMove, dumbComputerBoard,
                                            smartComputerRow, smartComputerColumn);
                while (checkResult != VALID_MOVE) {
                    smartComputerMove = randomMove();
                    checkResult = checkMove(smartComputerMove, dumbComputerBoard,
                                            smartComputerRow, smartComputerColumn);
                }

                // Execute moves
                dumbComputerResult = playMove(dumbComputerRow, dumbComputerColumn,
                                              smartComputerBoard);
                smartComputerResult = playMove(smartComputerRow, smartComputerColumn,
                                               dumbComputerBoard);
                updateMemoryKlukas(smartComputerRow, smartComputerColumn,
                                   smartComputerResult, smartComputerMemory);

                if (isASunk(dumbComputerResult)) {
                    numDumbComputerShipsSunk++;
                }
                if (isASunk(smartComputerResult)) {
                    numSmartComputerShipsSunk++;
                }

                if (numDumbComputerShipsSunk == 5 || numSmartComputerShipsSunk == 5) {
                    break;
                }
            }
            
            if (guardTripped) {
                cout << "\n==================================" << endl;
                cout << "GUARD TRIPPED after " << gamesRun << " games!" << endl;
                cout << "==================================" << endl;
                cout << "\nDebug log (last 50 entries):" << endl;
                cout << "----------------------------------" << endl;
                
                vector<string> log = getDebugLog();
                int start = max(0, (int)log.size() - 50);
                for (int i = start; i < (int)log.size(); i++) {
                    cout << log[i] << endl;
                }
                
                return 0;
            }
            
            if (gamesRun % 100 == 0) {
                cout << "Completed " << gamesRun << " games..." << endl;
            }
        }
    }

    if (!benchmark) {
        setDebugMode(true);
        welcome(true);
        verbose = true;  // Always show moves in interactive mode
    } else {
        setDebugMode(false);
        
        // Determine number of threads (use hardware concurrency)
        unsigned int numThreads = thread::hardware_concurrency();
        if (numThreads == 0) numThreads = 4; // Fallback if detection fails
        
        cout << "Running " << numGames << " games on " << numThreads << " threads..." << endl;
        
        // Progress tracking
        atomic<int> gamesCompleted{0};
        
        // Launch threads
        vector<thread> threads;
        int gamesPerThread = numGames / numThreads;
        int remainder = numGames % numThreads;
        
        int startGame = 0;
        for (unsigned int t = 0; t < numThreads; t++) {
            int endGame = startGame + gamesPerThread + (t < (unsigned)remainder ? 1 : 0);
            unsigned int threadSeed = time(NULL) + t; // Unique seed per thread
            
            threads.emplace_back(runGames, startGame, endGame, ref(stats), 
                               logLosses, ref(gamesCompleted), numGames, threadSeed);
            startGame = endGame;
        }
        
        // Progress monitoring thread
        thread progressThread([&]() {
            int lastReported = 0;
            int interval;
            if (numGames >= 10000) interval = 1000;
            else if (numGames >= 1000) interval = 100;
            else if (numGames >= 100) interval = 10;
            else interval = numGames / 5;
            
            while (gamesCompleted.load() < numGames) {
                this_thread::sleep_for(chrono::milliseconds(100));
                int completed = gamesCompleted.load();
                if (interval > 0 && completed >= lastReported + interval) {
                    cout << "Completed " << completed << " games..." << endl;
                    lastReported = (completed / interval) * interval;
                }
            }
        });
        
        // Wait for all game threads to complete
        for (auto &t : threads) {
            t.join();
        }
        
        // Stop progress thread
        progressThread.join();
    }

    if (!benchmark) {
        // Single game mode (existing code)
        Board          dumbComputerBoard, smartComputerBoard;
        ComputerMemory smartComputerMemory;

        string         dumbComputerMove, smartComputerMove;
        int            numDumbComputerShipsSunk = 0;
        int            numSmartComputerShipsSunk = 0;
        int            dumbComputerRow, dumbComputerColumn;
        int            smartComputerRow, smartComputerColumn;
        int            checkValue, dumbComputerResult, smartComputerResult;

        initializeBoard(dumbComputerBoard);
        initializeBoard(smartComputerBoard);
        initMemoryKlukas(smartComputerMemory);

        while (true) {
            if (verbose) {
                clearTheScreen();
                cout << "Dumb Computer Board:" << endl;
                displayBoard(1, 5, HUMAN, dumbComputerBoard);
                cout << "Smart Computer Board:" << endl;
                displayBoard(1, 40, HUMAN, smartComputerBoard);
            }

            // Dumb computer move
            dumbComputerMove = randomMove();
            checkValue = checkMove(dumbComputerMove, smartComputerBoard,
                                   dumbComputerRow, dumbComputerColumn);

            while (checkValue != VALID_MOVE) {
                dumbComputerMove = randomMove();
                checkValue = checkMove(dumbComputerMove, smartComputerBoard,
                                       dumbComputerRow, dumbComputerColumn);
            }

            // Smart computer move
            smartComputerMove = smartMoveKlukas(smartComputerMemory);
            int checkResult = checkMove(smartComputerMove, dumbComputerBoard,
                                        smartComputerRow, smartComputerColumn);

            while (checkResult != VALID_MOVE) {
                if (verbose) {
                    debug("INVALID! Using random instead", 0, 0);
                }
                smartComputerMove = randomMove();
                checkResult = checkMove(smartComputerMove, dumbComputerBoard,
                                        smartComputerRow, smartComputerColumn);
            }

            // Execute moves
            dumbComputerResult = playMove(dumbComputerRow, dumbComputerColumn,
                                          smartComputerBoard);
            smartComputerResult = playMove(smartComputerRow, smartComputerColumn,
                                           dumbComputerBoard);
            updateMemoryKlukas(smartComputerRow, smartComputerColumn,
                               smartComputerResult, smartComputerMemory);

            if (verbose) {
                clearTheScreen();
                cout << "Dumb Computer Board:" << endl;
                displayBoard(1, 5, HUMAN, dumbComputerBoard);
                cout << "Smart Computer Board:" << endl;
                displayBoard(1, 40, HUMAN, smartComputerBoard);

                writeMessage(15, 0, "The dumb  computer chooses:  " + dumbComputerMove);
                writeMessage(16, 0, "The smart computer chooses:  " + smartComputerMove);

                writeResult(18, 0, dumbComputerResult, COMPUTER);
                writeResult(19, 0, smartComputerResult, HUMAN);
                
                // Delay so the game is watchable
                this_thread::sleep_for(chrono::milliseconds(50));
            }

            if (isASunk(dumbComputerResult)) {
                numDumbComputerShipsSunk++;
            }
            if (isASunk(smartComputerResult)) {
                numSmartComputerShipsSunk++;
            }

            if (numDumbComputerShipsSunk == 5 || numSmartComputerShipsSunk == 5) {
                break;
            }
        }

        cout << "\nFinal Dumb Computer Board:" << endl;
        displayBoard(1, 5, HUMAN, dumbComputerBoard);
        cout << "Final Smart Computer Board:" << endl;
        displayBoard(1, 40, HUMAN, smartComputerBoard);

        if (numDumbComputerShipsSunk == 5 && numSmartComputerShipsSunk == 5) {
            writeMessage(21, 1, "The game is a tie.");
        } else if (numDumbComputerShipsSunk == 5) {
            writeMessage(21, 1, "Amazing, the dumb computer won.");
        } else {
            writeMessage(21, 1, "Smart AI won! As it should.");
        }
    }

    if (benchmark) {
        printStats(stats, numGames);
    }

    return 0;
}

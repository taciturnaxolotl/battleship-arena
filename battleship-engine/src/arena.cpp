// Arena - orchestrates battleship matches between two isolated player processes
// Each player runs in its own process, communicating via stdin/stdout pipes
// This prevents any cross-player memory access exploits
//
// Usage: arena <num_games> <player1_binary> <player2_binary>
//
// Output format (for Go runner to parse):
//   PLAYER1_WINS=<n>
//   PLAYER2_WINS=<n>
//   TIES=<n>
//   TOTAL_MOVES=<n>
//   AVG_MOVES=<n>

#include "battleship.h"
#include "memory.h"

#include <iostream>
#include <string>
#include <sstream>
#include <cstdlib>
#include <ctime>
#include <chrono>
#include <vector>
#include <unistd.h>
#include <sys/wait.h>
#include <signal.h>
#include <fcntl.h>
#include <poll.h>
#include <errno.h>
#include <cstring>

using namespace std;

const int TIMEOUT_MS = 5000;
const int MAX_INVALID_MOVES = 10;

struct PlayerProcess {
    pid_t pid;
    int stdinFd;
    int stdoutFd;
    string name;
    bool alive;
    
    PlayerProcess() : pid(-1), stdinFd(-1), stdoutFd(-1), alive(false) {}
};

bool sendLine(PlayerProcess &p, const string &line) {
    if (!p.alive) return false;
    string msg = line + "\n";
    ssize_t written = write(p.stdinFd, msg.c_str(), msg.size());
    return written == (ssize_t)msg.size();
}

bool readLine(PlayerProcess &p, string &line, int timeoutMs) {
    if (!p.alive) return false;
    
    line.clear();
    char buf[1];
    
    struct pollfd pfd;
    pfd.fd = p.stdoutFd;
    pfd.events = POLLIN;
    
    auto deadline = std::chrono::steady_clock::now() + std::chrono::milliseconds(timeoutMs);
    
    while (true) {
        auto remaining = std::chrono::duration_cast<std::chrono::milliseconds>(
            deadline - std::chrono::steady_clock::now()).count();
        if (remaining <= 0) return false;
        
        int ret = poll(&pfd, 1, remaining);
        if (ret <= 0) return false;
        
        if (pfd.revents & (POLLERR | POLLHUP | POLLNVAL)) {
            p.alive = false;
            return false;
        }
        
        ssize_t n = read(p.stdoutFd, buf, 1);
        if (n <= 0) {
            p.alive = false;
            return false;
        }
        
        if (buf[0] == '\n') {
            return true;
        }
        line += buf[0];
        
        if (line.size() > 1000) return false;
    }
}

PlayerProcess spawnPlayer(const string &binaryPath) {
    PlayerProcess p;
    p.name = binaryPath;
    
    int stdinPipe[2];
    int stdoutPipe[2];
    
    if (pipe(stdinPipe) < 0 || pipe(stdoutPipe) < 0) {
        cerr << "Failed to create pipes" << endl;
        return p;
    }
    
    pid_t pid = fork();
    if (pid < 0) {
        cerr << "Failed to fork" << endl;
        close(stdinPipe[0]); close(stdinPipe[1]);
        close(stdoutPipe[0]); close(stdoutPipe[1]);
        return p;
    }
    
    if (pid == 0) {
        // Child process
        close(stdinPipe[1]);
        close(stdoutPipe[0]);
        
        dup2(stdinPipe[0], STDIN_FILENO);
        dup2(stdoutPipe[1], STDOUT_FILENO);
        
        close(stdinPipe[0]);
        close(stdoutPipe[1]);
        
        execl(binaryPath.c_str(), binaryPath.c_str(), nullptr);
        cerr << "Failed to exec " << binaryPath << ": " << strerror(errno) << endl;
        _exit(1);
    }
    
    // Parent process
    close(stdinPipe[0]);
    close(stdoutPipe[1]);
    
    p.pid = pid;
    p.stdinFd = stdinPipe[1];
    p.stdoutFd = stdoutPipe[0];
    p.alive = true;
    
    // Set stdout non-blocking
    int flags = fcntl(p.stdoutFd, F_GETFL, 0);
    fcntl(p.stdoutFd, F_SETFL, flags | O_NONBLOCK);
    
    return p;
}

void killPlayer(PlayerProcess &p) {
    if (p.pid > 0) {
        kill(p.pid, SIGTERM);
        usleep(10000);
        kill(p.pid, SIGKILL);
        waitpid(p.pid, nullptr, 0);
    }
    if (p.stdinFd >= 0) close(p.stdinFd);
    if (p.stdoutFd >= 0) close(p.stdoutFd);
    p.alive = false;
}

bool handshake(PlayerProcess &p) {
    if (!sendLine(p, "HELLO 1")) return false;
    
    string response;
    if (!readLine(p, response, TIMEOUT_MS)) return false;
    
    return response == "HELLO OK";
}

bool initPlayer(PlayerProcess &p) {
    if (!sendLine(p, "INIT")) return false;
    
    string response;
    if (!readLine(p, response, TIMEOUT_MS)) return false;
    
    return response == "OK";
}

string getMove(PlayerProcess &p) {
    if (!sendLine(p, "GET_MOVE")) return "";
    
    string response;
    if (!readLine(p, response, TIMEOUT_MS)) return "";
    
    if (response.substr(0, 5) == "MOVE ") {
        return response.substr(5);
    }
    return "";
}

bool updatePlayer(PlayerProcess &p, int row, int col, int result) {
    ostringstream oss;
    oss << "UPDATE " << row << " " << col << " " << result;
    if (!sendLine(p, oss.str())) return false;
    
    string response;
    if (!readLine(p, response, TIMEOUT_MS)) return false;
    
    return response == "OK";
}

void quitPlayer(PlayerProcess &p) {
    sendLine(p, "QUIT");
    string response;
    readLine(p, response, 100);
}

struct MatchResult {
    int player1Wins = 0;
    int player2Wins = 0;
    int ties = 0;
    int totalMoves = 0;
};

int main(int argc, char* argv[]) {
    if (argc < 4) {
        cerr << "Usage: " << argv[0] << " <num_games> <player1_binary> <player2_binary>" << endl;
        return 1;
    }
    
    int numGames = atoi(argv[1]);
    if (numGames <= 0) numGames = 10;
    
    string player1Path = argv[2];
    string player2Path = argv[3];
    
    setDebugMode(false);
    srand(time(nullptr));
    
    // Spawn player processes
    PlayerProcess p1 = spawnPlayer(player1Path);
    PlayerProcess p2 = spawnPlayer(player2Path);
    
    if (!p1.alive || !p2.alive) {
        cerr << "Failed to spawn player processes" << endl;
        killPlayer(p1);
        killPlayer(p2);
        return 1;
    }
    
    // Handshake
    if (!handshake(p1)) {
        cerr << "Handshake failed with player 1" << endl;
        killPlayer(p1);
        killPlayer(p2);
        return 1;
    }
    
    if (!handshake(p2)) {
        cerr << "Handshake failed with player 2" << endl;
        killPlayer(p1);
        killPlayer(p2);
        return 1;
    }
    
    MatchResult result;
    
    for (int game = 0; game < numGames; game++) {
        // Check if players are still alive
        if (!p1.alive || !p2.alive) {
            cerr << "Player process died during match" << endl;
            break;
        }
        
        // Initialize boards (arena owns the authoritative state)
        Board board1, board2;
        initializeBoard(board1);
        initializeBoard(board2);
        
        // Initialize players for new game
        if (!initPlayer(p1) || !initPlayer(p2)) {
            cerr << "Failed to initialize players for game " << game << endl;
            break;
        }
        
        int shipsSunk1 = 0;
        int shipsSunk2 = 0;
        int moveCount = 0;
        int invalidMoves1 = 0;
        int invalidMoves2 = 0;
        
        while (true) {
            moveCount++;
            
            // Get move from player 1 (attacking board2)
            string move1 = getMove(p1);
            int row1, col1;
            int check1 = move1.empty() ? ILLEGAL_FORMAT : checkMove(move1, board2, row1, col1);
            
            while (check1 != VALID_MOVE) {
                invalidMoves1++;
                if (invalidMoves1 > MAX_INVALID_MOVES) {
                    move1 = randomMove();
                    check1 = checkMove(move1, board2, row1, col1);
                } else {
                    move1 = randomMove();
                    check1 = checkMove(move1, board2, row1, col1);
                }
            }
            
            // Get move from player 2 (attacking board1)
            string move2 = getMove(p2);
            int row2, col2;
            int check2 = move2.empty() ? ILLEGAL_FORMAT : checkMove(move2, board1, row2, col2);
            
            while (check2 != VALID_MOVE) {
                invalidMoves2++;
                if (invalidMoves2 > MAX_INVALID_MOVES) {
                    move2 = randomMove();
                    check2 = checkMove(move2, board1, row2, col2);
                } else {
                    move2 = randomMove();
                    check2 = checkMove(move2, board1, row2, col2);
                }
            }
            
            // Execute moves on the authoritative boards
            int result1 = playMove(row1, col1, board2);
            int result2 = playMove(row2, col2, board1);
            
            // Update players with results
            if (!updatePlayer(p1, row1, col1, result1)) {
                p1.alive = false;
                break;
            }
            if (!updatePlayer(p2, row2, col2, result2)) {
                p2.alive = false;
                break;
            }
            
            if (isASunk(result1)) shipsSunk1++;
            if (isASunk(result2)) shipsSunk2++;
            
            if (shipsSunk1 == 5 || shipsSunk2 == 5) {
                break;
            }
            
            // Safety: prevent infinite games
            if (moveCount > 200) {
                break;
            }
        }
        
        result.totalMoves += moveCount;
        
        if (shipsSunk1 == 5 && shipsSunk2 == 5) {
            result.ties++;
        } else if (shipsSunk1 == 5) {
            result.player1Wins++;
        } else if (shipsSunk2 == 5) {
            result.player2Wins++;
        } else {
            // Game ended abnormally (player died or max moves)
            // Count as a tie or handle as you prefer
            result.ties++;
        }
    }
    
    // Clean up
    quitPlayer(p1);
    quitPlayer(p2);
    killPlayer(p1);
    killPlayer(p2);
    
    // Output results in format Go expects
    cout << "PLAYER1_WINS=" << result.player1Wins << endl;
    cout << "PLAYER2_WINS=" << result.player2Wins << endl;
    cout << "TIES=" << result.ties << endl;
    cout << "TOTAL_MOVES=" << result.totalMoves << endl;
    cout << "AVG_MOVES=" << (numGames > 0 ? result.totalMoves / numGames : 0) << endl;
    
    return 0;
}

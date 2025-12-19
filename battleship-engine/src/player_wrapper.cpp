// Player wrapper - isolates student AI code in its own process
// Communicates with arena via stdin/stdout line-based protocol
//
// Protocol:
//   Arena -> Player: HELLO 1
//   Player -> Arena: HELLO OK
//   Arena -> Player: INIT
//   Player -> Arena: OK
//   Arena -> Player: GET_MOVE
//   Player -> Arena: MOVE <cell>  (e.g., MOVE A5)
//   Arena -> Player: UPDATE <row> <col> <result>
//   Player -> Arena: OK
//   Arena -> Player: QUIT
//   Player -> Arena: OK

#include "battleship.h"
#include "memory.h"
#include PLAYER_HEADER

#include <iostream>
#include <sstream>
#include <string>

#define CONCAT_INNER(a, b) a##b
#define CONCAT(a, b) CONCAT_INNER(a, b)

#define initMemoryFunc CONCAT(initMemory, PLAYER_SUFFIX)
#define smartMoveFunc CONCAT(smartMove, PLAYER_SUFFIX)
#define updateMemoryFunc CONCAT(updateMemory, PLAYER_SUFFIX)

static void doInit(ComputerMemory &mem) {
    initMemoryFunc(mem);
}

static std::string doSmartMove(const ComputerMemory &mem) {
    return smartMoveFunc(mem);
}

static void doUpdate(int row, int col, int result, ComputerMemory &mem) {
    updateMemoryFunc(row, col, result, mem);
}

int main() {
    std::ios::sync_with_stdio(false);
    std::cin.tie(nullptr);

    ComputerMemory memory{};
    bool initialized = false;

    std::string line;

    auto sendLine = [](const std::string &s) {
        std::cout << s << "\n";
        std::cout.flush();
    };

    // Handshake
    if (!std::getline(std::cin, line)) {
        return 0;
    }

    {
        std::istringstream iss(line);
        std::string cmd;
        int version;
        iss >> cmd >> version;
        if (cmd != "HELLO" || version != 1) {
            sendLine("ERROR bad_hello");
            return 1;
        }
        sendLine("HELLO OK");
    }

    while (std::getline(std::cin, line)) {
        if (line.empty()) continue;

        std::istringstream iss(line);
        std::string cmd;
        iss >> cmd;

        if (cmd == "INIT") {
            memory = ComputerMemory{};
            doInit(memory);
            initialized = true;
            sendLine("OK");
        } else if (cmd == "GET_MOVE") {
            if (!initialized) {
                sendLine("ERROR not_initialized");
                continue;
            }
            std::string move = doSmartMove(memory);
            if (move.empty()) {
                sendLine("ERROR empty_move");
            } else {
                sendLine("MOVE " + move);
            }
        } else if (cmd == "UPDATE") {
            int row, col, result;
            if (!(iss >> row >> col >> result)) {
                sendLine("ERROR bad_update_args");
                continue;
            }
            if (!initialized) {
                sendLine("ERROR not_initialized");
                continue;
            }
            doUpdate(row, col, result, memory);
            sendLine("OK");
        } else if (cmd == "QUIT") {
            sendLine("OK");
            break;
        } else {
            sendLine("ERROR unknown_command");
        }
    }

    return 0;
}

// hot_reload.cpp
// Watches for changes in Go source files and restarts the Go server.

#include <iostream>
#include <dirent.h>
#include <sys/stat.h>
#include <cstring>
#include <chrono>
#include <thread>
#include <map>
#include <cstdlib>
#include <csignal>
#include <unistd.h>
#include <sys/wait.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <errno.h>
#include <cstdio>

static volatile sig_atomic_t stopRequested = 0;
static pid_t serverPid = -1;

void handleSignal(int) {
    stopRequested = 1;
    if (serverPid > 0) {
        kill(serverPid, SIGTERM);
    }
}
// Recursively scan directory for .go files and return map of path -> mtime (seconds since epoch)
std::map<std::string, uint64_t> getFileTimes(const std::string& dir) {
    std::map<std::string, uint64_t> times;
    std::vector<std::string> stack;
    stack.push_back(dir);

    while (!stack.empty()) {
        std::string current = stack.back();
        stack.pop_back();

        DIR* dp = opendir(current.c_str());
        if (!dp) continue;

        struct dirent* entry;
        while ((entry = readdir(dp)) != nullptr) {
            const char* name = entry->d_name;
            if (strcmp(name, ".") == 0 || strcmp(name, "..") == 0) continue;
            std::string path = current + "/" + name;
            struct stat st;
            if (stat(path.c_str(), &st) == -1) continue;
            if (S_ISDIR(st.st_mode)) {
                stack.push_back(path);
            } else if (S_ISREG(st.st_mode)) {
                // check .go extension
                if (path.size() >= 3 && path.substr(path.size()-3) == ".go") {
                    times[path] = static_cast<uint64_t>(st.st_mtime);
                }
            }
        }
        closedir(dp);
    }
    return times;
}

bool filesChanged(const std::map<std::string, uint64_t>& oldTimes,
                 const std::map<std::string, uint64_t>& newTimes) {
    if (oldTimes.size() != newTimes.size()) return true;
    for (const auto& kv : newTimes) {
        const auto& path = kv.first;
        uint64_t time = kv.second;
        auto it = oldTimes.find(path);
        if (it == oldTimes.end() || it->second != time) return true;
    }
    return false;
}

pid_t startServer(const std::string& cmd) {
    pid_t pid = fork();
    if (pid == 0) {
        // Child
        execlp("sh", "sh", "-c", cmd.c_str(), (char*)NULL);
        _exit(127);
    }
    return pid;
}

// Wait for a given child pid to exit (blocking). Returns exit status or -1 on error.
int waitForChild(pid_t pid) {
    if (pid <= 0) return -1;
    int status = 0;
    pid_t w = waitpid(pid, &status, 0);
    if (w == pid) {
        if (WIFEXITED(status)) return WEXITSTATUS(status);
        if (WIFSIGNALED(status)) return 128 + WTERMSIG(status);
    }
    return -1;
}

// Reap the server child non-blocking; returns true if the child exited and was reaped.
bool reapChildIfExited() {
    if (serverPid <= 0) return false;
    int status = 0;
    pid_t w = waitpid(serverPid, &status, WNOHANG);
    if (w == 0) return false; // still running
    if (w == serverPid) {
        std::cerr << "[hot-reload] Server (PID " << serverPid << ") exited" << std::endl;
        serverPid = -1;
        return true;
    }
    return false;
}

// Check if TCP port is in use by attempting to connect to localhost:port.
// Returns true if a listener accepts the connection.
bool portInUse(int port) {
    int sock = socket(AF_INET, SOCK_STREAM, 0);
    if (sock < 0) return true; // conservatively assume in use
    struct sockaddr_in addr{};
    addr.sin_family = AF_INET;
    addr.sin_port = htons(port);
    inet_pton(AF_INET, "127.0.0.1", &addr.sin_addr);

    // attempt to connect (non-blocking connect with short timeout would be nicer,
    // but a blocking connect here is fine because it's local and quick)
    int res = connect(sock, (struct sockaddr*)&addr, sizeof(addr));
    if (res == 0) {
        close(sock);
        return true; // connection succeeded -> port in use
    }
    // if connection refused, port not in use; otherwise conservatively treat as in use
    if (errno == ECONNREFUSED) {
        close(sock);
        return false;
    }
    close(sock);
    return true;
}

// Return PID of the process listening on port, or -1 if none / on error.
int getListeningPid(int port) {
    char cmd[128];
    snprintf(cmd, sizeof(cmd), "lsof -iTCP:%d -sTCP:LISTEN -n -P -F p 2>/dev/null", port);
    FILE* pipe = popen(cmd, "r");
    if (!pipe) return -1;
    char buf[128];
    int pid = -1;
    while (fgets(buf, sizeof(buf), pipe)) {
        // each record like: p12345\n
        if (buf[0] == 'p') {
            pid = atoi(buf + 1);
            break;
        }
    }
    pclose(pipe);
    return pid;
}

int main() {
    std::string goRunCmd = "go run ./cmd/api/main.go";
    std::string srcDir = ".";

    // Setup signal handlers
    struct sigaction sa{};
    sa.sa_handler = handleSignal;
    sigemptyset(&sa.sa_mask);
    sa.sa_flags = 0;
    sigaction(SIGINT, &sa, nullptr);
    sigaction(SIGTERM, &sa, nullptr);

    // Initial scan + start server
    auto fileTimes = getFileTimes(srcDir);
    std::cout << "[hot-reload] Starting server..." << std::endl;
    // wait a short period for port 8080 to be free or owned by our previous child
    for (int i = 0; i < 10; ++i) {
        int listener = getListeningPid(8080);
        if (listener == -1) break; // no listener
        if (listener == serverPid) break; // our previous child still listening
        std::this_thread::sleep_for(std::chrono::seconds(1));
    }
    int listener = getListeningPid(8080);
    if (listener != -1 && listener != serverPid) {
        std::cerr << "[hot-reload] Port 8080 is used by PID " << listener << ". Not starting to avoid conflict." << std::endl;
    } else {
        serverPid = startServer(goRunCmd);
        std::cout << "[hot-reload] Server started with PID " << serverPid << std::endl;
    }

    const std::chrono::seconds pollInterval(2);

    while (!stopRequested) {
        // Reap if server exited unexpectedly and restart
        if (reapChildIfExited()) {
            std::cerr << "[hot-reload] Restarting server due to exit..." << std::endl;
            serverPid = startServer(goRunCmd);
            std::cout << "[hot-reload] Server restarted with PID " << serverPid << std::endl;
            // refresh file times
            fileTimes = getFileTimes(srcDir);
        }

        auto newFileTimes = getFileTimes(srcDir);
        if (filesChanged(fileTimes, newFileTimes)) {
            std::cout << "[hot-reload] Change detected. Restarting server..." << std::endl;
            if (serverPid > 0) {
                kill(serverPid, SIGTERM);
                // wait up to 5s for graceful shutdown
                for (int i = 0; i < 5; ++i) {
                    if (reapChildIfExited()) break;
                    std::this_thread::sleep_for(std::chrono::seconds(1));
                }
                if (serverPid > 0) {
                    std::cerr << "[hot-reload] Server did not exit gracefully, killing..." << std::endl;
                    kill(serverPid, SIGKILL);
                    waitForChild(serverPid);
                    serverPid = -1;
                }
            }
            // start new server
            serverPid = startServer(goRunCmd);
            // wait for port 8080 to be free or to be owned by us
            for (int i = 0; i < 10; ++i) {
                int listener2 = getListeningPid(8080);
                if (listener2 == -1) break;
                if (listener2 == serverPid) break;
                std::this_thread::sleep_for(std::chrono::seconds(1));
            }
            int listener2 = getListeningPid(8080);
            if (listener2 != -1 && listener2 != serverPid) {
                std::cerr << "[hot-reload] Port 8080 is used by PID " << listener2 << ". Not restarting to avoid conflict." << std::endl;
            } else {
                serverPid = startServer(goRunCmd);
                std::cout << "[hot-reload] Server started with PID " << serverPid << std::endl;
            }
            fileTimes = newFileTimes;
        }

        std::this_thread::sleep_for(pollInterval);
    }

    std::cout << "[hot-reload] Stop requested, shutting down..." << std::endl;
    if (serverPid > 0) {
        kill(serverPid, SIGTERM);
        waitForChild(serverPid);
    }
    return 0;
}

#include <iostream>
#include <vector>
#include <string>

#define MAX_SIZE 1024

using StringVec = std::vector<std::string>;

enum class LogLevel {
    Debug,
    Info,
    Error
};

struct Config {
    std::string host;
    int port;
};

class Logger {
public:
    Logger(const std::string& name) : name_(name) {}
    void log(LogLevel level, const std::string& msg) {
        std::cout << name_ << ": " << msg << std::endl;
    }
    const std::string& name() const { return name_; }
private:
    std::string name_;
};

namespace utils {
int helper(int x) {
    return x * 2;
}
}

template<typename T>
T clamp(T value, T low, T high) {
    if (value < low) return low;
    if (value > high) return high;
    return value;
}
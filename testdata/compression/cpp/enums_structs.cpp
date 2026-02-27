enum class Color : int {
    Red,
    Green,
    Blue
};

enum struct Direction {
    North,
    South,
    East,
    West
};

struct Point {
    double x;
    double y;
};

struct Config {
    std::string host;
    int port;
    bool debug;
};
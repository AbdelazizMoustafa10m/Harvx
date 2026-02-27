#include <string>
#include <vector>

using StringVec = std::vector<std::string>;

namespace mylib {
void init();
void shutdown();
int version();
}

namespace a::b::c {
int helper();
}

inline namespace v2 {
void process();
}
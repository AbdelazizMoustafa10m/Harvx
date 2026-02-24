#include <vector>
#include <string>

template<typename T>
class Stack {
public:
    void push(const T& value) {
        data_.push_back(value);
    }
    T pop() {
        T val = data_.back();
        data_.pop_back();
        return val;
    }
    bool empty() const { return data_.empty(); }
private:
    std::vector<T> data_;
};

class Animal {
public:
    virtual ~Animal() = default;
    virtual void speak() const = 0;
    std::string name() const { return name_; }
protected:
    std::string name_;
};

class Dog : public Animal {
public:
    void speak() const override {
        std::cout << "Woof!" << std::endl;
    }
private:
    int tricks_count;
};

template<typename T, typename U>
U convert(const T& input) {
    return static_cast<U>(input);
}
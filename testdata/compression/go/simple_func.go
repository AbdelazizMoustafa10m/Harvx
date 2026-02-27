package example

import "fmt"

// Greet returns a greeting message.
func Greet(name string) string {
	return fmt.Sprintf("Hello, %s!", name)
}

// add adds two integers.
func add(a, b int) int {
	return a + b
}

// Divide performs division with error handling.
func Divide(a, b float64) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return a / b, nil
}

// ReadAll reads everything and returns named results.
func ReadAll(path string) (data []byte, err error) {
	data, err = nil, nil
	return
}
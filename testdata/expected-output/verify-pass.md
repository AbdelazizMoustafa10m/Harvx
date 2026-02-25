# Harvx Context: test-project

| Field | Value |
|-------|-------|
| Generated | 2026-01-15T10:00:00Z |
| Content Hash | abc123 |
| Profile | default |
| Tokenizer | cl100k_base |
| Total Tokens | 150 |
| Total Files | 3 |

## File Summary

**Total Files:** 3 | **Total Tokens:** 150

## Directory Tree

```
test-project/
├── main.go
├── config.go
└── utils.go
```

## Files

### `main.go`

> **Size:** 64 B | **Tokens:** 50 | **Tier:** critical | **Compressed:** no

```go
package main

import "fmt"

func main() {
	fmt.Println("hello world")
}
```

### `config.go`

> **Size:** 48 B | **Tokens:** 40 | **Tier:** primary | **Compressed:** no

```go
package main

var defaultPort = 8080
```

### `utils.go`

> **Size:** 60 B | **Tokens:** 60 | **Tier:** supporting | **Compressed:** no

```go
package main

func add(a, b int) int {
	return a + b
}
```
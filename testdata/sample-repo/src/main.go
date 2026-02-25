package main

import (
	"fmt"
	"sample-repo/src/auth"
)

func main() {
	fmt.Println("Sample app")
	auth.CheckAuth()
}

package main

import "fmt"

import (
	"context"
	"net/http"
	"strings"

	"github.com/example/pkg"
	. "math"
	_ "net/http/pprof"
	alias "encoding/json"
)

func main() {
	fmt.Println("hello")
	ctx := context.Background()
	_ = ctx
	_ = http.StatusOK
	_ = strings.NewReader("")
	_ = pkg.New()
	_ = Sqrt(2)
	var v interface{}
	_ = alias.Marshal(v)
}
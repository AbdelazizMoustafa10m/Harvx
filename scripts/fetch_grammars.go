//go:build ignore

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	base := "https://unpkg.com/tree-sitter-wasms@0.1.13/out"
	fallback := "https://cdn.jsdelivr.net/npm/tree-sitter-wasms@0.1.13/out"

	grammars := []string{
		"tree-sitter-typescript.wasm",
		"tree-sitter-javascript.wasm",
		"tree-sitter-go.wasm",
		"tree-sitter-python.wasm",
		"tree-sitter-rust.wasm",
		"tree-sitter-java.wasm",
		"tree-sitter-c.wasm",
		"tree-sitter-cpp.wasm",
	}

	dir := filepath.Join("grammars")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir: %v\n", err)
		os.Exit(1)
	}

	for _, name := range grammars {
		dest := filepath.Join(dir, name)
		if _, err := os.Stat(dest); err == nil {
			fmt.Printf("SKIP  %s (exists)\n", name)
			continue
		}
		url := base + "/" + name
		fmt.Printf("FETCH %s ... ", name)
		err := download(url, dest)
		if err != nil {
			fmt.Printf("(trying fallback) ")
			url = fallback + "/" + name
			err = download(url, dest)
		}
		if err != nil {
			fmt.Printf("FAILED: %v\n", err)
			os.Exit(1)
		}
		info, _ := os.Stat(dest)
		fmt.Printf("OK (%d bytes)\n", info.Size())
	}
	fmt.Println("\nAll grammars downloaded.")
}

func download(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

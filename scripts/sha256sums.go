//go:build ignore

// Prints sha256sum-compatible lines ("<hex>  <basename>") for each file argument.
// Used by `make cross-build`; Go instead of sha256sum so it also runs from cmd.exe.
package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	for _, path := range os.Args[1:] {
		f, err := os.Open(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		h := sha256.New()
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		fmt.Printf("%x  %s\n", h.Sum(nil), filepath.Base(path))
	}
}

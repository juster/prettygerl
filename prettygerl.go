package main

import (
	"io"
	"fmt"
	"os"
)

func main() {
	buf, _ := io.ReadAll(os.Stdin)
	items := lexErlTerm("stdin", string(buf))
	if err := prettyErlTerm(items, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	os.Stdout.Sync()
}

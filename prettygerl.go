package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	rdr := bufio.NewReader(os.Stdin)
	items := lexErlTerm("stdin", rdr)
	if err := prettyErlTerm(items, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

package main

import (
	"github.com/kunit/rps/version"
	"os"

	"github.com/kunit/rps"
)

func main() {
	os.Exit(rps.RunCLI(rps.Env{
		Out:     os.Stdout,
		Err:     os.Stderr,
		Args:    os.Args[1:],
		Version: version.Version,
	}))
}

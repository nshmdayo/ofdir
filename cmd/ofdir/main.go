package main

import (
	"os"

	"github.com/nshmdayo/ofdir/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}

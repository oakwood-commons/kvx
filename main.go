package main

import (
	"fmt"
	"os"

	"github.com/oakwood-commons/kvx/cmd"
	"github.com/oakwood-commons/kvx/pkg/logger"
)

func main() {
	exitCode := 0
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		exitCode = 1
	}

	logger.Sync()
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

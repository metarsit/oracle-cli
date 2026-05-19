// cmd/oracle/main.go
package main

import (
	"fmt"
	"os"

	"github.com/metarsit/oracle-cli/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.NewRootCmd(version).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

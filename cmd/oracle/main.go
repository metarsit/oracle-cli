// cmd/oracle/main.go
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/metarsit/oracle-cli/internal/cli"
	"github.com/metarsit/oracle-cli/internal/client"
)

var version = "dev"

func main() {
	err := cli.NewRootCmd(version).Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	os.Exit(exitCode(err))
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var authErr *client.ErrAuth
	if errors.As(err, &authErr) {
		return 2
	}
	var nfErr *client.ErrNotFound
	if errors.As(err, &nfErr) {
		return 3
	}
	var netErr *client.ErrNetwork
	if errors.As(err, &netErr) {
		return 4
	}
	var degErr *client.ErrDegraded
	if errors.As(err, &degErr) {
		return 5
	}
	return 1
}

// cmd/oracle/main_test.go
package main

import (
	"errors"
	"testing"

	"github.com/metarsit/oracle-cli/internal/client"
)

func TestExitCodeMapping(t *testing.T) {
	cases := []struct {
		err  error
		want int
	}{
		{nil, 0},
		{&client.ErrAuth{Status: 401, Msg: ""}, 2},
		{&client.ErrNotFound{Msg: ""}, 3},
		{&client.ErrNetwork{Err: errors.New("x")}, 4},
		{&client.ErrDegraded{Status: 503, Msg: ""}, 5},
		{errors.New("anything"), 1},
	}
	for _, c := range cases {
		if got := exitCode(c.err); got != c.want {
			t.Errorf("exitCode(%v) = %d, want %d", c.err, got, c.want)
		}
	}
}

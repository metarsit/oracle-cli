// internal/client/errors.go
package client

import "fmt"

// ErrAuth is returned for HTTP 401/403.
type ErrAuth struct {
	Status int
	Msg    string
}

func (e *ErrAuth) Error() string { return fmt.Sprintf("auth failed (%d): %s", e.Status, e.Msg) }

// ErrNotFound is returned for HTTP 404.
type ErrNotFound struct{ Msg string }

func (e *ErrNotFound) Error() string { return "not found: " + e.Msg }

// ErrNetwork wraps transport-level failures (dial, EOF, timeout).
type ErrNetwork struct{ Err error }

func (e *ErrNetwork) Error() string { return "network: " + e.Err.Error() }
func (e *ErrNetwork) Unwrap() error { return e.Err }

// ErrDegraded is returned for HTTP 503 or non-ready /readyz payload.
type ErrDegraded struct {
	Status int
	Msg    string
}

func (e *ErrDegraded) Error() string { return fmt.Sprintf("degraded (%d): %s", e.Status, e.Msg) }

// ErrAPI is any other error envelope (4xx/5xx with .error).
type ErrAPI struct {
	Code, Msg string
	Status    int
}

func (e *ErrAPI) Error() string { return fmt.Sprintf("%s (%d): %s", e.Code, e.Status, e.Msg) }

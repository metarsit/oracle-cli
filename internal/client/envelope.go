// internal/client/envelope.go
package client

import (
	"encoding/json"
	"fmt"
)

// rawEnvelope mirrors the oracle's wire shape.
type rawEnvelope struct {
	Data  json.RawMessage `json:"data,omitempty"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
	Meta json.RawMessage `json:"meta,omitempty"`
}

// DecodeEnvelope parses body, classifies by status + envelope.error, and
// either unmarshals .data into out or returns a typed error.
func DecodeEnvelope(body []byte, status int, out any) error {
	var env rawEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("decode envelope: %w", err)
	}
	if env.Error != nil || status >= 400 {
		code, msg := "", ""
		if env.Error != nil {
			code, msg = env.Error.Code, env.Error.Message
		}
		switch {
		case status == 401 || status == 403:
			return &ErrAuth{Status: status, Msg: msg}
		case status == 404:
			return &ErrNotFound{Msg: msg}
		case status == 503:
			return &ErrDegraded{Status: status, Msg: msg}
		}
		return &ErrAPI{Code: code, Msg: msg, Status: status}
	}
	if len(env.Data) == 0 || string(env.Data) == "null" {
		return nil
	}
	if err := json.Unmarshal(env.Data, out); err != nil {
		return fmt.Errorf("decode data: %w", err)
	}
	return nil
}

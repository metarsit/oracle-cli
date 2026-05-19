// internal/format/json.go
package format

import (
	"encoding/json"
	"io"
	"time"
)

type jsonRenderer struct{}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Meta struct {
		TS string `json:"ts"`
	} `json:"meta"`
}

func (jsonRenderer) Render(w io.Writer, v any) error {
	env := struct {
		Data any `json:"data"`
		Meta struct {
			TS string `json:"ts"`
		} `json:"meta"`
	}{Data: v}
	env.Meta.TS = time.Now().UTC().Format(time.RFC3339Nano)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}

func (jsonRenderer) RenderError(w io.Writer, code, msg string) error {
	var env errorEnvelope
	env.Error.Code = code
	env.Error.Message = msg
	env.Meta.TS = time.Now().UTC().Format(time.RFC3339Nano)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(env)
}

package format

import (
	"io"
	"time"

	"gopkg.in/yaml.v3"
)

type yamlRenderer struct{}

func (yamlRenderer) Render(w io.Writer, v any) error {
	env := map[string]any{
		"data": v,
		"meta": map[string]string{"ts": time.Now().UTC().Format(time.RFC3339Nano)},
	}
	return yaml.NewEncoder(w).Encode(env)
}

func (yamlRenderer) RenderError(w io.Writer, code, msg string) error {
	env := map[string]any{
		"error": map[string]string{"code": code, "message": msg},
		"meta":  map[string]string{"ts": time.Now().UTC().Format(time.RFC3339Nano)},
	}
	return yaml.NewEncoder(w).Encode(env)
}

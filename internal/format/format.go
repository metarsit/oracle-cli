// internal/format/format.go
package format

import (
	"errors"
	"io"
)

// Renderer turns a typed value (or error) into the chosen output format.
type Renderer interface {
	Render(w io.Writer, v any) error
	RenderError(w io.Writer, code, msg string) error
}

// NewRenderer returns the renderer for the given format name. Unknown
// names fall back to JSON (callers should validate first; this is a safety net).
func NewRenderer(name string) Renderer {
	switch name {
	case "json":
		return jsonRenderer{}
	case "yaml":
		return yamlRenderer{}
	case "table":
		return tableRenderer{}
	}
	return jsonRenderer{}
}

// ErrUnknownFormat is returned by ValidateFormat for invalid names.
var ErrUnknownFormat = errors.New("unknown output format")

// ValidateFormat reports whether name is one of json/yaml/table.
func ValidateFormat(name string) error {
	switch name {
	case "json", "yaml", "table":
		return nil
	}
	return ErrUnknownFormat
}

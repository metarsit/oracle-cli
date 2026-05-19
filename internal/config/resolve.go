// internal/config/resolve.go
package config

import (
	"os"
	"time"
)

// Flags holds the parsed CLI flag values.
type Flags struct {
	BaseURL string
	Token   string
	Output  string
	Timeout string
}

// Inputs aggregates all four config sources.
type Inputs struct {
	Flag  Flags
	File  File
	Vault map[string]string
}

// Resolved is the merged, typed config the rest of the CLI consumes.
type Resolved struct {
	BaseURL             string
	Token               string
	DeribitBaseURL      string
	DeribitClientID     string
	DeribitClientSecret string
	Output              string
	Timeout             time.Duration
}

const (
	defaultBaseURL        = "http://localhost:8080"
	defaultDeribitBaseURL = "https://www.deribit.com/api/v2"
	defaultOutput         = "table"
	defaultTimeout        = 10 * time.Second
)

// Resolve applies the precedence rules: flag > env > vault > file > default.
func Resolve(in Inputs) Resolved {
	r := Resolved{}

	r.BaseURL = firstNonEmpty(in.Flag.BaseURL, os.Getenv("ORACLE_BASE_URL"), in.Vault["base_url"], in.File.BaseURL, defaultBaseURL)
	r.Token = firstNonEmpty(in.Flag.Token, os.Getenv("ORACLE_API_TOKEN"), in.Vault["oracle_api_token"])
	r.DeribitBaseURL = firstNonEmpty(os.Getenv("DERIBIT_BASE_URL"), in.Vault["deribit_base_url"], in.File.DeribitBaseURL, defaultDeribitBaseURL)
	r.DeribitClientID = firstNonEmpty(os.Getenv("DERIBIT_CLIENT_ID"), in.Vault["deribit_client_id"])
	r.DeribitClientSecret = firstNonEmpty(os.Getenv("DERIBIT_CLIENT_SECRET"), in.Vault["deribit_client_secret"])
	r.Output = firstNonEmpty(in.Flag.Output, os.Getenv("ORACLE_OUTPUT"), in.File.Output, defaultOutput)

	timeoutStr := firstNonEmpty(in.Flag.Timeout, os.Getenv("ORACLE_TIMEOUT"), in.File.Timeout)
	if d, err := time.ParseDuration(timeoutStr); err == nil && d > 0 {
		r.Timeout = d
	} else {
		r.Timeout = defaultTimeout
	}
	return r
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

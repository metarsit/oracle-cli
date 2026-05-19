// Package client provides an HTTP client for the oracle API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const maxResponseBytes = 10 * 1024 * 1024

// Client talks to the oracle HTTP API.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New builds a client with bearer token and timeout.
func New(baseURL, token string, timeout time.Duration) *Client {
	return &Client{
		baseURL: baseURL,
		token:   token,
		http:    &http.Client{Timeout: timeout},
	}
}

// Get issues GET path?query and decodes envelope into out.
func (c *Client) Get(ctx context.Context, path string, query map[string]string, out any) error {
	return c.do(ctx, http.MethodGet, path, query, nil, out, true)
}

// Post issues POST path with optional JSON body.
func (c *Client) Post(ctx context.Context, path string, body any, out any) error {
	return c.do(ctx, http.MethodPost, path, nil, body, out, false)
}

func (c *Client) do(ctx context.Context, method, path string, query map[string]string, body, out any, retry bool) error {
	u, err := buildURL(c.baseURL, path, query)
	if err != nil {
		return err
	}
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal body: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}
	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
		if err != nil {
			return fmt.Errorf("build request: %w", err)
		}
		if c.token != "" && path != "/healthz" && path != "/readyz" {
			req.Header.Set("Authorization", "Bearer "+c.token)
		}
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.http.Do(req)
		if err != nil {
			if retry && attempt == 0 {
				time.Sleep(250 * time.Millisecond)
				continue
			}
			return &ErrNetwork{Err: redactErr(err, c.token)}
		}
		respBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
		_ = resp.Body.Close()
		if readErr != nil {
			return &ErrNetwork{Err: redactErr(readErr, c.token)}
		}
		if len(respBody) > maxResponseBytes {
			return fmt.Errorf("response exceeded %d byte cap", maxResponseBytes)
		}
		return DecodeEnvelope(respBody, resp.StatusCode, out)
	}
	return errors.New("unreachable")
}

func buildURL(base, path string, query map[string]string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse base: %w", err)
	}
	u = u.JoinPath(path)
	if len(query) > 0 {
		q := u.Query()
		for k, v := range query {
			if v != "" {
				q.Set(k, v)
			}
		}
		u.RawQuery = q.Encode()
	}
	return u.String(), nil
}

func redactErr(err error, token string) error {
	if token == "" {
		return err
	}
	msg := err.Error()
	if !contains(msg, token) {
		return err
	}
	return errors.New(replace(msg, token, "REDACTED"))
}

func contains(s, sub string) bool { return len(sub) > 0 && bytes.Contains([]byte(s), []byte(sub)) }
func replace(s, old, repl string) string {
	return string(bytes.ReplaceAll([]byte(s), []byte(old), []byte(repl)))
}

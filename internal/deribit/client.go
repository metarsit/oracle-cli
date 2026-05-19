// internal/deribit/client.go
package deribit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// Client wraps the Deribit OAuth2 client_credentials flow.
type Client struct {
	baseURL  string
	clientID string
	secret   string
	http     *http.Client

	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

// New returns an OAuth2 client. Pass the API base URL (typically
// https://www.deribit.com/api/v2 or the testnet equivalent).
func New(baseURL, clientID, secret string, timeout time.Duration) *Client {
	return &Client{
		baseURL:  baseURL,
		clientID: clientID,
		secret:   secret,
		http:     &http.Client{Timeout: timeout},
	}
}

func (c *Client) authToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExp) {
		return c.token, nil
	}
	if c.clientID == "" || c.secret == "" {
		return "", errors.New("deribit: client_id and client_secret required")
	}
	q := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {c.clientID},
		"client_secret": {c.secret},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/public/auth?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("auth: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var parsed struct {
		Result struct {
			AccessToken string `json:"access_token"`
			ExpiresIn   int    `json:"expires_in"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", fmt.Errorf("auth parse: %w", err)
	}
	if parsed.Error != nil {
		return "", fmt.Errorf("deribit auth error %d: %s", parsed.Error.Code, parsed.Error.Message)
	}
	c.token = parsed.Result.AccessToken
	c.tokenExp = time.Now().Add(time.Duration(parsed.Result.ExpiresIn-30) * time.Second)
	return c.token, nil
}

func (c *Client) privateGet(ctx context.Context, path string, query url.Values, out any) error {
	tok, err := c.authToken(ctx)
	if err != nil {
		return err
	}
	u := c.baseURL + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("deribit get: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return fmt.Errorf("deribit auth failed: %s", string(body))
	}
	return json.Unmarshal(body, out)
}

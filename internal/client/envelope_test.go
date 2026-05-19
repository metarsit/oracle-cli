// internal/client/envelope_test.go
package client

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestDecodeData(t *testing.T) {
	body := []byte(`{"data":{"x":1},"meta":{"ts":"2026-01-01T00:00:00Z"}}`)
	var data struct{ X int }
	if err := DecodeEnvelope(body, 200, &data); err != nil {
		t.Fatal(err)
	}
	if data.X != 1 {
		t.Errorf("X = %d, want 1", data.X)
	}
}

func TestDecodeAPIError(t *testing.T) {
	body := []byte(`{"error":{"code":"BAD","message":"nope"},"meta":{"ts":"..."}}`)
	var data struct{}
	err := DecodeEnvelope(body, 400, &data)
	var apiErr *ErrAPI
	if !errors.As(err, &apiErr) {
		t.Fatalf("want ErrAPI, got %T %v", err, err)
	}
	if apiErr.Code != "BAD" || apiErr.Msg != "nope" || apiErr.Status != 400 {
		t.Errorf("apiErr = %+v", apiErr)
	}
}

func TestDecodeAuthError(t *testing.T) {
	body := []byte(`{"error":{"code":"AUTH","message":"x"},"meta":{}}`)
	var data struct{}
	err := DecodeEnvelope(body, 401, &data)
	var authErr *ErrAuth
	if !errors.As(err, &authErr) {
		t.Fatalf("want ErrAuth, got %T %v", err, err)
	}
}

func TestDecodeNotFound(t *testing.T) {
	body := []byte(`{"error":{"code":"NOT_FOUND","message":"x"},"meta":{}}`)
	var data struct{}
	err := DecodeEnvelope(body, 404, &data)
	var nfErr *ErrNotFound
	if !errors.As(err, &nfErr) {
		t.Fatalf("want ErrNotFound, got %T %v", err, err)
	}
}

func TestDecodeDegraded(t *testing.T) {
	body := []byte(`{"error":{"code":"DEGRADED","message":"warmup"},"meta":{}}`)
	var data struct{}
	err := DecodeEnvelope(body, 503, &data)
	var d *ErrDegraded
	if !errors.As(err, &d) {
		t.Fatalf("want ErrDegraded, got %T %v", err, err)
	}
}

func TestDecodeMalformed(t *testing.T) {
	var data struct{}
	if err := DecodeEnvelope([]byte("not-json"), 200, &data); err == nil {
		t.Fatal("expected json error")
	}
}

var _ = json.Marshal // keep import

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

func TestDecodeEmptyBody(t *testing.T) {
	// `{}` with 200 -> no error, out unchanged.
	type holder struct{ X int }
	out := holder{X: 42}
	if err := DecodeEnvelope([]byte(`{}`), 200, &out); err != nil {
		t.Fatalf("empty envelope should not error: %v", err)
	}
	if out.X != 42 {
		t.Errorf("out mutated: X = %d, want 42", out.X)
	}
}

func TestDecodeNullData(t *testing.T) {
	// {"data":null,"meta":{}} with 200 -> no error, out unchanged.
	type holder struct{ X int }
	out := holder{X: 7}
	body := []byte(`{"data":null,"meta":{}}`)
	if err := DecodeEnvelope(body, 200, &out); err != nil {
		t.Fatalf("null data should not error: %v", err)
	}
	if out.X != 7 {
		t.Errorf("out mutated: X = %d, want 7", out.X)
	}
}

func TestDecodeStatus200WithErrorEnvelope(t *testing.T) {
	// Status 200 but envelope has .error -> ErrAPI.
	body := []byte(`{"error":{"code":"WEIRD","message":"200 with error"},"meta":{}}`)
	var out struct{}
	err := DecodeEnvelope(body, 200, &out)
	var apiErr *ErrAPI
	if !errors.As(err, &apiErr) {
		t.Fatalf("want ErrAPI, got %T %v", err, err)
	}
	if apiErr.Code != "WEIRD" || apiErr.Status != 200 {
		t.Errorf("apiErr = %+v", apiErr)
	}
}

func TestDecodeStatus500NoEnvelope(t *testing.T) {
	// Status 500 with no .error -> ErrAPI with empty code.
	body := []byte(`{}`)
	var out struct{}
	err := DecodeEnvelope(body, 500, &out)
	var apiErr *ErrAPI
	if !errors.As(err, &apiErr) {
		t.Fatalf("want ErrAPI, got %T %v", err, err)
	}
	if apiErr.Code != "" || apiErr.Status != 500 {
		t.Errorf("apiErr = %+v, want empty code and 500", apiErr)
	}
}

func TestDecodeMalformedData(t *testing.T) {
	// 200 with .data of wrong shape (string into struct) -> decode-data error.
	body := []byte(`{"data":"not-a-struct","meta":{}}`)
	var out struct{ X int }
	err := DecodeEnvelope(body, 200, &out)
	if err == nil {
		t.Fatal("expected decode-data error")
	}
	if !json.Valid([]byte(`"not-a-struct"`)) {
		t.Fatal("seed sanity check failed")
	}
	if got := err.Error(); !contains(got, "decode data") {
		t.Errorf("error missing 'decode data' wrapper: %q", got)
	}
}

func TestDecodeStatus503NoEnvelope(t *testing.T) {
	// Status 503 with bare body still returns ErrDegraded (no error block needed).
	body := []byte(`{}`)
	var out struct{}
	err := DecodeEnvelope(body, 503, &out)
	var d *ErrDegraded
	if !errors.As(err, &d) {
		t.Fatalf("want ErrDegraded, got %T %v", err, err)
	}
}

// FuzzDecodeEnvelope ensures DecodeEnvelope never panics on arbitrary input.
// Run with `go test -fuzz=FuzzDecodeEnvelope` for extended runs; the default
// test mode only exercises the seed corpus, keeping CI fast and deterministic.
func FuzzDecodeEnvelope(f *testing.F) {
	seeds := [][]byte{
		[]byte(`{}`),
		[]byte(`{"data":{"x":1},"meta":{}}`),
		[]byte(`{"data":null,"meta":{}}`),
		[]byte(`{"error":{"code":"E","message":"m"},"meta":{}}`),
		[]byte(`{"data":"oops","meta":{}}`),
		[]byte(`not-json`),
		[]byte(``),
		[]byte(`{"data":{"nested":{"a":1}}}`),
	}
	for _, s := range seeds {
		for _, status := range []int{200, 400, 401, 404, 500, 503} {
			f.Add(s, status)
		}
	}
	f.Fuzz(func(t *testing.T, body []byte, status int) {
		var out struct{ X int }
		// Must not panic for any input — error is fine.
		_ = DecodeEnvelope(body, status, &out)
	})
}

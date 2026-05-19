// internal/vault/format_test.go
package vault

import (
	"bytes"
	"testing"
)

func TestHeaderRoundtrip(t *testing.T) {
	salt := bytes.Repeat([]byte{0xAB}, saltLen)
	nonce := bytes.Repeat([]byte{0xCD}, nonceLen)
	in := header{version: 1, kdfID: kdfArgon2id, salt: salt, nonce: nonce, ctLen: 42}

	raw := in.marshal()
	if len(raw) != headerLen {
		t.Fatalf("marshal len = %d, want %d", len(raw), headerLen)
	}
	out, err := parseHeader(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if out.version != in.version || out.kdfID != in.kdfID || out.ctLen != in.ctLen {
		t.Errorf("scalar mismatch: %+v vs %+v", out, in)
	}
	if !bytes.Equal(out.salt, in.salt) || !bytes.Equal(out.nonce, in.nonce) {
		t.Error("salt/nonce mismatch")
	}
}

func TestParseHeaderRejectsBadMagic(t *testing.T) {
	bad := make([]byte, headerLen)
	copy(bad, []byte("XXXX"))
	if _, err := parseHeader(bad); err == nil {
		t.Fatal("expected magic error")
	}
}

func TestParseHeaderRejectsUnknownVersion(t *testing.T) {
	raw := header{version: 99, kdfID: kdfArgon2id, salt: make([]byte, saltLen), nonce: make([]byte, nonceLen)}.marshal()
	if _, err := parseHeader(raw); err == nil {
		t.Fatal("expected version error")
	}
}

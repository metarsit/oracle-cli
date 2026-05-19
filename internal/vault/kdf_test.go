// internal/vault/kdf_test.go
package vault

import (
	"bytes"
	"testing"
)

func TestDeriveKeyDeterministic(t *testing.T) {
	salt := bytes.Repeat([]byte{1}, saltLen)
	k1 := deriveKey([]byte("hunter2"), salt)
	k2 := deriveKey([]byte("hunter2"), salt)
	if !bytes.Equal(k1, k2) {
		t.Error("derive must be deterministic for same input")
	}
	if len(k1) != keyLen {
		t.Errorf("key len = %d, want %d", len(k1), keyLen)
	}
}

func TestDeriveKeyDifferentSaltDifferentKey(t *testing.T) {
	k1 := deriveKey([]byte("pw"), bytes.Repeat([]byte{1}, saltLen))
	k2 := deriveKey([]byte("pw"), bytes.Repeat([]byte{2}, saltLen))
	if bytes.Equal(k1, k2) {
		t.Error("different salts must produce different keys")
	}
}

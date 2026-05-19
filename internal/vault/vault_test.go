// internal/vault/vault_test.go
package vault

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func newPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "secrets.vault")
}

func TestCreateOpenRoundtrip(t *testing.T) {
	p := newPath(t)
	pw := []byte("correct horse battery staple")

	v := NewEmpty()
	v.Set("oracle_api_token", "tok-123")
	v.Set("deribit_client_id", "cid")
	if err := Save(p, v, pw); err != nil {
		t.Fatalf("save: %v", err)
	}

	got, err := Open(p, pw)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if v, ok := got.Get("oracle_api_token"); !ok || v != "tok-123" {
		t.Errorf("token = %q,%v", v, ok)
	}
	if v, ok := got.Get("deribit_client_id"); !ok || v != "cid" {
		t.Errorf("cid = %q,%v", v, ok)
	}
}

func TestOpenWrongPassphrase(t *testing.T) {
	p := newPath(t)
	if err := Save(p, NewEmpty(), []byte("right")); err != nil {
		t.Fatal(err)
	}
	_, err := Open(p, []byte("wrong"))
	if !errors.Is(err, ErrAuthFailed) {
		t.Errorf("want ErrAuthFailed, got %v", err)
	}
}

func TestOpenCorruptCiphertext(t *testing.T) {
	p := newPath(t)
	if err := Save(p, NewEmpty(), []byte("pw")); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(p)
	b[len(b)-1] ^= 0xFF // flip last byte
	_ = os.WriteFile(p, b, 0o600)
	_, err := Open(p, []byte("pw"))
	if !errors.Is(err, ErrAuthFailed) {
		t.Errorf("want ErrAuthFailed for corrupt vault, got %v", err)
	}
}

func TestOpenRejectsLoosePerms(t *testing.T) {
	p := newPath(t)
	if err := Save(p, NewEmpty(), []byte("pw")); err != nil {
		t.Fatal(err)
	}
	_ = os.Chmod(p, 0o644)
	_, err := Open(p, []byte("pw"))
	if !errors.Is(err, ErrInsecurePerm) {
		t.Errorf("want ErrInsecurePerm, got %v", err)
	}
}

func TestRotateChangesPassphrase(t *testing.T) {
	p := newPath(t)
	v := NewEmpty()
	v.Set("k", "v")
	if err := Save(p, v, []byte("old")); err != nil {
		t.Fatal(err)
	}
	if err := Rotate(p, []byte("old"), []byte("new")); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(p, []byte("old")); !errors.Is(err, ErrAuthFailed) {
		t.Errorf("old should now fail: %v", err)
	}
	got, err := Open(p, []byte("new"))
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := got.Get("k"); v != "v" {
		t.Errorf("data lost across rotate")
	}
}

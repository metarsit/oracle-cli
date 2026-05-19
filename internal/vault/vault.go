// Package vault implements AES-256-GCM encrypted secret storage.
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	toml "github.com/pelletier/go-toml/v2"
)

// Vault is an in-memory, mutable view of a decrypted secrets file.
type Vault struct {
	Secrets map[string]string `toml:"secrets"`
}

// Errors callers branch on.
var (
	ErrAuthFailed   = errors.New("vault: wrong passphrase or corrupt vault")
	ErrInsecurePerm = errors.New("vault: file or parent dir has insecure permissions")
)

// NewEmpty returns a zero-secret vault.
func NewEmpty() *Vault { return &Vault{Secrets: map[string]string{}} }

// Get returns (value, true) when key is present.
func (v *Vault) Get(k string) (string, bool) { val, ok := v.Secrets[k]; return val, ok }

// Set inserts or overwrites a key.
func (v *Vault) Set(k, val string) { v.Secrets[k] = val }

// Delete removes a key (no-op if absent).
func (v *Vault) Delete(k string) { delete(v.Secrets, k) }

// Keys returns a sorted slice of currently held keys.
func (v *Vault) Keys() []string {
	out := make([]string, 0, len(v.Secrets))
	for k := range v.Secrets {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Save serialises v, derives a key from passphrase, encrypts, and writes path with mode 0600.
// A fresh nonce is generated each call. Salt is regenerated for new vaults; existing
// vaults' salts are reused by Rotate via saveWithSalt below.
func Save(path string, v *Vault, passphrase []byte) error {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return fmt.Errorf("rand salt: %w", err)
	}
	return saveWithSalt(path, v, passphrase, salt)
}

func saveWithSalt(path string, v *Vault, passphrase, salt []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	nonce := make([]byte, nonceLen)
	if _, err := rand.Read(nonce); err != nil {
		return fmt.Errorf("rand nonce: %w", err)
	}
	pt, err := toml.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal vault: %w", err)
	}
	key := deriveKey(passphrase, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("gcm: %w", err)
	}
	hdr := header{version: currentVer, kdfID: kdfArgon2id, salt: salt, nonce: nonce, ctLen: uint32(len(pt) + gcm.Overhead())} //nolint:gosec // ct length bounded by RAM
	aad := hdr.marshal()
	ct := gcm.Seal(nil, nonce, pt, aad)
	hdr.ctLen = uint32(len(ct)) //nolint:gosec
	out := append(hdr.marshal(), ct...)
	return os.WriteFile(path, out, 0o600)
}

// Open reads, validates perms, decrypts, and returns the vault.
func Open(path string, passphrase []byte) (*Vault, error) {
	if err := checkPerms(path); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path) //nolint:gosec // user-controlled path
	if err != nil {
		return nil, fmt.Errorf("read vault: %w", err)
	}
	if len(raw) < headerLen {
		return nil, fmt.Errorf("vault: truncated file")
	}
	hdr, err := parseHeader(raw[:headerLen])
	if err != nil {
		return nil, err
	}
	ct := raw[headerLen:]
	if uint32(len(ct)) != hdr.ctLen { //nolint:gosec
		return nil, fmt.Errorf("vault: ct length mismatch")
	}
	key := deriveKey(passphrase, hdr.salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm: %w", err)
	}
	pt, err := gcm.Open(nil, hdr.nonce, ct, raw[:headerLen])
	if err != nil {
		return nil, ErrAuthFailed
	}
	v := NewEmpty()
	if err := toml.Unmarshal(pt, v); err != nil {
		return nil, fmt.Errorf("unmarshal vault: %w", err)
	}
	if v.Secrets == nil {
		v.Secrets = map[string]string{}
	}
	return v, nil
}

// Rotate re-encrypts the vault with newPass and a fresh salt.
func Rotate(path string, oldPass, newPass []byte) error {
	v, err := Open(path, oldPass)
	if err != nil {
		return err
	}
	return Save(path, v, newPass)
}

func checkPerms(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat vault: %w", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("%w: file mode %o, want 0600", ErrInsecurePerm, info.Mode().Perm())
	}
	parent, err := os.Stat(filepath.Dir(path))
	if err == nil && parent.Mode().Perm()&0o002 != 0 {
		return fmt.Errorf("%w: parent dir is world-writable", ErrInsecurePerm)
	}
	return nil
}

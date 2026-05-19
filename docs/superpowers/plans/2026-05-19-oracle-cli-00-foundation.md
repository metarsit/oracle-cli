# oracle-cli Phase 00 — Foundation, Config, Vault

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Establish repo scaffold, layered config resolution, and AES-256-GCM encrypted secret vault — the substrate every other subcommand will depend on.

**Architecture:** Go 1.23 single-binary CLI with cobra+viper. Config resolved flag→env→vault→file→default. Vault is a binary file with AES-256-GCM ciphertext, argon2id KDF, mode-0600 enforced.

**Tech Stack:** Go 1.23, cobra, viper, pelletier/go-toml/v2, golang.org/x/crypto/argon2, golang.org/x/term, log/slog.

**Spec:** `docs/superpowers/specs/2026-05-19-oracle-cli-design.md`

---

## Task 1: Initialise Go module and Makefile

**Files:**
- Create: `go.mod`
- Create: `Makefile`
- Create: `.gitignore`
- Create: `.golangci.yml`

- [ ] **Step 1: Init module**

```bash
cd /Users/metarsit/repo/metarsit/oracle-cli
go mod init github.com/metarsit/oracle-cli
```

- [ ] **Step 2: Write Makefile**

```makefile
.PHONY: build test lint cover fmt vet vuln release clean

BIN     := oracle
PKG     := ./cmd/oracle
VERSION := $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BIN) $(PKG)

test:
	go test ./...

cover:
	go test -coverprofile=cover.out ./...
	go tool cover -func cover.out | tail -1

fmt:
	gofmt -l -w .

vet:
	go vet ./...

lint:
	golangci-lint run

vuln:
	govulncheck ./...

clean:
	rm -rf bin cover.out dist

release:
	mkdir -p dist
	for os_arch in darwin/arm64 darwin/amd64 linux/amd64; do \
		os=$${os_arch%/*}; arch=$${os_arch#*/}; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -ldflags "$(LDFLAGS)" \
			-o dist/oracle-cli_$(VERSION)_$${os}_$${arch} $(PKG); \
	done
	cd dist && shasum -a 256 oracle-cli_* > SHA256SUMS
```

- [ ] **Step 3: Write .gitignore**

```
bin/
dist/
cover.out
*.vault
.env
.idea/
.vscode/
```

- [ ] **Step 4: Write .golangci.yml**

```yaml
version: "2"
run:
  timeout: 5m
linters:
  default: standard
  enable:
    - errcheck
    - gosec
    - staticcheck
    - revive
    - gocritic
    - bodyclose
    - ineffassign
    - unconvert
  disable:
    - gochecknoglobals
linters-settings:
  gosec:
    excludes:
      - G104  # already covered by errcheck
```

- [ ] **Step 5: Commit**

```bash
git add go.mod Makefile .gitignore .golangci.yml
git commit -m "chore: scaffold go module, makefile, lint config"
```

---

## Task 2: Add cobra root command + version subcommand

**Files:**
- Create: `cmd/oracle/main.go`
- Create: `internal/cli/root.go`
- Create: `internal/cli/version.go`
- Create: `internal/cli/version_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/cli/version_test.go
package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand(t *testing.T) {
	root := NewRootCmd("test-version")
	var out bytes.Buffer
	root.SetOut(&out)
	root.SetArgs([]string{"version"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.Contains(out.String(), "test-version") {
		t.Errorf("got %q, want substring 'test-version'", out.String())
	}
}
```

- [ ] **Step 2: Add cobra dep + run test**

```bash
go get github.com/spf13/cobra@latest
go test ./internal/cli/... -run TestVersionCommand
```

Expected: FAIL (package compile error — NewRootCmd not defined).

- [ ] **Step 3: Write root.go**

```go
// internal/cli/root.go
package cli

import "github.com/spf13/cobra"

// NewRootCmd builds the oracle command tree. version is injected from main.
func NewRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "oracle",
		Short:         "Deribit Oracle CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(newVersionCmd(version))
	return cmd
}
```

- [ ] **Step 4: Write version.go**

```go
// internal/cli/version.go
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVersionCmd(version string) *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print oracle-cli version",
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), version)
			return err
		},
	}
}
```

- [ ] **Step 5: Write main.go**

```go
// cmd/oracle/main.go
package main

import (
	"fmt"
	"os"

	"github.com/metarsit/oracle-cli/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.NewRootCmd(version).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 6: Verify test passes + binary builds**

```bash
go test ./internal/cli/... -run TestVersionCommand
make build
./bin/oracle version
```

Expected: PASS, prints `dev`.

- [ ] **Step 7: Commit**

```bash
git add cmd internal/cli/root.go internal/cli/version.go internal/cli/version_test.go go.mod go.sum
git commit -m "feat(cli): root command with version subcommand"
```

---

## Task 3: XDG-aware path resolver

**Files:**
- Create: `internal/config/paths.go`
- Create: `internal/config/paths_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/config/paths_test.go
package config

import (
	"path/filepath"
	"testing"
)

func TestConfigPath_XDGHomeOverridesHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	t.Setenv("HOME", "/home/u")
	got := ConfigPath()
	want := "/tmp/xdg/oracle-cli/config.toml"
	if got != want {
		t.Errorf("ConfigPath = %q, want %q", got, want)
	}
}

func TestConfigPath_DefaultsToHomeDotConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/u")
	got := ConfigPath()
	want := filepath.Join("/home/u", ".config", "oracle-cli", "config.toml")
	if got != want {
		t.Errorf("ConfigPath = %q, want %q", got, want)
	}
}

func TestVaultPath_DefaultsToHomeDotConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "/home/u")
	got := VaultPath()
	want := filepath.Join("/home/u", ".config", "oracle-cli", "secrets.vault")
	if got != want {
		t.Errorf("VaultPath = %q, want %q", got, want)
	}
}
```

- [ ] **Step 2: Run test**

```bash
go test ./internal/config/... -run TestConfigPath -v
```

Expected: FAIL (undefined).

- [ ] **Step 3: Implement**

```go
// internal/config/paths.go
package config

import (
	"os"
	"path/filepath"
)

const (
	appDir         = "oracle-cli"
	configFile     = "config.toml"
	vaultFile      = "secrets.vault"
	envXDGConfig   = "XDG_CONFIG_HOME"
	envHome        = "HOME"
)

// configDir returns the directory where config and vault files live.
func configDir() string {
	if xdg := os.Getenv(envXDGConfig); xdg != "" {
		return filepath.Join(xdg, appDir)
	}
	return filepath.Join(os.Getenv(envHome), ".config", appDir)
}

// ConfigPath returns the absolute path of config.toml.
func ConfigPath() string { return filepath.Join(configDir(), configFile) }

// VaultPath returns the absolute path of secrets.vault.
func VaultPath() string { return filepath.Join(configDir(), vaultFile) }

// EnsureConfigDir creates the config directory with mode 0700.
func EnsureConfigDir() error {
	return os.MkdirAll(configDir(), 0o700)
}
```

- [ ] **Step 4: Verify + commit**

```bash
go test ./internal/config/... -v
git add internal/config/paths.go internal/config/paths_test.go
git commit -m "feat(config): XDG-aware path resolver"
```

---

## Task 4: Config file load/save (TOML, non-secret keys only)

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

- [ ] **Step 1: Add toml dep + write failing test**

```bash
go get github.com/pelletier/go-toml/v2@latest
```

```go
// internal/config/config_test.go
package config

import (
	"path/filepath"
	"testing"
)

func TestSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	in := File{
		BaseURL:        "https://oracle.example",
		DeribitBaseURL: "https://www.deribit.com/api/v2",
		Output:         "json",
		Timeout:        "15s",
	}
	if err := SaveFile(path, in); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := LoadFile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got != in {
		t.Errorf("roundtrip mismatch: got %+v want %+v", got, in)
	}
}

func TestLoadFileMissingReturnsZeroValue(t *testing.T) {
	got, err := LoadFile(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("load missing: %v", err)
	}
	if (got != File{}) {
		t.Errorf("expected zero File, got %+v", got)
	}
}

func TestSaveFileRejectsSecretKeys(t *testing.T) {
	dir := t.TempDir()
	in := File{BaseURL: "x", OracleAPIToken: "leak"}
	err := SaveFile(filepath.Join(dir, "c.toml"), in)
	if err == nil {
		t.Fatal("expected error rejecting secret in config file")
	}
}
```

- [ ] **Step 2: Run test (expect compile fail)**

```bash
go test ./internal/config/...
```

- [ ] **Step 3: Implement config.go**

```go
// internal/config/config.go
package config

import (
	"errors"
	"fmt"
	"os"

	toml "github.com/pelletier/go-toml/v2"
)

// File is the on-disk config representation. Non-secret keys only.
// OracleAPIToken / DeribitClientID / DeribitClientSecret never persist here;
// they live in the vault. The struct tag-less field exists to detect accidental
// writes and reject them in SaveFile.
type File struct {
	BaseURL        string `toml:"base_url,omitempty"`
	DeribitBaseURL string `toml:"deribit_base_url,omitempty"`
	Output         string `toml:"output,omitempty"`
	Timeout        string `toml:"timeout,omitempty"`

	// Guard fields — must remain empty in persisted state.
	OracleAPIToken      string `toml:"-"`
	DeribitClientID     string `toml:"-"`
	DeribitClientSecret string `toml:"-"`
}

// ErrSecretInConfig is returned when SaveFile is asked to persist a secret.
var ErrSecretInConfig = errors.New("config file refuses to store secrets; use vault")

// LoadFile reads path; missing file returns zero-value File and nil error.
func LoadFile(path string) (File, error) {
	b, err := os.ReadFile(path) //nolint:gosec // path is config-controlled
	if errors.Is(err, os.ErrNotExist) {
		return File{}, nil
	}
	if err != nil {
		return File{}, fmt.Errorf("read config: %w", err)
	}
	var f File
	if err := toml.Unmarshal(b, &f); err != nil {
		return File{}, fmt.Errorf("parse config: %w", err)
	}
	return f, nil
}

// SaveFile writes f to path with mode 0600. Refuses to write secret-bearing fields.
func SaveFile(path string, f File) error {
	if f.OracleAPIToken != "" || f.DeribitClientID != "" || f.DeribitClientSecret != "" {
		return ErrSecretInConfig
	}
	b, err := toml.Marshal(f)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, b, 0o600)
}
```

- [ ] **Step 4: Verify + commit**

```bash
go test ./internal/config/... -v
git add internal/config go.mod go.sum
git commit -m "feat(config): TOML load/save with secret-key rejection"
```

---

## Task 5: Vault file format header

**Files:**
- Create: `internal/vault/format.go`
- Create: `internal/vault/format_test.go`

- [ ] **Step 1: Write failing test**

```go
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
```

- [ ] **Step 2: Run (expect compile fail)**

```bash
go test ./internal/vault/...
```

- [ ] **Step 3: Implement format.go**

```go
// internal/vault/format.go
package vault

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	saltLen    = 16
	nonceLen   = 12
	magicLen   = 4
	headerLen  = magicLen + 1 + 1 + saltLen + nonceLen + 4 // 38
	currentVer = 1
	kdfArgon2id = 1
)

var magicBytes = []byte("OCLI")

// header is the unencrypted vault header. The marshalled form also serves
// as the AAD for AES-GCM, binding ciphertext to header bytes.
type header struct {
	version uint8
	kdfID   uint8
	salt    []byte // saltLen
	nonce   []byte // nonceLen
	ctLen   uint32
}

func (h header) marshal() []byte {
	out := make([]byte, headerLen)
	copy(out[0:magicLen], magicBytes)
	out[4] = h.version
	out[5] = h.kdfID
	copy(out[6:6+saltLen], h.salt)
	copy(out[6+saltLen:6+saltLen+nonceLen], h.nonce)
	binary.BigEndian.PutUint32(out[6+saltLen+nonceLen:], h.ctLen)
	return out
}

// parseHeader validates the prefix and returns the header. The slices it
// returns share memory with raw — callers must not mutate them.
func parseHeader(raw []byte) (header, error) {
	if len(raw) < headerLen {
		return header{}, fmt.Errorf("vault: truncated header (%d bytes)", len(raw))
	}
	if string(raw[0:magicLen]) != string(magicBytes) {
		return header{}, errors.New("vault: not an oracle-cli vault (bad magic)")
	}
	ver := raw[4]
	if ver != currentVer {
		return header{}, fmt.Errorf("vault: version %d, this CLI supports %d; upgrade", ver, currentVer)
	}
	kdf := raw[5]
	if kdf != kdfArgon2id {
		return header{}, fmt.Errorf("vault: unknown kdf id %d", kdf)
	}
	return header{
		version: ver,
		kdfID:   kdf,
		salt:    raw[6 : 6+saltLen],
		nonce:   raw[6+saltLen : 6+saltLen+nonceLen],
		ctLen:   binary.BigEndian.Uint32(raw[6+saltLen+nonceLen:]),
	}, nil
}
```

- [ ] **Step 4: Verify + commit**

```bash
go test ./internal/vault/... -v
git add internal/vault/format.go internal/vault/format_test.go
git commit -m "feat(vault): binary header marshal/parse"
```

---

## Task 6: argon2id KDF wrapper

**Files:**
- Create: `internal/vault/kdf.go`
- Create: `internal/vault/kdf_test.go`

- [ ] **Step 1: Add crypto dep + write failing test**

```bash
go get golang.org/x/crypto/argon2@latest
```

```go
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
```

- [ ] **Step 2: Run (expect compile fail)**

```bash
go test ./internal/vault/... -run TestDeriveKey
```

- [ ] **Step 3: Implement kdf.go**

```go
// internal/vault/kdf.go
package vault

import "golang.org/x/crypto/argon2"

const (
	keyLen       = 32
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64 MiB
	argonThreads = 4
)

// deriveKey runs argon2id with calibrated parameters (~250ms on M-series).
// Returns a 32-byte key suitable for AES-256-GCM.
func deriveKey(passphrase, salt []byte) []byte {
	return argon2.IDKey(passphrase, salt, argonTime, argonMemory, argonThreads, keyLen)
}
```

- [ ] **Step 4: Verify + commit**

```bash
go test ./internal/vault/... -v -timeout 60s
git add internal/vault/kdf.go internal/vault/kdf_test.go go.mod go.sum
git commit -m "feat(vault): argon2id key derivation"
```

---

## Task 7: Vault open / save (AES-256-GCM)

**Files:**
- Create: `internal/vault/vault.go`
- Create: `internal/vault/vault_test.go`

- [ ] **Step 1: Write failing test**

```go
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
```

- [ ] **Step 2: Run (expect compile fail)**

```bash
go test ./internal/vault/... -run TestCreateOpenRoundtrip
```

- [ ] **Step 3: Implement vault.go**

```go
// internal/vault/vault.go
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
```

- [ ] **Step 4: Verify + commit**

```bash
go test ./internal/vault/... -v -timeout 60s
git add internal/vault/vault.go internal/vault/vault_test.go
git commit -m "feat(vault): AES-256-GCM encrypted secret store with rotate"
```

---

## Task 8: Passphrase source (env → TTY prompt → error)

**Files:**
- Create: `internal/vault/passphrase.go`
- Create: `internal/vault/passphrase_test.go`

- [ ] **Step 1: Add term dep + write failing test**

```bash
go get golang.org/x/term@latest
```

```go
// internal/vault/passphrase_test.go
package vault

import (
	"errors"
	"testing"
)

func TestPassphraseFromEnv(t *testing.T) {
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "hunter2")
	got, err := readPassphrase(nopPrompter{}, false)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "hunter2" {
		t.Errorf("got %q, want hunter2", got)
	}
}

func TestPassphraseMissingNoTTY(t *testing.T) {
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "")
	_, err := readPassphrase(nopPrompter{}, false)
	if !errors.Is(err, ErrPassphraseUnavailable) {
		t.Errorf("want ErrPassphraseUnavailable, got %v", err)
	}
}

func TestPassphraseFromPrompter(t *testing.T) {
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "")
	got, err := readPassphrase(staticPrompter("typed-it"), true)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "typed-it" {
		t.Errorf("got %q", got)
	}
}

type nopPrompter struct{}

func (nopPrompter) Prompt(string) ([]byte, error) { return nil, errors.New("should not be called") }

type staticPrompter string

func (s staticPrompter) Prompt(string) ([]byte, error) { return []byte(s), nil }
```

- [ ] **Step 2: Implement passphrase.go**

```go
// internal/vault/passphrase.go
package vault

import (
	"errors"
	"fmt"
	"os"

	"golang.org/x/term"
)

const envPassphrase = "ORACLE_VAULT_PASSPHRASE"

// ErrPassphraseUnavailable is returned when no passphrase source is reachable.
var ErrPassphraseUnavailable = errors.New("vault: no passphrase available; set ORACLE_VAULT_PASSPHRASE or run interactively")

// Prompter abstracts TTY interaction for testability.
type Prompter interface {
	Prompt(label string) ([]byte, error)
}

// TermPrompter reads from /dev/tty with echo off.
type TermPrompter struct{}

func (TermPrompter) Prompt(label string) ([]byte, error) {
	fmt.Fprint(os.Stderr, label)
	pw, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr)
	return pw, err
}

// ReadPassphrase is the public helper used by cli code.
func ReadPassphrase() ([]byte, error) {
	isTTY := term.IsTerminal(int(os.Stdin.Fd()))
	return readPassphrase(TermPrompter{}, isTTY)
}

func readPassphrase(p Prompter, isTTY bool) ([]byte, error) {
	if v := os.Getenv(envPassphrase); v != "" {
		return []byte(v), nil
	}
	if !isTTY {
		return nil, ErrPassphraseUnavailable
	}
	return p.Prompt("Vault passphrase: ")
}
```

- [ ] **Step 3: Verify + commit**

```bash
go test ./internal/vault/... -v
git add internal/vault/passphrase.go internal/vault/passphrase_test.go go.mod go.sum
git commit -m "feat(vault): passphrase resolver env→TTY"
```

---

## Task 9: `oracle vault` subcommand group

**Files:**
- Create: `internal/cli/vault.go`
- Create: `internal/cli/vault_test.go`
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Write failing test**

```go
// internal/cli/vault_test.go
package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestVaultInitSetGetListRm(t *testing.T) {
	dir := t.TempDir()
	vaultPath := filepath.Join(dir, "secrets.vault")
	t.Setenv("ORACLE_VAULT", vaultPath)
	t.Setenv("ORACLE_VAULT_PASSPHRASE", "hunter2")
	t.Setenv("XDG_CONFIG_HOME", dir)

	run := func(args ...string) (string, string, error) {
		root := NewRootCmd("test")
		var out, errBuf bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&errBuf)
		root.SetArgs(args)
		err := root.Execute()
		return out.String(), errBuf.String(), err
	}

	if _, _, err := run("vault", "init"); err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := os.Stat(vaultPath); err != nil {
		t.Fatalf("vault file not created: %v", err)
	}
	if _, _, err := run("vault", "set", "oracle_api_token", "tok-xyz"); err != nil {
		t.Fatalf("set: %v", err)
	}
	out, _, err := run("vault", "get", "oracle_api_token")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if strings.TrimSpace(out) != "tok-xyz" {
		t.Errorf("got %q want tok-xyz", out)
	}
	out, _, err = run("vault", "list")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if !strings.Contains(out, "oracle_api_token") {
		t.Errorf("list missing key: %q", out)
	}
	if _, _, err := run("vault", "rm", "oracle_api_token"); err != nil {
		t.Fatalf("rm: %v", err)
	}
	out, _, _ = run("vault", "list")
	if strings.Contains(out, "oracle_api_token") {
		t.Errorf("rm failed: %q", out)
	}
}
```

- [ ] **Step 2: Implement vault.go**

```go
// internal/cli/vault.go
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/metarsit/oracle-cli/internal/config"
	"github.com/metarsit/oracle-cli/internal/vault"
	"github.com/spf13/cobra"
)

// vaultPath resolves --vault flag → ORACLE_VAULT env → default XDG path.
func vaultPath(cmd *cobra.Command) string {
	if v, _ := cmd.Flags().GetString("vault"); v != "" {
		return v
	}
	if v := os.Getenv("ORACLE_VAULT"); v != "" {
		return v
	}
	return config.VaultPath()
}

func newVaultCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vault",
		Short: "Manage the encrypted secret store",
	}
	cmd.PersistentFlags().String("vault", "", "Path to vault file (default $XDG_CONFIG_HOME/oracle-cli/secrets.vault)")
	cmd.AddCommand(
		newVaultInitCmd(),
		newVaultSetCmd(),
		newVaultGetCmd(),
		newVaultListCmd(),
		newVaultRmCmd(),
		newVaultRotateCmd(),
		newVaultExportCmd(),
	)
	return cmd
}

func newVaultInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Create an empty vault",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := vaultPath(cmd)
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("vault already exists at %s", path)
			}
			pw, err := vault.ReadPassphrase()
			if err != nil {
				return err
			}
			if err := config.EnsureConfigDir(); err != nil {
				return err
			}
			return vault.Save(path, vault.NewEmpty(), pw)
		},
	}
}

func newVaultSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a secret",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := vaultPath(cmd)
			pw, err := vault.ReadPassphrase()
			if err != nil {
				return err
			}
			v, err := vault.Open(path, pw)
			if err != nil {
				return err
			}
			v.Set(args[0], args[1])
			return vault.Save(path, v, pw)
		},
	}
}

func newVaultGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Print a secret to stdout",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := vaultPath(cmd)
			pw, err := vault.ReadPassphrase()
			if err != nil {
				return err
			}
			v, err := vault.Open(path, pw)
			if err != nil {
				return err
			}
			val, ok := v.Get(args[0])
			if !ok {
				return fmt.Errorf("key %q not in vault", args[0])
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), val)
			return err
		},
	}
}

func newVaultListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List secret keys (values never shown)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := vaultPath(cmd)
			pw, err := vault.ReadPassphrase()
			if err != nil {
				return err
			}
			v, err := vault.Open(path, pw)
			if err != nil {
				return err
			}
			for _, k := range v.Keys() {
				fmt.Fprintln(cmd.OutOrStdout(), k)
			}
			return nil
		},
	}
}

func newVaultRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <key>",
		Short: "Delete a secret",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := vaultPath(cmd)
			pw, err := vault.ReadPassphrase()
			if err != nil {
				return err
			}
			v, err := vault.Open(path, pw)
			if err != nil {
				return err
			}
			v.Delete(args[0])
			return vault.Save(path, v, pw)
		},
	}
}

func newVaultRotateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rotate",
		Short: "Rotate vault passphrase (uses TTY prompts for old + new)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := vaultPath(cmd)
			oldPw, err := vault.TermPrompter{}.Prompt("Current passphrase: ")
			if err != nil {
				return err
			}
			newPw, err := vault.TermPrompter{}.Prompt("New passphrase: ")
			if err != nil {
				return err
			}
			if len(newPw) == 0 {
				return errors.New("new passphrase empty")
			}
			return vault.Rotate(path, oldPw, newPw)
		},
	}
}

func newVaultExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Print decrypted vault to stdout (requires --confirm)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			confirm, _ := cmd.Flags().GetBool("confirm")
			if !confirm {
				return errors.New("refusing to export without --confirm")
			}
			path := vaultPath(cmd)
			pw, err := vault.ReadPassphrase()
			if err != nil {
				return err
			}
			v, err := vault.Open(path, pw)
			if err != nil {
				return err
			}
			for _, k := range v.Keys() {
				val, _ := v.Get(k)
				fmt.Fprintf(cmd.OutOrStdout(), "%s = %q\n", k, val)
			}
			return nil
		},
	}
	cmd.Flags().Bool("confirm", false, "Acknowledge that secrets will be printed in plaintext")
	return cmd
}
```

- [ ] **Step 3: Wire into root.go**

Modify `internal/cli/root.go`:

```go
func NewRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "oracle",
		Short:         "Deribit Oracle CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(
		newVersionCmd(version),
		newVaultCmd(),
	)
	return cmd
}
```

- [ ] **Step 4: Verify + commit**

```bash
go test ./internal/cli/... -v -timeout 120s
git add internal/cli go.mod go.sum
git commit -m "feat(cli): vault init/set/get/list/rm/rotate/export"
```

---

## Task 10: `oracle config` subcommand group

**Files:**
- Create: `internal/cli/config.go`
- Create: `internal/cli/config_test.go`
- Modify: `internal/cli/root.go`

- [ ] **Step 1: Write failing test**

```go
// internal/cli/config_test.go
package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigSetShowGet(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	cfgPath := filepath.Join(dir, "oracle-cli", "config.toml")

	run := func(args ...string) (string, error) {
		root := NewRootCmd("t")
		var out bytes.Buffer
		root.SetOut(&out)
		root.SetErr(&out)
		root.SetArgs(args)
		return out.String(), root.Execute()
	}

	if _, err := run("config", "set", "base_url", "https://x"); err != nil {
		t.Fatalf("set: %v", err)
	}
	got, err := run("config", "get", "base_url")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if strings.TrimSpace(got) != "https://x" {
		t.Errorf("got %q want https://x", got)
	}
	got, _ = run("config", "show")
	if !strings.Contains(got, "https://x") {
		t.Errorf("show missing base_url: %q", got)
	}
	if _, err := run("config", "set", "oracle_api_token", "leak"); err == nil {
		t.Error("expected refusal to store secret in config")
	}
	_ = cfgPath
}
```

- [ ] **Step 2: Implement config.go**

```go
// internal/cli/config.go
package cli

import (
	"fmt"
	"os"

	"github.com/metarsit/oracle-cli/internal/config"
	"github.com/spf13/cobra"
)

func configPath(cmd *cobra.Command) string {
	if v, _ := cmd.Flags().GetString("config"); v != "" {
		return v
	}
	if v := os.Getenv("ORACLE_CONFIG"); v != "" {
		return v
	}
	return config.ConfigPath()
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "config", Short: "Manage CLI config (non-secret keys)"}
	cmd.PersistentFlags().String("config", "", "Path to config.toml")
	cmd.AddCommand(newConfigShowCmd(), newConfigGetCmd(), newConfigSetCmd(), newConfigRmCmd())
	return cmd
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use: "show", Short: "Print resolved non-secret config",
		RunE: func(cmd *cobra.Command, _ []string) error {
			f, err := config.LoadFile(configPath(cmd))
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "base_url = %q\n", f.BaseURL)
			fmt.Fprintf(cmd.OutOrStdout(), "deribit_base_url = %q\n", f.DeribitBaseURL)
			fmt.Fprintf(cmd.OutOrStdout(), "output = %q\n", f.Output)
			fmt.Fprintf(cmd.OutOrStdout(), "timeout = %q\n", f.Timeout)
			return nil
		},
	}
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "get <key>", Short: "Print one config value",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := config.LoadFile(configPath(cmd))
			if err != nil {
				return err
			}
			v, err := configFieldGet(f, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), v)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use: "set <key> <value>", Short: "Set one config value",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if isSecretKey(args[0]) {
				return fmt.Errorf("%q is a secret; use `oracle vault set` instead", args[0])
			}
			path := configPath(cmd)
			f, err := config.LoadFile(path)
			if err != nil {
				return err
			}
			if err := configFieldSet(&f, args[0], args[1]); err != nil {
				return err
			}
			if err := config.EnsureConfigDir(); err != nil {
				return err
			}
			return config.SaveFile(path, f)
		},
	}
}

func newConfigRmCmd() *cobra.Command {
	return &cobra.Command{
		Use: "rm <key>", Short: "Clear one config value",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if isSecretKey(args[0]) {
				return fmt.Errorf("%q is a secret; use `oracle vault rm`", args[0])
			}
			path := configPath(cmd)
			f, err := config.LoadFile(path)
			if err != nil {
				return err
			}
			if err := configFieldSet(&f, args[0], ""); err != nil {
				return err
			}
			return config.SaveFile(path, f)
		},
	}
}

func isSecretKey(k string) bool {
	switch k {
	case "oracle_api_token", "deribit_client_id", "deribit_client_secret":
		return true
	}
	return false
}

func configFieldGet(f config.File, k string) (string, error) {
	switch k {
	case "base_url":
		return f.BaseURL, nil
	case "deribit_base_url":
		return f.DeribitBaseURL, nil
	case "output":
		return f.Output, nil
	case "timeout":
		return f.Timeout, nil
	}
	return "", fmt.Errorf("unknown config key %q", k)
}

func configFieldSet(f *config.File, k, v string) error {
	switch k {
	case "base_url":
		f.BaseURL = v
	case "deribit_base_url":
		f.DeribitBaseURL = v
	case "output":
		f.Output = v
	case "timeout":
		f.Timeout = v
	default:
		return fmt.Errorf("unknown config key %q", k)
	}
	return nil
}
```

- [ ] **Step 3: Wire into root.go**

```go
cmd.AddCommand(
    newVersionCmd(version),
    newVaultCmd(),
    newConfigCmd(),
)
```

- [ ] **Step 4: Verify + commit**

```bash
go test ./... -v
git add internal/cli
git commit -m "feat(cli): config show/get/set/rm with secret-key guard"
```

---

## Task 11: Layered config resolver

**Files:**
- Create: `internal/config/resolve.go`
- Create: `internal/config/resolve_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/config/resolve_test.go
package config

import (
	"testing"
	"time"
)

func TestResolvePrecedence(t *testing.T) {
	t.Setenv("ORACLE_BASE_URL", "env-url")
	t.Setenv("ORACLE_API_TOKEN", "env-tok")
	t.Setenv("ORACLE_OUTPUT", "")
	t.Setenv("ORACLE_TIMEOUT", "")

	in := Inputs{
		Flag: Flags{}, // no flag override
		File: File{BaseURL: "file-url", Output: "json"},
		Vault: map[string]string{"oracle_api_token": "vault-tok"},
	}
	got := Resolve(in)
	if got.BaseURL != "env-url" {
		t.Errorf("BaseURL: env should beat file: got %q", got.BaseURL)
	}
	if got.Token != "env-tok" {
		t.Errorf("Token: env should beat vault: got %q", got.Token)
	}
	if got.Output != "json" {
		t.Errorf("Output: file should fill missing env: got %q", got.Output)
	}
	if got.Timeout != 10*time.Second {
		t.Errorf("Timeout default missing: got %v", got.Timeout)
	}
}

func TestResolveFlagBeatsAll(t *testing.T) {
	t.Setenv("ORACLE_BASE_URL", "env")
	in := Inputs{
		Flag:  Flags{BaseURL: "flag"},
		File:  File{BaseURL: "file"},
		Vault: map[string]string{},
	}
	if got := Resolve(in); got.BaseURL != "flag" {
		t.Errorf("got %q want flag", got.BaseURL)
	}
}
```

- [ ] **Step 2: Implement resolve.go**

```go
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
```

- [ ] **Step 3: Verify + commit**

```bash
go test ./internal/config/... -v
git add internal/config
git commit -m "feat(config): layered resolver flag>env>vault>file>default"
```

---

## Task 12: CI workflow

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Write workflow**

```yaml
# .github/workflows/ci.yml
name: ci
on:
  push:
    branches: [main]
  pull_request:
jobs:
  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.23"
          cache: true
      - name: vet
        run: go vet ./...
      - name: gofmt
        run: |
          out=$(gofmt -l .)
          if [ -n "$out" ]; then echo "$out"; exit 1; fi
      - name: lint
        uses: golangci/golangci-lint-action@v6
        with:
          version: v1.62.0
      - name: govulncheck
        run: |
          go install golang.org/x/vuln/cmd/govulncheck@latest
          govulncheck ./...
      - name: test
        run: go test -race -coverprofile=cover.out ./...
      - name: coverage gate
        run: |
          pct=$(go tool cover -func cover.out | tail -1 | awk '{print $3}' | tr -d '%')
          awk -v p="$pct" 'BEGIN { exit !(p+0 >= 80) }'
```

- [ ] **Step 2: Commit**

```bash
git add .github/workflows/ci.yml
git commit -m "ci: add lint + vuln + test + coverage workflow"
```

---

## Phase 00 Done — proceed to Phase 01 (`2026-05-19-oracle-cli-01-client.md`).

# oracle-cli — Design Spec

- **Date:** 2026-05-19
- **Author:** Nicholas (`nicholas@alcoholnic.com`)
- **Status:** Draft → Approved on user sign-off
- **Target release:** v0.1.0
- **Repo:** `github.com/metarsit/oracle-cli`

## 1. Problem & Goals

The Deribit Oracle is a read-only analytics service that exposes a bearer-authed
HTTP API for instruments, prices, books, daily collar suggestions, settlements,
hedges, and positions. Today, the only canonical client is `curl`. We need a
first-class CLI that:

1. Wraps every oracle HTTP endpoint with a clean subcommand surface.
2. Is invokable both interactively by a human and as a subprocess by the
   Hermes Agent (machine-parseable JSON on stdout, logs on stderr).
3. Manages local config and secrets on disk safely, including Deribit OAuth2
   credentials for limited read-only Deribit calls.
4. Ships as a single static Go binary with no runtime dependencies.

Non-goals for v0.1.0:

- Order placement against Deribit (read-only only).
- A polling `watch` mode.
- Homebrew tap, goreleaser, `curl|bash` installer.
- Multi-account / multi-profile management.
- TUI mode.

## 2. Users & Use Cases

| User    | Use case                                                                                     |
|---------|----------------------------------------------------------------------------------------------|
| Human   | `oracle suggest --asset BTC` to inspect today's collar before manual trade.                  |
| Human   | `oracle vault init`, `oracle vault set oracle_api_token …` to provision credentials.         |
| Hermes  | `oracle plan --output json` once per planning loop; parses combined JSON for prompt context. |
| Hermes  | `oracle vault get deribit_client_secret` to pull Deribit creds before its own execution.     |

## 3. Tech Stack

- **Language:** Go 1.23
- **CLI framework:** `spf13/cobra` + `spf13/viper`
- **HTTP:** stdlib `net/http` with custom `internal/client`
- **Crypto:** stdlib `crypto/aes`, `crypto/cipher` (AES-256-GCM); argon2id KDF via `golang.org/x/crypto/argon2`
- **Config file:** TOML via `pelletier/go-toml/v2`
- **Output formatters:** stdlib `encoding/json`, `gopkg.in/yaml.v3`, `jedib0t/go-pretty/v6/table`
- **Logging:** stdlib `log/slog` (text handler → stderr)
- **Tests:** stdlib `testing`, `httptest`; no testify
- **Lint:** `golangci-lint` (errcheck, gosec, staticcheck, revive, gocritic, bodyclose, ineffassign, unconvert)
- **Vuln:** `govulncheck`
- **Build:** `Makefile`, `CGO_ENABLED=0`, single static binary

Rationale: matches sibling `deribit-analyser` toolchain; stdlib-first keeps
the attack surface small for a secrets-handling tool.

## 4. Repo Layout

```
oracle-cli/
├── cmd/oracle/main.go              # entrypoint, exit-code mapping
├── internal/
│   ├── cli/                        # cobra command files
│   │   ├── root.go                 # global flags, config/vault bootstrap
│   │   ├── health.go ready.go status.go
│   │   ├── instruments.go price.go book.go
│   │   ├── suggest.go engine.go settle.go hedge.go positions.go
│   │   ├── plan.go                 # aggregator
│   │   ├── config.go               # config show/set/get/rm
│   │   └── vault.go                # vault init/unlock/set/get/list/rm/rotate/export
│   ├── client/                     # oracle HTTP client
│   │   ├── client.go               # http.Client, bearer middleware, retry
│   │   ├── envelope.go             # data/error envelope types + decoder
│   │   ├── errors.go               # typed errors (ErrAuth, ErrNotFound, ...)
│   │   └── types.go                # Suggestion, Hedge, Settlement, Position, Instrument
│   ├── deribit/                    # read-only Deribit OAuth2 + queries
│   │   ├── client.go               # OAuth2 client_credentials token cache
│   │   └── account.go              # get_positions, get_account_summary
│   ├── vault/                      # encrypted secret store
│   │   ├── vault.go                # open/save, AES-256-GCM
│   │   ├── kdf.go                  # argon2id passphrase → key
│   │   ├── file.go                 # path resolution, perm checks
│   │   └── format.go               # header marshalling
│   ├── config/                     # layered config resolution
│   │   ├── config.go               # flag>env>vault>file>default merge
│   │   └── paths.go                # XDG dirs, default file paths
│   └── format/                     # output renderers
│       ├── format.go               # Renderer interface, NewRenderer factory
│       ├── json.go yaml.go table.go
│       └── testdata/               # golden files
├── go.mod
├── Makefile
├── .github/workflows/ci.yml
├── .golangci.yml
├── LICENSE
├── CHANGELOG.md
├── SECURITY.md
└── README.md
```

## 5. CLI Surface

### 5.1 Subcommands (v0.1.0)

| Command                                       | Endpoint                          | Auth |
|-----------------------------------------------|-----------------------------------|------|
| `oracle health`                               | GET `/healthz`                    | none |
| `oracle ready`                                | GET `/readyz`                     | none |
| `oracle status`                               | GET `/v1/status`                  | bearer |
| `oracle instruments --base <BTC|ETH> [--kind option|perp|future]` | GET `/v1/instruments` | bearer |
| `oracle price --instrument <NAME>`            | GET `/v1/prices/latest`           | bearer |
| `oracle book --instrument <NAME>`             | GET `/v1/book/top`                | bearer |
| `oracle suggest --asset <BTC|ETH>`            | GET `/v1/suggestions/latest`      | bearer |
| `oracle engine run`                           | POST `/v1/engine/run`             | bearer |
| `oracle settle --asset <BTC|ETH>`             | GET `/v1/settlements/latest`      | bearer |
| `oracle hedge --asset <BTC|ETH>`              | GET `/v1/hedges/latest`           | bearer |
| `oracle positions`                            | GET `/v1/positions/current`       | bearer |
| `oracle plan`                                 | aggregator (see §6)               | bearer |
| `oracle deribit positions`                    | Deribit `/private/get_positions`  | deribit oauth2 |
| `oracle deribit balance [--currency BTC|ETH|USDC]` | Deribit `/private/get_account_summary` | deribit oauth2 |
| `oracle config show`                          | —                                 | none |
| `oracle config get <key>`                     | —                                 | none |
| `oracle config set <key> <value>`             | —                                 | none |
| `oracle config rm <key>`                      | —                                 | none |
| `oracle vault init`                           | —                                 | passphrase |
| `oracle vault set <key> <value>`              | —                                 | passphrase |
| `oracle vault get <key>`                      | —                                 | passphrase |
| `oracle vault list`                           | —                                 | passphrase |
| `oracle vault rm <key>`                       | —                                 | passphrase |
| `oracle vault rotate`                         | —                                 | old + new passphrase |
| `oracle vault export --confirm`               | —                                 | passphrase (gated) |
| `oracle version`                              | —                                 | none |

### 5.2 Global flags

```
--base-url <url>          ORACLE_BASE_URL,        default http://localhost:8080
--token   <token>         ORACLE_API_TOKEN,       no default (never persisted via flag)
--output  json|table|yaml ORACLE_OUTPUT,          default table
--timeout <duration>      ORACLE_TIMEOUT,         default 10s
--verbose / -v            (slog level Debug to stderr)
--config  <path>          ORACLE_CONFIG,          default $XDG_CONFIG_HOME/oracle-cli/config.toml
--vault   <path>          ORACLE_VAULT,           default $XDG_CONFIG_HOME/oracle-cli/secrets.vault
```

### 5.3 Exit codes

| Code | Meaning                                            |
|------|----------------------------------------------------|
| 0    | success                                            |
| 1    | generic error (validation, config, decode, vault) |
| 2    | auth failed (HTTP 401/403)                         |
| 3    | not found / no data yet (HTTP 404)                 |
| 4    | oracle unreachable (network, DNS, timeout)         |
| 5    | oracle degraded (HTTP 503 from any endpoint, or non-ready `/readyz` payload) |

## 6. `oracle plan` aggregator

Hermes' primary entrypoint. One CLI invocation, one combined JSON document.

**Calls fan-out concurrently via `errgroup`:**

1. GET `/v1/status`
2. GET `/v1/suggestions/latest?asset=BTC`
3. GET `/v1/suggestions/latest?asset=ETH`
4. GET `/v1/hedges/latest?asset=BTC`
5. GET `/v1/hedges/latest?asset=ETH`
6. GET `/v1/positions/current`

Per-call failure becomes `null` plus an entry in the `errors` array. Exit
code semantics:

- ≥ 1 successful call → exit 0 (partial data still useful to Hermes).
- All calls failed with the same kind → map per §5.3 (all 401 → exit 2,
  all 404 → exit 3, all network → exit 4, all 503 → exit 5).
- All calls failed with mixed kinds → exit 1.

**Output shape (always JSON regardless of `--output`):**

```json
{
  "data": {
    "status": { ... },
    "suggestions": { "BTC": { ... }, "ETH": null },
    "hedges":      { "BTC": { ... }, "ETH": { ... } },
    "positions":   { ... }
  },
  "errors": [
    { "endpoint": "suggest:ETH", "code": "NOT_FOUND", "message": "..." }
  ],
  "meta": { "ts": "2026-05-19T10:00:00Z", "duration_ms": 142 }
}
```

## 7. Data Flow

```
cobra parse
  → config.Load() merges flag > env > vault > file > default
    → vault.Open() decrypts iff command needs a vaulted key
      → client.New(baseURL, token, timeout)
        → client.Get/Post(ctx, endpoint, params)
          → http.Client.Do (bearer header, 10 MiB body cap)
            → decode envelope { data | error, meta }
              → unmarshal data into typed struct
                → format.Render(struct, output) → os.Stdout
slog → os.Stderr throughout
errors.As → exit-code mapping in main()
```

## 8. Config Resolution

| Key                     | Sources (highest priority first)                                                       |
|-------------------------|----------------------------------------------------------------------------------------|
| `base_url`              | `--base-url`, `ORACLE_BASE_URL`, vault, config.toml, default `http://localhost:8080`   |
| `oracle_api_token`      | `--token`, `ORACLE_API_TOKEN`, vault, _(no file, no default)_                          |
| `deribit_client_id`     | `--deribit-client-id`, `DERIBIT_CLIENT_ID`, vault                                      |
| `deribit_client_secret` | `DERIBIT_CLIENT_SECRET` env, vault, _(never a flag — leaks in shell history)_          |
| `deribit_base_url`      | `--deribit-base-url`, `DERIBIT_BASE_URL`, config.toml, default `https://www.deribit.com/api/v2` |
| `output`                | `--output`, `ORACLE_OUTPUT`, config.toml, default `table`                              |
| `timeout`               | `--timeout`, `ORACLE_TIMEOUT`, config.toml, default `10s`                              |

`config.toml` carries only non-secret keys. `config set deribit_client_secret …`
is rejected with a hint to use `vault set` instead.

Vault unlock is **lazy**: triggered only when a command needs a key still
missing after flag+env resolution. `health` / `ready` / `version` skip vault
entirely.

## 9. Vault — Encrypted Secret Store

### 9.1 Threat model

- **In scope:** attacker reads `secrets.vault` from a stolen laptop, leaked
  backup, accidental git commit. Brute force must be infeasible.
- **Out of scope:** in-process memory disclosure, root on the box, malicious
  CLI binary, side-channel attacks.

### 9.2 File format (binary)

```
offset  field      bytes  notes
0       magic      4      "OCLI"
4       version    1      = 1
5       kdf_id     1      = 1 (argon2id)
6       salt       16     random, regenerated only on `vault rotate`
22      nonce      12     random per-write, AES-GCM nonce
34      ct_len     4      uint32 big-endian
38      ct         ct_len AES-256-GCM(plaintext, AAD = bytes[0:34])
```

AAD binds ciphertext to header → header tampering invalidates auth tag.

### 9.3 KDF parameters (argon2id)

- `time = 3`
- `memory = 64 MiB`
- `threads = 4`
- `keyLen = 32`
- `saltLen = 16`

Tuned for ~250 ms on Apple M-series. Acceptable single-invocation cost; key
cached for process lifetime so multi-step interactions stay fast.

### 9.4 Plaintext payload (TOML)

```toml
[secrets]
oracle_api_token       = "..."
deribit_client_id      = "..."
deribit_client_secret  = "..."
```

### 9.5 Operations

| Subcommand                | Behavior                                                          |
|---------------------------|-------------------------------------------------------------------|
| `vault init`              | Prompt passphrase twice (no echo), create empty vault, mode 0600. |
| `vault set <key> <value>` | Decrypt → patch → re-encrypt with fresh nonce.                    |
| `vault get <key>`         | Decrypt → write to stdout (warn on stderr if stdout is a TTY).    |
| `vault list`              | Decrypt → print keys only (never values).                         |
| `vault rm <key>`          | Decrypt → delete key → re-encrypt.                                |
| `vault rotate`            | Prompt old + new, re-encrypt with new salt + nonce.               |
| `vault export --confirm`  | Print decrypted TOML to stdout. Refuse without `--confirm` flag.  |

### 9.6 Passphrase source

1. `ORACLE_VAULT_PASSPHRASE` env var (Hermes path).
2. Else, if stdin is a TTY, prompt with no echo (`golang.org/x/term`).
3. Else, error `vault locked: set ORACLE_VAULT_PASSPHRASE`.

### 9.7 Refuse-to-operate guards

- File mode looser than `0600` (i.e. any bit in `0o077` set) → error, suggest `chmod 600`.
- Immediate parent dir world-writable (`Mode().Perm()&0o002 != 0`) → error.
- Magic mismatch → "not an oracle-cli vault".
- Unknown version → "vault version X, this CLI supports 1; upgrade".
- GCM auth failure → "wrong passphrase or corrupt vault" (do not distinguish).

## 10. HTTP Client Behavior

- One `*http.Client` per CLI invocation, reused across fan-out calls.
- `context.Context` per request, deadline = `--timeout`.
- Bearer header added by a `RoundTripper` wrapper; bearer redacted in any
  error string via custom `Error()` methods.
- Single retry on `ErrNetwork` only (idempotent GETs), 250 ms backoff. POSTs
  never retried.
- Response body cap: 10 MiB (defense vs. runaway response).
- Decode envelope first → branch on `.error` presence → unmarshal `.data`
  into the typed struct.

## 11. Error Handling

```go
type ErrAuth     struct{ Status int; Msg string }
type ErrNotFound struct{ Msg string }
type ErrNetwork  struct{ Err error }
type ErrDegraded struct{ Status int; Msg string }
type ErrAPI      struct{ Code, Msg string; Status int }
```

- `main.go` uses `errors.As` to map to exit codes.
- No `panic`s escape `cli.Run`.
- With `--output json`, errors print the same envelope shape the oracle uses
  (single parser for Hermes).
- With `--output table`/`yaml`, errors go to stderr; stdout stays empty.

## 12. Deribit Read-Only Client

- OAuth2 grant: `client_credentials` against `/public/auth`.
- Access token cached in-memory for `expires_in − 30s`.
- Endpoints used (read-only): `/private/get_positions`, `/private/get_account_summary`.
- `--currency` defaults to `BTC` for `balance`; `positions` returns all.
- Errors map: `unauthorized` → `ErrAuth` → exit 2.

## 13. Testing Strategy

| Layer                                          | Tool                                  | Target |
|------------------------------------------------|---------------------------------------|--------|
| Formatters (json/yaml/table golden files)      | stdlib `testing`, `testdata/`         | ≥ 90%  |
| Vault (KDF, encrypt/decrypt, perms)            | stdlib + `t.TempDir()`                | ≥ 95%  |
| Config resolution (flag>env>vault>file>default)| table-driven                          | ≥ 90%  |
| Client envelope + error mapping                | `httptest.NewServer`                  | ≥ 90%  |
| Integration — CLI subprocess vs fake oracle    | `httptest` + `exec.Command`           | every subcommand |
| Property — config-merge associativity          | `testing/quick`                       | sanity |

- TDD: each `internal/cli/*.go` has `*_test.go` written first.
- Coverage gate: `go test -coverprofile=cover.out ./...`; fail CI if total < 80%.

## 14. Build, Lint, CI

**Makefile targets:** `build`, `test`, `lint`, `cover`, `fmt`, `vet`, `vuln`, `release`.

**Build flags:**
- `CGO_ENABLED=0`
- `-ldflags "-s -w -X main.version=$(git describe --tags --dirty)"`

**`.golangci.yml`** enables: `errcheck`, `gosec`, `staticcheck`, `revive`,
`gocritic`, `bodyclose`, `ineffassign`, `unconvert`. Disable `gochecknoglobals`
(cobra needs globals).

**CI — `.github/workflows/ci.yml`:**
- Matrix: `ubuntu-latest`, `macos-latest`. Go 1.23.
- Steps: checkout → setup-go (cache) → `make fmt vet lint vuln test cover` →
  upload coverage artifact.
- On tag `v*`: trigger release job that runs `make release` and `gh release create`.

## 15. Distribution

**v0.1.0 minimum:**
- `make release` builds `oracle-cli_${VERSION}_{darwin_arm64,darwin_amd64,linux_amd64}` + `SHA256SUMS`.
- GitHub Release attaches binaries via `gh release create`.
- `go install github.com/metarsit/oracle-cli/cmd/oracle@latest` works.

**Deferred to v0.2.0:**
- Goreleaser, Homebrew tap, `curl|bash` installer, `watch` polling mode.

## 16. Project Hygiene

- `LICENSE` MIT
- `CHANGELOG.md` (Keep a Changelog)
- `SECURITY.md` (disclosure address, vault threat model link)
- `README.md` (quickstart, vault setup, Hermes integration snippet, exit-code table)

## 17. Open Risks

| Risk                                            | Mitigation                                                    |
|-------------------------------------------------|---------------------------------------------------------------|
| Decimal precision loss on prices                | Decode JSON numbers into `json.Number`; convert only at format-time. |
| Bearer token leaks via shell history (`--token`)| Document env var preferred; flag exists but is discouraged.  |
| Vault key cached in process memory              | Documented out-of-scope for v0.1; future: `mlock` on Linux.  |
| Hermes parses partial JSON on subprocess crash  | Buffer full response, write atomically with single `os.Stdout.Write`. |
| Oracle API drift (envelope shape change)        | Versioned types in `internal/client/types.go`; integration tests pin shape. |

## 18. Hermes Integration Snippet (target UX)

```bash
# One-time setup
oracle vault init
oracle vault set oracle_api_token     "$ORACLE_API_TOKEN"
oracle vault set deribit_client_id    "$DERIBIT_CLIENT_ID"
oracle vault set deribit_client_secret "$DERIBIT_CLIENT_SECRET"
oracle config set base_url "https://oracle.up.railway.app"

# Hermes loop (env-injected passphrase)
ORACLE_VAULT_PASSPHRASE=$VAULT_PW \
  oracle plan --output json > /tmp/oracle-plan.json
```

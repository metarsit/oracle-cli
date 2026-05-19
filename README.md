# oracle-cli

Command-line client for the Deribit Oracle HTTP API. Used directly by humans
and as a subprocess by the Hermes Agent.

## Install

```bash
go install github.com/metarsit/oracle-cli/cmd/oracle@latest
```

Or grab a binary from [Releases](https://github.com/metarsit/oracle-cli/releases).

## Quickstart

```bash
# One-time
oracle vault init                       # prompts for vault passphrase
oracle vault set oracle_api_token "<token-from-oracle-deploy>"
oracle config set base_url "https://oracle.up.railway.app"

# Daily use
oracle suggest --asset BTC
oracle plan                             # JSON aggregate for Hermes
oracle deribit balance --currency BTC
```

## Configuration

Resolution order: flag -> env -> vault -> `~/.config/oracle-cli/config.toml` -> default.

| Key                     | Env                       | Default                                    |
|-------------------------|---------------------------|--------------------------------------------|
| `base_url`              | `ORACLE_BASE_URL`         | `http://localhost:8080`                    |
| `oracle_api_token`      | `ORACLE_API_TOKEN`        | _(vault or env only)_                      |
| `deribit_client_id`     | `DERIBIT_CLIENT_ID`       | _(vault or env only)_                      |
| `deribit_client_secret` | `DERIBIT_CLIENT_SECRET`   | _(vault or env only)_                      |
| `deribit_base_url`      | `DERIBIT_BASE_URL`        | `https://www.deribit.com/api/v2`           |
| `output`                | `ORACLE_OUTPUT`           | `table`                                    |
| `timeout`               | `ORACLE_TIMEOUT`          | `10s`                                      |

**Secrets never persist to `config.toml`.** Use `oracle vault set` instead.

## Exit codes

| Code | Meaning                                                    |
|------|------------------------------------------------------------|
| 0    | success                                                    |
| 1    | generic error                                              |
| 2    | auth failed (HTTP 401/403)                                 |
| 3    | not found (HTTP 404)                                       |
| 4    | oracle unreachable (network)                               |
| 5    | oracle degraded (HTTP 503 / non-ready)                     |

## Hermes integration

```bash
ORACLE_VAULT_PASSPHRASE=$VAULT_PW \
  oracle plan --output json > /tmp/oracle-plan.json
```

Stdout = single JSON document. Stderr = logs / progress / errors.

## Vault security

AES-256-GCM with argon2id KDF (time=3, memory=64 MiB, threads=4). Fresh nonce
per write. Refuses to operate when file mode is looser than `0600` or the
parent directory is world-writable. See `SECURITY.md` for the full threat model.

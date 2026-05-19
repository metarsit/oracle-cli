# Security policy

## Reporting a vulnerability

Email `nicholas@alcoholnic.com`. Please do not file public issues for security
problems.

## Vault threat model

In scope:
- Attacker reads `secrets.vault` from a stolen laptop, leaked backup, or
  accidental git commit. Brute-force must be infeasible.

Out of scope (v0.1.0):
- In-process memory disclosure (cleared key in memory).
- Root-on-the-box, malicious modified binary, side-channel attacks.

Crypto:
- AES-256-GCM with random 12-byte nonce per write.
- argon2id KDF: time=3, memory=64 MiB, threads=4, 16-byte salt.
- Header binds AAD -> tamper-evident.

The CLI refuses to operate when:
- vault file mode is looser than `0600`.
- the parent directory is world-writable.

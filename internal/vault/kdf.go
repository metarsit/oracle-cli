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

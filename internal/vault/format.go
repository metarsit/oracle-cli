package vault

import (
	"encoding/binary"
	"errors"
	"fmt"
)

const (
	saltLen     = 16
	nonceLen    = 12
	magicLen    = 4
	headerLen   = magicLen + 1 + 1 + saltLen + nonceLen + 4 // 38
	currentVer  = 1
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

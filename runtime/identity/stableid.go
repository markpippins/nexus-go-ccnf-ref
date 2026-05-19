package identity

import (
	"crypto/sha256"
	"fmt"
)

func Assemble(seed OriginSeed, sig SemanticSignature) StableID {
	h := sha256.New()
	h.Write([]byte(seed))
	h.Write([]byte{0x00})
	h.Write([]byte(sig))
	return StableID(fmt.Sprintf("%x", h.Sum(nil)))
}

package internal

import (
	"crypto/sha256"
	"fmt"
)

func HashStableID(seed, semanticSignature string) string {
	h := sha256.New()
	h.Write([]byte(seed))
	h.Write([]byte{0x00})
	h.Write([]byte(semanticSignature))
	return fmt.Sprintf("%x", h.Sum(nil))
}

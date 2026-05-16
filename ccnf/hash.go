package ccnf

import (
	"crypto/sha256"
	"fmt"
)

func ComputeHash(cer *CER) string {
	m := BuildCERMap(cer)
	delete(m, "signature")
	canonical := CanonicalJSON(m)
	h := sha256.Sum256(canonical)
	return fmt.Sprintf("%x", h[:])
}

func BuildSignature(cer *CER) map[string]any {
	hash := ComputeHash(cer)
	return map[string]any{
		"hash":      hash,
		"signed_by": nil,
	}
}

package internal

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

type CanonicalInput struct {
	Name       string
	DirectDeps []string
}

func CanonicalSemanticSignature(name string, deps []string) string {
	sorted := make([]string, len(deps))
	copy(sorted, deps)
	sort.Strings(sorted)
	return name + "|" + strings.Join(sorted, ",")
}

func HashSemanticSignature(input CanonicalInput) string {
	canonical := CanonicalSemanticSignature(input.Name, input.DirectDeps)
	h := sha256.Sum256([]byte(canonical))
	return fmt.Sprintf("%x", h[:])
}

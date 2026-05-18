package runtime

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

var traceDomainHash []byte

func init() {
	h := sha256.Sum256([]byte("ccnf-domain:trace:v1"))
	traceDomainHash = h[:]
}

type TraceBuilder struct {
	hashes [][]byte
}

func NewTraceBuilder() *TraceBuilder {
	return &TraceBuilder{
		hashes: make([][]byte, 0, 1),
	}
}

func (t *TraceBuilder) Append(cerHashHex string) {
	raw, err := hex.DecodeString(cerHashHex)
	if err != nil || len(raw) != 32 {
		panic(fmt.Sprintf("TraceBuilder: invalid CER hash: %q", cerHashHex))
	}
	t.hashes = append(t.hashes, raw)
}

func (t *TraceBuilder) RootHash() string {
	h := sha256.New()
	h.Write(traceDomainHash)

	countBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(countBytes, uint64(len(t.hashes)))
	h.Write(countBytes)

	for _, raw := range t.hashes {
		h.Write(raw)
	}

	return hex.EncodeToString(h.Sum(nil))
}

func (t *TraceBuilder) EventCount() uint64 {
	return uint64(len(t.hashes))
}

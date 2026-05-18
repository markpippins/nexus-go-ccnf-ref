package runtime

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
)

var replayDomainHash []byte

func init() {
	h := sha256.Sum256([]byte("ccnf-domain:replay-binding:v1"))
	replayDomainHash = h[:]
}

func ComputeReplayBinding(cerRootHash, traceRootHash string, semanticsVersion, ccnfVersion int) string {
	cerRaw, err := hex.DecodeString(cerRootHash)
	if err != nil || len(cerRaw) != 32 {
		panic("ComputeReplayBinding: invalid cer_root_hash")
	}
	traceRaw, err := hex.DecodeString(traceRootHash)
	if err != nil || len(traceRaw) != 32 {
		panic("ComputeReplayBinding: invalid trace_root_hash")
	}

	h := sha256.New()
	h.Write(replayDomainHash)

	lenBuf := make([]byte, 8)
	binary.BigEndian.PutUint64(lenBuf, uint64(len(cerRaw)))
	h.Write(lenBuf)
	h.Write(cerRaw)

	binary.BigEndian.PutUint64(lenBuf, uint64(len(traceRaw)))
	h.Write(lenBuf)
	h.Write(traceRaw)

	binary.BigEndian.PutUint64(lenBuf, uint64(semanticsVersion))
	h.Write(lenBuf)

	binary.BigEndian.PutUint64(lenBuf, uint64(ccnfVersion))
	h.Write(lenBuf)

	return hex.EncodeToString(h.Sum(nil))
}

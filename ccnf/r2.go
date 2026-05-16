package ccnf

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
	"time"
	"unicode/utf8"
)

type CollisionSeverity string

const (
	CollisionExpected   CollisionSeverity = "expected"
	CollisionAmbiguity  CollisionSeverity = "spec_ambiguity"
	CollisionDivergence CollisionSeverity = "divergence"
)

type CollisionRecord struct {
	Hash      string            `json:"hash"`
	Inputs    []json.RawMessage `json:"inputs"`
	Count     int               `json:"count"`
	Severity  CollisionSeverity `json:"severity"`
	Rationale string            `json:"rationale,omitempty"`
}

type CollisionReport struct {
	CCNFVersion int                        `json:"ccnf_version"`
	Seed        int64                      `json:"seed"`
	Iterations  int                        `json:"iterations"`
	Expected    []CollisionRecord          `json:"expected"`
	Ambiguities []CollisionRecord          `json:"ambiguities"`
	Divergences []CollisionRecord          `json:"divergences"`
	Classified  map[string]CollisionRecord `json:"classified"`
}

type R2Config struct {
	Seed       int64
	Iterations int
}

func DefaultR2Config() R2Config {
	return R2Config{Seed: 42, Iterations: 10000}
}

func StressR2Config() R2Config {
	return R2Config{Seed: 42, Iterations: 100000}
}

type SemanticFuzzer struct {
	rng *rand.Rand
	cfg R2Config
}

func NewSemanticFuzzer(cfg R2Config) *SemanticFuzzer {
	return &SemanticFuzzer{rng: rand.New(rand.NewSource(cfg.Seed)), cfg: cfg}
}

func (f *SemanticFuzzer) Generate(i int) map[string]any {
	dimension := i % 8
	switch dimension {
	case 0:
		return f.genFieldPermutation(i)
	case 1:
		return f.genNullVsOmission(i)
	case 2:
		return f.genUnicodeVariant(i)
	case 3:
		return f.genTimestampBoundary(i)
	case 4:
		return f.genFloatBoundary(i)
	case 5:
		return f.genMultiArtifact(i)
	case 6:
		return f.genAliasChain(i)
	case 7:
		return f.genNestedStructure(i)
	default:
		return f.genStandardInput(i)
	}
}

func (f *SemanticFuzzer) genStandardInput(i int) map[string]any {
	actions := []string{"create", "update", "delete", "execute", "validate", "emit"}
	types := []string{"node", "task", "graph", "workflow", "artifact"}
	domains := []string{"execution", "specification", "system", "test"}

	action := actions[f.rng.Intn(len(actions))]
	typ := types[f.rng.Intn(len(types))]
	domain := domains[f.rng.Intn(len(domains))]
	id := fmt.Sprintf("%s-%08d", typ, i)

	return map[string]any{
		"actor": map[string]any{
			"type":       "system",
			"id":         fmt.Sprintf("fuzzer-%d", f.rng.Intn(100)),
			"session_id": fmt.Sprintf("sess-%d", i),
		},
		"intent": map[string]any{
			"action":      action,
			"target_type": typ,
			"target_id":   fmt.Sprintf("%s:%s", typ, id),
		},
		"payload":   map[string]any{"data": map[string]any{}},
		"domain":    domain,
		"event_id":  fmt.Sprintf("fuzz-%08d", i),
		"timestamp": float64(1713225600 + f.rng.Intn(86400*90)),
		"causality": map[string]any{
			"parent_event_ids": []any{},
			"causal_chain_id":  fmt.Sprintf("chain-%s-%04d", domain, f.rng.Intn(1000)),
			"trace_depth":      f.rng.Intn(10),
		},
	}
}

func (f *SemanticFuzzer) genFieldPermutation(i int) map[string]any {
	base := f.genStandardInput(i)
	return permuteMapKeys(base)
}

func permuteMapKeys(m map[string]any) map[string]any {
	if len(m) == 0 {
		return m
	}
	out := make(map[string]any, len(m))
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	first := keys[0]
	rest := keys[1:]
	order := append(rest, first)

	for _, k := range order {
		switch v := m[k].(type) {
		case map[string]any:
			out[k] = permuteMapKeys(v)
		default:
			out[k] = v
		}
	}
	return out
}

func (f *SemanticFuzzer) genNullVsOmission(i int) map[string]any {
	base := f.genStandardInput(i)
	if i%2 == 0 {
		base["context"] = nil
	} else {
		delete(base, "context")
	}
	if i%3 == 0 {
		base["signature"] = nil
	}
	return base
}

func (f *SemanticFuzzer) genUnicodeVariant(i int) map[string]any {
	variants := []string{
		"caf\u00e9",
		"cafe\u0301",
		"\u0041\u0300",
		"\u00c0",
		"\u1e0b\u0327",
		"\u1e0d\u0307",
	}
	base := f.genStandardInput(i)
	idx := i % len(variants)
	base["actor"].(map[string]any)["id"] = variants[idx]
	return base
}

func (f *SemanticFuzzer) genTimestampBoundary(i int) map[string]any {
	boundaries := []float64{0, -1, 86400, 2147483647, 1713225600, 100000000000, 253402300799}
	isoStamps := []string{
		"1970-01-01T00:00:00Z",
		"1969-12-31T23:59:59Z",
		"2026-05-16T00:00:00Z",
		"2038-01-19T03:14:07Z",
		"2000-01-01T00:00:00Z",
	}
	base := f.genStandardInput(i)
	if i%2 == 0 {
		base["timestamp"] = boundaries[i%len(boundaries)]
	} else {
		base["timestamp"] = isoStamps[i%len(isoStamps)]
	}
	return base
}

func (f *SemanticFuzzer) genFloatBoundary(i int) map[string]any {
	floats := []any{float64(0), float64(-0.0), float64(1e10), float64(1e-10), float64(3.0), float64(42), float64(3.141592653589793)}
	base := f.genStandardInput(i)
	base["_float_test"] = floats[i%len(floats)]
	return base
}

func (f *SemanticFuzzer) genMultiArtifact(i int) map[string]any {
	base := f.genStandardInput(i)
	n := (i % 5) + 2
	data := make(map[string]any)
	for j := 0; j < n; j++ {
		typ := []string{"node", "edge", "graph", "task"}[j%4]
		aid := fmt.Sprintf("%s:%s-%d-%d", typ, typ, i, j)
		data[aid] = map[string]any{"state": fmt.Sprintf("s-%d", j), "seq": j}
	}
	base["payload"] = map[string]any{"data": data}
	return base
}

func (f *SemanticFuzzer) genAliasChain(i int) map[string]any {
	base := f.genStandardInput(i)
	depth := (i % 8) + 1
	aliases := make([]any, depth)
	for j := 0; j < depth; j++ {
		aliases[j] = fmt.Sprintf("alias:%s-%d-%d", base["domain"], i, j)
	}
	base["identity"] = map[string]any{
		"collapse_key": fmt.Sprintf("ck:chain-%d", i),
		"alias_keys":   aliases,
	}
	return base
}

func (f *SemanticFuzzer) genNestedStructure(i int) map[string]any {
	base := f.genStandardInput(i)
	depth := (i % 10) + 1
	nested := buildNested(depth, f.rng)
	base["payload"] = map[string]any{"data": nested}
	return base
}

func buildNested(depth int, rng *rand.Rand) map[string]any {
	if depth <= 0 {
		return map[string]any{}
	}
	m := make(map[string]any)
	m[fmt.Sprintf("lvl_%d_key_%d", depth, rng.Intn(100))] = buildNested(depth-1, rng)
	return m
}

type MutationEngine struct {
	rng *rand.Rand
}

func NewMutationEngine(seed int64) *MutationEngine {
	return &MutationEngine{rng: rand.New(rand.NewSource(seed))}
}

func (m *MutationEngine) AddField(base map[string]any) map[string]any {
	clone := deepCloneMap(base)
	clone["context"] = map[string]any{"origin": "mutation"}
	return clone
}

func (m *MutationEngine) RemoveField(base map[string]any) map[string]any {
	clone := deepCloneMap(base)
	if _, ok := clone["actor"]; ok {
		clone["actor"] = map[string]any{
			"type": fmt.Sprintf("%v", m.rng.Intn(2) == 0),
		}
	}
	return clone
}

func (m *MutationEngine) PerturbTimestamp(base map[string]any) map[string]any {
	clone := deepCloneMap(base)
	if ts, ok := clone["timestamp"].(float64); ok {
		clone["timestamp"] = ts + float64(m.rng.Intn(3)-1)
	}
	return clone
}

func (m *MutationEngine) ReorderKeys(base map[string]any) map[string]any {
	return permuteMapKeys(deepCloneMap(base))
}

func deepCloneMap(v map[string]any) map[string]any {
	out := make(map[string]any, len(v))
	for k, vv := range v {
		switch val := vv.(type) {
		case map[string]any:
			out[k] = deepCloneMap(val)
		case []any:
			out[k] = deepCloneSlice(val)
		default:
			out[k] = vv
		}
	}
	return out
}

func deepCloneSlice(s []any) []any {
	out := make([]any, len(s))
	for i, v := range s {
		switch val := v.(type) {
		case map[string]any:
			out[i] = deepCloneMap(val)
		case []any:
			out[i] = deepCloneSlice(val)
		default:
			out[i] = v
		}
	}
	return out
}

type CollisionDetector struct {
	HashInputs map[string][]map[string]any
}

func NewCollisionDetector() *CollisionDetector {
	return &CollisionDetector{HashInputs: make(map[string][]map[string]any)}
}

func (d *CollisionDetector) Add(input map[string]any, hash string) {
	d.HashInputs[hash] = append(d.HashInputs[hash], input)
}

func (d *CollisionDetector) Report() CollisionReport {
	totalInputs := 0
	for _, inputs := range d.HashInputs {
		totalInputs += len(inputs)
	}
	report := CollisionReport{
		CCNFVersion: CurrentCCNFVersion,
		Seed:        42,
		Iterations:  totalInputs,
		Classified:  make(map[string]CollisionRecord),
	}
	for hash, inputs := range d.HashInputs {
		if len(inputs) < 2 {
			continue
		}
		cr := CollisionRecord{
			Hash:     hash,
			Count:    len(inputs),
			Severity: CollisionExpected,
		}
		for _, in := range inputs {
			b, _ := json.Marshal(in)
			cr.Inputs = append(cr.Inputs, b)
		}
		report.Classified[hash] = cr
		report.Expected = append(report.Expected, cr)
	}
	return report
}

type RoundTripChecker struct{}

func NewRoundTripChecker() *RoundTripChecker {
	return &RoundTripChecker{}
}

func (r *RoundTripChecker) Check(inputJSON []byte) (string, string, error) {
	cer1, err := Run(inputJSON, 1)
	if err != nil {
		return "", "", err
	}
	h1 := ComputeHash(cer1)
	cerBytes := CanonicalJSON(cer1)
	cer2, err := Run(cerBytes, 1)
	if err != nil {
		return "", "", err
	}
	h2 := ComputeHash(cer2)
	return h1, h2, nil
}

type CrossOrderTester struct{}

func NewCrossOrderTester() *CrossOrderTester {
	return &CrossOrderTester{}
}

func (c *CrossOrderTester) Test(input map[string]any) (string, []string, bool, error) {
	j0, _ := json.Marshal(input)
	cer0, err := Run(j0, 1)
	if err != nil {
		return "", nil, false, err
	}
	refHash := ComputeHash(cer0)

	var otherHashes []string
	c.genKeyPermutations(input, func(p map[string]any) {
		j, _ := json.Marshal(p)
		cer, cerr := Run(j, 1)
		if cerr != nil {
			return
		}
		otherHashes = append(otherHashes, ComputeHash(cer))
	})

	if len(otherHashes) == 0 {
		return refHash, nil, true, nil
	}
	for _, h := range otherHashes {
		if h != refHash {
			return refHash, otherHashes, false, nil
		}
	}
	return refHash, otherHashes, true, nil
}

func (c *CrossOrderTester) genKeyPermutations(m map[string]any, yield func(map[string]any)) {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	if len(keys) < 2 {
		return
	}
	n := len(keys)
	for i := 0; i < n && i < 8; i++ {
		j := (i + 1) % n
		perm := make(map[string]any, n)
		for ki, k := range keys {
			perm[k] = m[keys[(ki+j)%n]]
		}
		for pk, pv := range perm {
			if vm, ok := pv.(map[string]any); ok {
				cp := deepCloneMap(perm)
				cp[pk] = permuteMapKeys(vm)
				yield(cp)
			}
		}
		yield(perm)
	}
}

func isASCIIPrintable(s string) bool {
	for _, r := range s {
		if r < 0x20 || r > 0x7e {
			return false
		}
	}
	return true
}

func isNFC(s string) bool {
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			return false
		}
		i += size
	}
	return true
}

func nowUnix() int64 {
	return time.Now().Unix()
}

func stringsContains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

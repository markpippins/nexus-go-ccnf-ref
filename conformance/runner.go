package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anomalyco/nexus-ccnf-ref/ccnf"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: ccnf-conformance <command> [args]\n")
		fmt.Fprintf(os.Stderr, "Commands:\n")
		fmt.Fprintf(os.Stderr, "  run <vector-dir>   Run golden vector suite\n")
		fmt.Fprintf(os.Stderr, "  verify <file>      Validate event file\n")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		runVectors()
	case "verify":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: ccnf-conformance verify <file>")
			os.Exit(1)
		}
		verifyFile(os.Args[2])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runVectors() {
	vectorDir := "vectors/v1"
	if len(os.Args) > 2 {
		vectorDir = os.Args[2]
	}

	entries, err := os.ReadDir(vectorDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read vector directory %s: %v\n", vectorDir, err)
		os.Exit(1)
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, e.Name())
		}
	}

	passed := 0
	failed := 0

	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(vectorDir, f))
		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: read error: %v\n", f, err)
			failed++
			continue
		}

		var vec struct {
			Name        string                 `json:"name"`
			CCNFVersion int                    `json:"ccnf_version"`
			Input       map[string]any         `json:"input"`
			Expected    map[string]any         `json:"expected"`
		}
		if err := json.Unmarshal(data, &vec); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: parse error: %v\n", f, err)
			failed++
			continue
		}

		inputJSON, _ := json.Marshal(vec.Input)
		cer, err := ccnf.Run(inputJSON, vec.CCNFVersion)

		errStr, hasError := vec.Expected["error"].(string)
		if hasError && errStr != "" {
			if err == nil {
				fmt.Fprintf(os.Stderr, "FAIL %s: expected error %q but got none\n", f, errStr)
				failed++
				continue
			}
			if strings.Contains(err.Error(), errStr) {
				fmt.Printf("PASS %s: got expected error %q\n", f, errStr)
				passed++
			} else {
				fmt.Fprintf(os.Stderr, "FAIL %s: expected error %q but got %q\n", f, errStr, err.Error())
				failed++
			}
			continue
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: unexpected error: %v\n", f, err)
			failed++
			continue
		}

		if entityKey, ok := vec.Expected["entity_key"].(string); ok && entityKey != "" {
			if cer.Identity.EntityKey != entityKey {
				fmt.Fprintf(os.Stderr, "FAIL %s: entity_key mismatch\n  want: %s\n  got:  %s\n", f, entityKey, cer.Identity.EntityKey)
				failed++
				continue
			}
		}

		if canHash, ok := vec.Expected["canonical_hash"].(string); ok && canHash != "" {
			actualHash := ccnf.ComputeHash(cer)
			if actualHash != canHash {
				fmt.Fprintf(os.Stderr, "FAIL %s: canonical_hash mismatch\n  want: %s\n  got:  %s\n", f, canHash, actualHash)
				failed++
				continue
			}
		}

		fmt.Printf("PASS %s\n", f)
		passed++
	}

	fmt.Printf("\n%d passed, %d failed\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func verifyFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to read %s: %v\n", path, err)
		os.Exit(1)
	}

	cer, err := ccnf.Run(data, ccnf.CurrentCCNFVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Validation FAILED: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Validation PASSED\n")
	fmt.Printf("  event_id:    %s\n", cer.EventID)
	fmt.Printf("  entity_key:  %s\n", cer.Identity.EntityKey)
	fmt.Printf("  domain:      %s\n", cer.Domain)
	fmt.Printf("  timestamp:   %d\n", cer.Timestamp)
	fmt.Printf("  signature:   %s\n", cer.Signature["hash"])
}

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
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") && e.Name() != "expected-hashes.json" {
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

		var raw map[string]any
		if err := json.Unmarshal(data, &raw); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: parse error: %v\n", f, err)
			failed++
			continue
		}

		ccnfVersion := 1
		if v, ok := raw["ccnf_version"].(float64); ok {
			ccnfVersion = int(v)
		}

		expected, _ := raw["expected"].(map[string]any)
		errStr, hasError := expected["error"].(string)

		inputs := []struct {
			label string
			data  map[string]any
		}{}
		if input, ok := raw["input"].(map[string]any); ok {
			inputs = append(inputs, struct {
				label string
				data  map[string]any
			}{"input", input})
		}
		if inputA, ok := raw["input_a"].(map[string]any); ok {
			inputs = append(inputs, struct {
				label string
				data  map[string]any
			}{"input_a", inputA})
		}
		if inputB, ok := raw["input_b"].(map[string]any); ok {
			inputs = append(inputs, struct {
				label string
				data  map[string]any
			}{"input_b", inputB})
		}

		if len(inputs) == 0 {
			fmt.Fprintf(os.Stderr, "FAIL %s: no input/input_a/input_b found\n", f)
			failed++
			continue
		}

		inputOK := true
		for _, in := range inputs {
			inputJSON, err := json.Marshal(in.data)
			if err != nil {
				fmt.Fprintf(os.Stderr, "FAIL %s (%s): marshal error: %v\n", f, in.label, err)
				failed++
				inputOK = false
				continue
			}

			cer, cerr := ccnf.Run(inputJSON, ccnfVersion)

			if hasError && errStr != "" {
				if cerr == nil {
					fmt.Fprintf(os.Stderr, "FAIL %s (%s): expected error %q but got none\n", f, in.label, errStr)
					failed++
					inputOK = false
					continue
				}
				if !strings.Contains(cerr.Error(), errStr) {
					fmt.Fprintf(os.Stderr, "FAIL %s (%s): expected error %q but got %q\n", f, in.label, errStr, cerr.Error())
					failed++
					inputOK = false
					continue
				}
				continue
			}

			if cerr != nil {
				fmt.Fprintf(os.Stderr, "FAIL %s (%s): unexpected error: %v\n", f, in.label, cerr)
				failed++
				inputOK = false
				continue
			}
			if cer == nil {
				fmt.Fprintf(os.Stderr, "FAIL %s (%s): expected non-nil CER\n", f, in.label)
				failed++
				inputOK = false
				continue
			}

			suffix := ""
			if in.label == "input_a" {
				suffix = "_a"
			} else if in.label == "input_b" {
				suffix = "_b"
			}
			expectedHash(expected, cer, "entity_key"+suffix, cer.Identity.EntityKey, f, in.label, &inputOK, &failed)
			expectedHash(expected, cer, "canonical_hash"+suffix, ccnf.ComputeHash(cer), f, in.label, &inputOK, &failed)
		}

		if inputOK {
			fmt.Printf("PASS %s\n", f)
			passed++
		}
	}

	fmt.Printf("\n%d passed, %d failed\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func expectedHash(expected map[string]any, cer *ccnf.CER, key, actual string, file, label string, ok *bool, failed *int) {
	want, exists := expected[key].(string)
	if !exists || want == "" {
		return
	}
	if actual != want {
		fmt.Fprintf(os.Stderr, "FAIL %s (%s): %s mismatch\n  want: %s\n  got:  %s\n", file, label, key, want, actual)
		*ok = false
		*failed++
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

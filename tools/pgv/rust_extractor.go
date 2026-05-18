package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type RustMappingConfig struct {
	CrateRoot string // "ccnf-verifier"
	Namespace string // "rust"
	SrcPath   string // "../../../rust/wrp/ccnf-verifier/src"
}

type RustExtractor struct {
	Config RustMappingConfig
}

func (e *RustExtractor) Name() string { return "rust" }

func (e *RustExtractor) Extract() ([]Node, error) {
	srcPath := e.Config.SrcPath
	info, err := os.Stat(srcPath)
	if err != nil {
		return nil, fmt.Errorf("rust src path %s: %w", srcPath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("rust src path %s is not a directory", srcPath)
	}

	moduleFiles := map[string]string{}
	moduleDeps := map[string][]string{}

	err = filepath.Walk(srcPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".rs") {
			return nil
		}

		rel, err := filepath.Rel(srcPath, path)
		if err != nil {
			return err
		}

		modPath := toRustModulePath(rel)
		fullMod := e.Config.Namespace + "/" + modPath

		deps := scanRustDeps(path, e.Config.Namespace)
		sort.Strings(deps)

		moduleFiles[fullMod] = path
		moduleDeps[fullMod] = deps

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk rust src: %w", err)
	}

	var nodes []Node
	for modPath, deps := range moduleDeps {
		name := modPath
		if idx := strings.LastIndex(modPath, "/"); idx >= 0 {
			name = modPath[idx+1:]
		}
		nodes = append(nodes, Node{
			ImportPath: modPath,
			Name:       name,
			DirectDeps: deps,
		})
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].ImportPath < nodes[j].ImportPath
	})

	return nodes, nil
}

func toRustModulePath(rel string) string {
	mod := filepath.ToSlash(rel)
	mod = strings.TrimSuffix(mod, ".rs")
	if strings.HasSuffix(mod, "/mod") {
		mod = strings.TrimSuffix(mod, "/mod")
	}
	return mod
}

func scanRustDeps(path, namespace string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	depSet := map[string]bool{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		raw := scanner.Text()
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "use crate::") {
			continue
		}
		rest := strings.TrimPrefix(line, "use crate::")
		if idx := strings.IndexAny(rest, " \t;{}"); idx >= 0 {
			rest = rest[:idx]
		}
		dep := strings.ReplaceAll(rest, "::", "/")
		depSet[namespace+"/"+dep] = true
	}

	var deps []string
	for d := range depSet {
		deps = append(deps, d)
	}
	sort.Strings(deps)
	return deps
}

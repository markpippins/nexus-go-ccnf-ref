package main

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type DeclaredDeps struct {
	Node     string
	Expected []string
}

type CommentExtractor struct{}

func (e *CommentExtractor) Name() string    { return "comment" }
func (e *CommentExtractor) Version() string { return "comment_extractor.v1" }

func (e *CommentExtractor) Extract() ([]Node, error) {
	return nil, nil
}

func (e *CommentExtractor) ExtractDeclared() ([]DeclaredDeps, error) {
	var result []DeclaredDeps

	err := filepath.Walk("projection", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}

		deps := scanDependsOn(path)
		if deps == nil {
			return nil
		}

		pkgPath := filepathToPackage(path)
		result = append(result, DeclaredDeps{
			Node:     pkgPath,
			Expected: deps,
		})

		return nil
	})

	return result, err
}

func scanDependsOn(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		raw := scanner.Text()
		trimmed := strings.TrimSpace(raw)
		if !strings.HasPrefix(trimmed, "// DependsOn:") {
			continue
		}
		rest := strings.TrimPrefix(trimmed, "// DependsOn:")
		rest = strings.TrimSpace(rest)

		if rest == "(no external deps)" {
			return []string{}
		}

		for _, part := range strings.Split(rest, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				deps = append(deps, part)
			}
		}
	}

	if len(deps) == 0 {
		return nil
	}

	sort.Strings(deps)
	return deps
}

func filepathToPackage(path string) string {
	dir := filepath.Dir(path)
	pkg := filepath.ToSlash(dir)
	if strings.HasPrefix(pkg, "projection") {
		pkg = "github.com/anomalyco/nexus-ccnf-ref/" + pkg
	}
	return pkg
}

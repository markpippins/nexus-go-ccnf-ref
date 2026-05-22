package main

import "strings"

type ViolationCode string

const (
	P6DepthExceeded   ViolationCode = "P6_DEPTH_EXCEEDED"
	P7CrossImport     ViolationCode = "P7_CROSS_IMPORT"
	P8Cycle           ViolationCode = "P8_CYCLE"
	P9Backflow        ViolationCode = "P9_BACKFLOW"
	P10aForbidden     ViolationCode = "P10A_FORBIDDEN_IMPORT"
	P10bUndeclared    ViolationCode = "P10B_UNDECLARED_DEP"
	P10bDeclaredVsAct ViolationCode = "P10B_DECLARED_VS_ACTUAL"
)

type Severity string

const SeverityError Severity = "ERROR"

type Violation struct {
	Code     ViolationCode
	Node     string
	Edge     string
	Message  string
	Severity Severity
}

type Config struct {
	KernelRoots   []string
	Allowlist     []string
	SchemaVersion string
}

type ValidationResult struct {
	Violations []Violation
	Depth      int
	HasCycles  bool
	Hash       string
}

func isProjection(path string) bool {
	return strings.HasPrefix(path, "projection/") || strings.HasPrefix(path, "rust/projection/")
}

func isKernel(path string, roots []string) bool {
	for _, r := range roots {
		if strings.HasPrefix(path, r) {
			return true
		}
	}
	return false
}

func (c *Config) isAllowed(path string) bool {
	for _, a := range c.Allowlist {
		if strings.HasPrefix(path, a) {
			return true
		}
	}
	return false
}

func projectionNodes(nodes []Node) []Node {
	var result []Node
	for _, n := range nodes {
		if isProjection(n.ImportPath) {
			result = append(result, n)
		}
	}
	return result
}

func projectionEdges(edges []Edge, projNodes []Node) []Edge {
	projSet := map[string]bool{}
	for _, n := range projNodes {
		projSet[n.ImportPath] = true
	}
	var result []Edge
	for _, e := range edges {
		if projSet[e.From] && projSet[e.To] {
			result = append(result, e)
		}
	}
	return result
}

func backflowEdges(edges []Edge, roots []string, projNodes []Node) []Edge {
	projSet := map[string]bool{}
	for _, n := range projNodes {
		projSet[n.ImportPath] = true
	}
	var result []Edge
	for _, e := range edges {
		if isKernel(e.From, roots) && projSet[e.To] {
			result = append(result, e)
		}
	}
	return result
}

func bfsMaxDepth(start string, adj map[string][]string, memo map[string]int) int {
	if d, ok := memo[start]; ok {
		return d
	}
	maxD := 0
	for _, next := range adj[start] {
		d := 1 + bfsMaxDepth(next, adj, memo)
		if d > maxD {
			maxD = d
		}
	}
	memo[start] = maxD
	return maxD
}

func hasCycleDFS(node string, adj map[string][]string, white, gray, black map[string]bool) bool {
	delete(white, node)
	gray[node] = true
	for _, next := range adj[node] {
		if black[next] {
			continue
		}
		if gray[next] {
			return true
		}
		if hasCycleDFS(next, adj, white, gray, black) {
			return true
		}
	}
	delete(gray, node)
	black[node] = true
	return false
}

func checkDepth(projEdges []Edge, projNodes []Node) (int, []Violation) {
	if len(projEdges) == 0 {
		return 0, nil
	}

	adj := map[string][]string{}
	for _, e := range projEdges {
		adj[e.From] = append(adj[e.From], e.To)
	}

	maxDepth := 0
	var violations []Violation

	for _, n := range projNodes {
		depth := bfsMaxDepth(n.ImportPath, adj, map[string]int{})
		if depth > maxDepth {
			maxDepth = depth
		}
		if depth > 1 {
			violations = append(violations, Violation{
				Code:     P6DepthExceeded,
				Node:     n.ImportPath,
				Message:  "depth " + itoa(depth) + " exceeds limit of 1",
				Severity: SeverityError,
			})
		}
	}

	return maxDepth, violations
}

func checkCrossImport(projEdges []Edge) []Violation {
	var violations []Violation
	for _, e := range projEdges {
		violations = append(violations, Violation{
			Code:     P7CrossImport,
			Node:     e.From,
			Edge:     e.From + " → " + e.To,
			Message:  "projection imports another projection: " + e.From + " → " + e.To,
			Severity: SeverityError,
		})
	}
	return violations
}

func checkCycles(projEdges []Edge, projNodes []Node) (bool, []Violation) {
	adj := map[string][]string{}
	for _, e := range projEdges {
		adj[e.From] = append(adj[e.From], e.To)
	}

	white := map[string]bool{}
	gray := map[string]bool{}
	black := map[string]bool{}

	for _, n := range projNodes {
		white[n.ImportPath] = true
	}

	hasCycle := false
	var violations []Violation

	for n := range white {
		if hasCycleDFS(n, adj, white, gray, black) {
			hasCycle = true
			violations = append(violations, Violation{
				Code:     P8Cycle,
				Node:     n,
				Message:  "cycle detected in projection dependency graph",
				Severity: SeverityError,
			})
			break
		}
	}

	return hasCycle, violations
}

func checkBackflow(bfEdges []Edge) []Violation {
	var violations []Violation
	for _, e := range bfEdges {
		violations = append(violations, Violation{
			Code:     P9Backflow,
			Node:     e.From,
			Edge:     e.From + " → " + e.To,
			Message:  "kernel package imports projection: " + e.From + " → " + e.To,
			Severity: SeverityError,
		})
	}
	return violations
}

func checkP10aForbidden(edges []Edge, roots []string, projNodes []Node) []Violation {
	projSet := map[string]bool{}
	for _, n := range projNodes {
		projSet[n.ImportPath] = true
	}
	var violations []Violation
	for _, e := range edges {
		if isKernel(e.From, roots) && projSet[e.To] {
			violations = append(violations, Violation{
				Code:     P10aForbidden,
				Node:     e.From,
				Edge:     e.From + " → " + e.To,
				Message:  "architectural law: kernel must not import projection: " + e.From + " → " + e.To,
				Severity: SeverityError,
			})
		}
	}
	return violations
}

func checkP10bAllowlist(projNodes []Node, cfg *Config) []Violation {
	var violations []Violation
	for _, n := range projNodes {
		for _, dep := range n.DirectDeps {
			if isKernel(dep, cfg.KernelRoots) {
				continue
			}
			if isProjection(dep) {
				continue
			}
			if cfg.isAllowed(dep) {
				continue
			}
			violations = append(violations, Violation{
				Code:     P10bUndeclared,
				Node:     n.ImportPath,
				Edge:     n.ImportPath + " → " + dep,
				Message:  "projection depends on non-allowlisted target: " + dep,
				Severity: SeverityError,
			})
		}
	}
	return violations
}

func checkP10bDeclared(projNodes []Node, declaredDeps []DeclaredDeps) []Violation {
	declared := map[string][]string{}
	for _, d := range declaredDeps {
		declared[d.Node] = d.Expected
	}

	var violations []Violation
	for _, n := range projNodes {
		if strings.HasPrefix(n.ImportPath, "rust/") {
			continue
		}

		expected, ok := declared[n.ImportPath]
		if !ok {
			violations = append(violations, Violation{
				Code:     P10bDeclaredVsAct,
				Node:     n.ImportPath,
				Message:  "projection package has no // DependsOn: annotation",
				Severity: SeverityError,
			})
			continue
		}

		got := n.DirectDeps
		if !stringSlicesEqual(got, expected) {
			violations = append(violations, Violation{
				Code:     P10bDeclaredVsAct,
				Node:     n.ImportPath,
				Message:  "declared deps mismatch: actual=" + joinStrings(got) + " expected=" + joinStrings(expected),
				Severity: SeverityError,
			})
		}
	}
	return violations
}

func stringSlicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func joinStrings(s []string) string {
	if len(s) == 0 {
		return "[]"
	}
	var b string
	b = "["
	for i, v := range s {
		if i > 0 {
			b += " "
		}
		b += v
	}
	b += "]"
	return b
}

func Validate(g *Graph, cfg *Config) *ValidationResult {
	return ValidateWithDeclared(g, cfg, nil)
}

func ValidateWithDeclared(g *Graph, cfg *Config, declaredDeps []DeclaredDeps) *ValidationResult {
	if cfg.SchemaVersion != "" && cfg.SchemaVersion != IrSchemaVersion {
		return &ValidationResult{
			Violations: []Violation{{
				Code:     "SCHEMA_MISMATCH",
				Message:  "IR schema version mismatch: config=" + cfg.SchemaVersion + " expected=" + IrSchemaVersion,
				Severity: SeverityError,
			}},
		}
	}

	projNodes := projectionNodes(g.Nodes)
	projEdges := projectionEdges(g.Edges, projNodes)
	bfEdges := backflowEdges(g.Edges, cfg.KernelRoots, projNodes)

	var allViolations []Violation

	p7violations := checkCrossImport(projEdges)
	allViolations = append(allViolations, p7violations...)

	hasCycle, p8violations := checkCycles(projEdges, projNodes)
	allViolations = append(allViolations, p8violations...)

	maxDepth, p6violations := checkDepth(projEdges, projNodes)
	allViolations = append(allViolations, p6violations...)

	p9violations := checkBackflow(bfEdges)
	allViolations = append(allViolations, p9violations...)

	p10aviolations := checkP10aForbidden(g.Edges, cfg.KernelRoots, projNodes)
	allViolations = append(allViolations, p10aviolations...)

	p10bviolations := checkP10bAllowlist(projNodes, cfg)
	allViolations = append(allViolations, p10bviolations...)

	if declaredDeps != nil {
		p10bdeclared := checkP10bDeclared(projNodes, declaredDeps)
		allViolations = append(allViolations, p10bdeclared...)
	}

	return &ValidationResult{
		Violations: allViolations,
		Depth:      maxDepth,
		HasCycles:  hasCycle,
		Hash:       g.Metadata.Hash,
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

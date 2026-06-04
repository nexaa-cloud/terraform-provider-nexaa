// schema-coverage: analyzes which terraform-plugin-framework schema attributes
// are exercised by acceptance tests, with modifier-aware coverage rules.
//
// Coverage rules per attribute modifier:
//   required            — covered if the name appears in any test config (it must be set)
//   optional            — covered if set in ≥1 config AND there exists ≥1 config for that
//                         resource type where it is absent
//   optional+computed   — same as optional, PLUS a tfjsonpath.New() assertion must exist
//   computed (only)     — covered if a tfjsonpath.New() assertion exists
//
// Usage:
//   go run ./tools/schema-coverage.go \
//     --resources ./internal/resources \
//     --tests     ./internal/tests
//
// Flags:
//   --resources   path to resource schema definitions (default: ./internal/resources)
//   --tests       path to acceptance test files      (default: ./internal/tests)
//   --threshold   fail if overall coverage < N%      (default: 0, disabled)
//   --format      output format: text|json           (default: text)

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// ---- data model ------------------------------------------------------------

type Attribute struct {
	Name     string
	Type     string
	Required bool
	Optional bool
	Computed bool
	Path     string // dotted path, e.g. "external_connection.ports.internal_port"
}

type CoverageStatus int

const (
	StatusUncovered    CoverageStatus = iota // not seen in any test
	StatusSetOnly                            // optional: set but never omitted
	StatusOmitOnly                           // optional: omitted but never set
	StatusNoStateCheck                       // optional+computed or computed: missing assertion
	StatusCovered                            // fully covered
)

func (s CoverageStatus) IsCovered() bool { return s == StatusCovered }

type AttributeCoverage struct {
	Attr   Attribute
	Status CoverageStatus
	Hint   string
}

type ResourceCoverage struct {
	Resource   string
	Attributes []AttributeCoverage
}

func (rc *ResourceCoverage) CoveredCount() int {
	n := 0
	for _, ac := range rc.Attributes {
		if ac.Status.IsCovered() {
			n++
		}
	}
	return n
}

func (rc *ResourceCoverage) Percent() float64 {
	total := len(rc.Attributes)
	if total == 0 {
		return 100
	}
	return float64(rc.CoveredCount()) / float64(total) * 100
}

// ---- schema extraction -----------------------------------------------------

var knownAttrTypes = map[string]bool{
	"StringAttribute":       true,
	"BoolAttribute":         true,
	"Int64Attribute":        true,
	"Float64Attribute":      true,
	"NumberAttribute":       true,
	"ListAttribute":         true,
	"SetAttribute":          true,
	"MapAttribute":          true,
	"ObjectAttribute":       true,
	"ListNestedAttribute":   true,
	"SetNestedAttribute":    true,
	"MapNestedAttribute":    true,
	"SingleNestedAttribute": true,
	"DynamicAttribute":      true,
}

// extractAttributes walks resourcesDir and returns resource-name → []Attribute.
// Uses go/parser directly per file — no deprecated ast.Package.
func extractAttributes(resourcesDir string) (map[string][]Attribute, error) {
	fset := token.NewFileSet()
	result := map[string][]Attribute{}

	err := filepath.WalkDir(resourcesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil // skip unparseable files
		}
		name := inferResourceName(path)
		attrs := collectAttrsFromFile(f)
		if len(attrs) > 0 {
			result[name] = append(result[name], attrs...)
		}
		return nil
	})
	return result, err
}

func inferResourceName(filePath string) string {
	base := filepath.Base(filePath)
	base = strings.TrimSuffix(base, ".go")
	for _, suffix := range []string{"_resource", "_data_source", "_schema"} {
		base = strings.TrimSuffix(base, suffix)
	}
	return base
}

// collectAttrsFromFile finds top-level map[string]schema.Attribute literals
// and collects attributes. Returns false from the inspector once a map is
// handled so nested maps are only visited through our own recursion.
func collectAttrsFromFile(f *ast.File) []Attribute {
	var attrs []Attribute
	ast.Inspect(f, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		if !isStringKeyedMap(cl) {
			return true
		}
		attrs = append(attrs, collectAttrsFromMap(cl, "")...)
		return false
	})
	return attrs
}

// collectAttrsFromMap processes one map[string]schema.Attribute literal.
// parentPath is the dotted prefix ("" at root, "timeouts" one level down …).
func collectAttrsFromMap(cl *ast.CompositeLit, parentPath string) []Attribute {
	var attrs []Attribute
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		key := stringLiteral(kv.Key)
		if key == "" {
			continue
		}
		attrType := resolveAttrType(kv.Value)
		if attrType == "" {
			continue
		}
		path := key
		if parentPath != "" {
			path = parentPath + "." + key
		}
		attr := Attribute{Name: key, Type: attrType, Path: path}
		fillModifiers(&attr, kv.Value)
		attrs = append(attrs, attr)
		attrs = append(attrs, collectNestedAttrs(kv.Value, path)...)
	}
	return attrs
}

// collectNestedAttrs walks one attribute's value expression looking for
// Attributes: or NestedObject: fields that contain another map.
// It never uses ast.Inspect to avoid re-visiting maps already handled above.
func collectNestedAttrs(expr ast.Expr, parentPath string) []Attribute {
	cl := unwrapCompositeLit(expr)
	if cl == nil {
		return nil
	}
	var attrs []Attribute
	for _, elt := range cl.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		switch identName(kv.Key) {
		case "Attributes":
			inner := unwrapCompositeLit(kv.Value)
			if inner != nil && isStringKeyedMap(inner) {
				attrs = append(attrs, collectAttrsFromMap(inner, parentPath)...)
			}
		case "NestedObject":
			// NestedObject is a struct (schema.NestedAttributeObject) that
			// itself contains an Attributes field — recurse one more level.
			attrs = append(attrs, collectNestedAttrs(kv.Value, parentPath)...)
		}
	}
	return attrs
}

func unwrapCompositeLit(expr ast.Expr) *ast.CompositeLit {
	switch e := expr.(type) {
	case *ast.CompositeLit:
		return e
	case *ast.UnaryExpr:
		if cl, ok := e.X.(*ast.CompositeLit); ok {
			return cl
		}
	}
	return nil
}

func isStringKeyedMap(cl *ast.CompositeLit) bool {
	mt, ok := cl.Type.(*ast.MapType)
	if !ok {
		return false
	}
	id, ok := mt.Key.(*ast.Ident)
	return ok && id.Name == "string"
}

func resolveAttrType(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.CallExpr:
		return selectorOrIdent(e.Fun)
	case *ast.CompositeLit:
		return selectorOrIdent(e.Type)
	case *ast.UnaryExpr:
		return resolveAttrType(e.X)
	}
	return ""
}

func selectorOrIdent(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.SelectorExpr:
		if knownAttrTypes[e.Sel.Name] {
			return e.Sel.Name
		}
	case *ast.Ident:
		if knownAttrTypes[e.Name] {
			return e.Name
		}
	}
	return ""
}

func fillModifiers(attr *Attribute, expr ast.Expr) {
	ast.Inspect(expr, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, elt := range cl.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			switch identName(kv.Key) {
			case "Required":
				attr.Required = boolLiteral(kv.Value)
			case "Optional":
				attr.Optional = boolLiteral(kv.Value)
			case "Computed":
				attr.Computed = boolLiteral(kv.Value)
			}
		}
		return true
	})
}

// ---- test scanning ---------------------------------------------------------

// ConfigBlock represents one parsed resource "type" "name" { … } block.
type ConfigBlock struct {
	ResourceType string
	Attrs        map[string]bool // leaf attribute names present in this block
}

// TestEvidence holds everything extracted from test files.
type TestEvidence struct {
	// ConfigBlocks is the list of every HCL resource block found across all tests.
	// Used to determine set-path and omit-path per resource type.
	ConfigBlocks []ConfigBlock

	// SetInConfig: all attribute leaf names / paths seen set in any config.
	SetInConfig map[string]bool

	// StateChecked: attr names referenced via tfjsonpath.New("x").
	StateChecked map[string]bool
}

// seenSet returns true if attr was set in at least one config block.
func (ev *TestEvidence) seenSet(leafName, path string) bool {
	return ev.SetInConfig[leafName] || ev.SetInConfig[path]
}

// seenOmitted returns true if there is at least one config block for
// resourceType that does NOT contain leafName.
func (ev *TestEvidence) seenOmitted(resourceType, leafName string) bool {
	for _, b := range ev.ConfigBlocks {
		if b.ResourceType == resourceType && !b.Attrs[leafName] {
			return true
		}
	}
	return false
}

func scanTests(testsDir string) (*TestEvidence, error) {
	ev := &TestEvidence{
		SetInConfig:  map[string]bool{},
		StateChecked: map[string]bool{},
	}

	err := filepath.WalkDir(testsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.HasSuffix(path, ".go") {
			_ = scanGoFile(path, ev)
		}
		return nil
	})
	return ev, err
}

func scanGoFile(path string, ev *TestEvidence) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return err
	}

	// 1. Walk all string literals (catches backtick HCL configs).
	ast.Inspect(f, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		s, _ := strconv.Unquote(lit.Value)
		collectHCLTokens(s, ev.SetInConfig)
		parseConfigBlocks(s, &ev.ConfigBlocks)
		detectStateChecks(s, ev.StateChecked)
		return true
	})

	// 2. Walk call expressions for all known state-check patterns.
	ast.Inspect(f, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		fnName := ""
		switch fn := call.Fun.(type) {
		case *ast.SelectorExpr:
			fnName = fn.Sel.Name
		case *ast.Ident:
			fnName = fn.Name
		}

		switch fnName {
		// tfjsonpath.New("attr")
		case "New":
			if len(call.Args) == 1 {
				if arg := stringLiteral(call.Args[0]); arg != "" {
					ev.StateChecked[arg] = true
				}
			}

		// resource.TestCheckResourceAttrSet("res.name", "attr")
		// resource.TestCheckResourceAttr("res.name", "attr", "value")
		case "TestCheckResourceAttrSet", "TestCheckResourceAttr",
			"TestCheckResourceAttrPair", "TestCheckResourceAttrWith":
			// second argument is always the attribute path
			if len(call.Args) >= 2 {
				if arg := stringLiteral(call.Args[1]); arg != "" {
					ev.StateChecked[arg] = true
					// also record the leaf segment after the last dot
					// e.g. "timeouts.0.create" -> also record "create"
					if dot := strings.LastIndex(arg, "."); dot >= 0 {
						ev.StateChecked[arg[dot+1:]] = true
					}
				}
			}

		// statecheck.ExpectKnownValue / ExpectKnownOutputValue
		case "ExpectKnownValue", "ExpectKnownOutputValue":
			if len(call.Args) >= 2 {
				if arg := stringLiteral(call.Args[1]); arg != "" {
					ev.StateChecked[arg] = true
				}
			}
		}

		return true
	})

	return nil
}

// collectHCLTokens tokenises an HCL string and records every word in dst.
func collectHCLTokens(s string, dst map[string]bool) {
	parts := splitHCL(s)
	for _, p := range parts {
		dst[p] = true
		for _, seg := range strings.Split(p, ".") {
			dst[seg] = true
		}
	}
}

// parseConfigBlocks parses resource "type" "name" { … } blocks in an HCL
// string and appends a ConfigBlock for each one to *blocks.
// Attrs records both leaf names AND full dotted paths so nested attributes
// like external_connection.ports.internal_port are tracked correctly.
func parseConfigBlocks(s string, blocks *[]ConfigBlock) {
	type frame struct {
		label string // identifier that opened this brace level
		path  string // dotted path down to this level
	}
	type outerFrame struct {
		resourceType string
		attrs        map[string]bool
		stack        []frame
	}

	var current *outerFrame

	addAttr := func(name string) {
		if current == nil || !isIdentifier(name) {
			return
		}
		current.attrs[name] = true
		if len(current.stack) > 0 {
			top := current.stack[len(current.stack)-1]
			if top.path != "" {
				current.attrs[top.path+"."+name] = true
			}
		}
	}

	lines := strings.Split(s, "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") {
			continue
		}

		opens := strings.Count(line, "{")
		closes := strings.Count(line, "}")

		// start of a resource block
		if current == nil && strings.HasPrefix(line, "resource ") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				rtype := strings.Trim(fields[1], `"`)
				current = &outerFrame{resourceType: rtype, attrs: map[string]bool{}}
				if opens > closes {
					current.stack = append(current.stack, frame{label: "", path: ""})
				}
			}
			continue
		}

		if current == nil {
			continue
		}

		// attribute assignment: name = value (no braces on this line)
		if eqIdx := strings.Index(line, "="); eqIdx > 0 && opens == 0 && closes == 0 {
			name := strings.TrimSpace(line[:eqIdx])
			addAttr(name)
			continue
		}

		// nested block opener: label { or label "alias" {
		if opens > 0 {
			fields := strings.Fields(strings.NewReplacer("{", " ", "}", " ").Replace(line))
			if len(fields) > 0 {
				label := strings.Trim(fields[0], `"`)
				if isIdentifier(label) && label != "resource" && label != "data" {
					parentPath := ""
					if len(current.stack) > 0 {
						parentPath = current.stack[len(current.stack)-1].path
					}
					path := label
					if parentPath != "" {
						path = parentPath + "." + label
					}
					addAttr(label)
					for i := 0; i < opens-closes; i++ {
						current.stack = append(current.stack, frame{label: label, path: path})
					}
				}
			}
			continue
		}

		// closing braces only
		if closes > 0 {
			for i := 0; i < closes && len(current.stack) > 0; i++ {
				current.stack = current.stack[:len(current.stack)-1]
			}
			if len(current.stack) == 0 {
				*blocks = append(*blocks, ConfigBlock{
					ResourceType: current.resourceType,
					Attrs:        current.attrs,
				})
				current = nil
			}
		}
	}
}
func splitHCL(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' ||
			r == '=' || r == '{' || r == '}' || r == '"' || r == '\'' ||
			r == ',' || r == '(' || r == ')' || r == '#'
	})
}

func detectStateChecks(s string, dst map[string]bool) {
	const needle = `tfjsonpath.New(`
	rest := s
	for {
		pos := strings.Index(rest, needle)
		if pos < 0 {
			break
		}
		rest = rest[pos+len(needle):]
		end := strings.Index(rest, ")")
		if end < 0 {
			break
		}
		arg := strings.Trim(rest[:end], `"' `)
		if arg != "" {
			dst[arg] = true
		}
		rest = rest[end+1:]
	}
}

func isIdentifier(s string) bool {
	if s == "" {
		return false
	}
	for i, r := range s {
		if i == 0 {
			if !('a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || r == '_') {
				return false
			}
		} else {
			if !('a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9' || r == '_') {
				return false
			}
		}
	}
	return true
}

// ---- coverage evaluation ---------------------------------------------------

func evaluate(attr Attribute, resourceType string, ev *TestEvidence) (CoverageStatus, string) {
	set := ev.seenSet(attr.Name, attr.Path)
	omitted := ev.seenOmitted(resourceType, attr.Name)
	checked := ev.StateChecked[attr.Name] || ev.StateChecked[attr.Path]

	switch {
	case attr.Required && !attr.Optional && !attr.Computed:
		if set {
			return StatusCovered, ""
		}
		return StatusUncovered, "never set in any test config"

	case attr.Optional && attr.Computed:
		var missing []string
		if !set {
			missing = append(missing, "never set in config")
		}
		if !omitted {
			missing = append(missing, "never omitted in config (omit-path not tested)")
		}
		if !checked {
			missing = append(missing, "no tfjsonpath.New() assertion")
		}
		if len(missing) == 0 {
			return StatusCovered, ""
		}
		if !checked {
			return StatusNoStateCheck, strings.Join(missing, "; ")
		}
		if set && !omitted {
			return StatusSetOnly, strings.Join(missing, "; ")
		}
		return StatusUncovered, strings.Join(missing, "; ")

	case attr.Optional && !attr.Computed:
		if set && omitted {
			return StatusCovered, ""
		}
		if set {
			return StatusSetOnly, "never omitted in config (omit-path not tested)"
		}
		if omitted {
			return StatusOmitOnly, "never set in any test config"
		}
		return StatusUncovered, "never set or omitted in any test config"

	case !attr.Optional && attr.Computed:
		if checked {
			return StatusCovered, ""
		}
		return StatusNoStateCheck, "no tfjsonpath.New() assertion for computed value"

	default:
		if set {
			return StatusCovered, ""
		}
		return StatusUncovered, "never seen in any test config"
	}
}

// ---- helpers ----------------------------------------------------------------

func stringLiteral(expr ast.Expr) string {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return ""
	}
	s, _ := strconv.Unquote(lit.Value)
	return s
}

func identName(expr ast.Expr) string {
	id, ok := expr.(*ast.Ident)
	if !ok {
		return ""
	}
	return id.Name
}

func boolLiteral(expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return id.Name == "true"
}

// ---- report -----------------------------------------------------------------

type JSONReport struct {
	Overall   float64              `json:"overall_percent"`
	Resources []JSONResourceReport `json:"resources"`
}

type JSONResourceReport struct {
	Resource string          `json:"resource"`
	Percent  float64         `json:"percent"`
	Covered  int             `json:"covered"`
	Total    int             `json:"total"`
	Gaps     []JSONGapReport `json:"gaps,omitempty"`
}

type JSONGapReport struct {
	Path   string `json:"path"`
	Type   string `json:"type"`
	Status string `json:"status"`
	Hint   string `json:"hint"`
}

func statusLabel(s CoverageStatus) string {
	switch s {
	case StatusSetOnly:
		return "set-only"
	case StatusOmitOnly:
		return "omit-only"
	case StatusNoStateCheck:
		return "no-state-check"
	case StatusUncovered:
		return "uncovered"
	}
	return "covered"
}

func modifierStr(attr Attribute) string {
	var parts []string
	if attr.Required {
		parts = append(parts, "required")
	}
	if attr.Optional {
		parts = append(parts, "optional")
	}
	if attr.Computed {
		parts = append(parts, "computed")
	}
	return strings.Join(parts, "+")
}

func printText(resources []*ResourceCoverage, overall float64, threshold int) int {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              Terraform Schema Coverage Report                   ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	for _, rc := range resources {
		pct := rc.Percent()
		bar := progressBar(pct, 30)
		// Show ✗ whenever there are actual gaps, regardless of threshold
		hasGaps := rc.CoveredCount() < len(rc.Attributes)
		symbol := "✓"
		if hasGaps {
			symbol = "✗"
		}
		fmt.Printf("  %s %-38s %s  %5.1f%%  (%d/%d)\n",
			symbol, rc.Resource, bar, pct, rc.CoveredCount(), len(rc.Attributes))

		for _, ac := range rc.Attributes {
			if ac.Status.IsCovered() {
				continue
			}
			fmt.Printf("      ✗ %-42s [%s] %s\n",
				ac.Attr.Path, ac.Attr.Type, modifierStr(ac.Attr))
			fmt.Printf("        → %s\n", ac.Hint)
		}
		fmt.Println()
	}

	fmt.Printf("  Overall coverage: %.1f%%\n", overall)
	if threshold > 0 && overall < float64(threshold) {
		fmt.Printf("  FAILED: below threshold of %d%%\n\n", threshold)
		return 1
	}
	fmt.Println()
	return 0
}

func progressBar(pct float64, width int) string {
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}

// ---- main -------------------------------------------------------------------

func main() {
	resourcesDir := flag.String("resources", "./internal/resources", "path to resource schema definitions")
	testsDir := flag.String("tests", "./internal/tests", "path to acceptance test files")
	threshold := flag.Int("threshold", 0, "fail if overall coverage is below this percentage (0 = disabled)")
	format := flag.String("format", "text", "output format: text or json")
	flag.Parse()

	schemaAttrs, err := extractAttributes(*resourcesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading resources: %v\n", err)
		os.Exit(2)
	}
	if len(schemaAttrs) == 0 {
		fmt.Fprintf(os.Stderr, "no schema attributes found under %s\n", *resourcesDir)
		os.Exit(2)
	}

	ev, err := scanTests(*testsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading tests: %v\n", err)
		os.Exit(2)
	}

	var resources []*ResourceCoverage
	totalAttrs, totalCovered := 0, 0

	sortedNames := make([]string, 0, len(schemaAttrs))
	for name := range schemaAttrs {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	for _, name := range sortedNames {
		rc := &ResourceCoverage{Resource: name}
		for _, attr := range schemaAttrs[name] {
			status, hint := evaluate(attr, name, ev)
			rc.Attributes = append(rc.Attributes, AttributeCoverage{
				Attr: attr, Status: status, Hint: hint,
			})
			totalAttrs++
			if status.IsCovered() {
				totalCovered++
			}
		}
		resources = append(resources, rc)
	}

	overall := 0.0
	if totalAttrs > 0 {
		overall = float64(totalCovered) / float64(totalAttrs) * 100
	}

	exitCode := 0
	if *format == "json" {
		report := JSONReport{Overall: overall}
		for _, rc := range resources {
			jr := JSONResourceReport{
				Resource: rc.Resource,
				Percent:  rc.Percent(),
				Covered:  rc.CoveredCount(),
				Total:    len(rc.Attributes),
			}
			for _, ac := range rc.Attributes {
				if !ac.Status.IsCovered() {
					jr.Gaps = append(jr.Gaps, JSONGapReport{
						Path:   ac.Attr.Path,
						Type:   ac.Attr.Type,
						Status: statusLabel(ac.Status),
						Hint:   ac.Hint,
					})
				}
			}
			report.Resources = append(report.Resources, jr)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
		if *threshold > 0 && overall < float64(*threshold) {
			exitCode = 1
		}
	} else {
		exitCode = printText(resources, overall, *threshold)
	}

	os.Exit(exitCode)
}
// schema-coverage: analyzes which terraform-plugin-framework schema attributes
// are exercised by acceptance tests.
//
// Usage:
//   go run ./schema-coverage \
//     --resources ./internal/resources \
//     --tests    ./internal/tests
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
	Type     string // e.g. StringAttribute, BoolAttribute, ListNestedAttribute …
	Required bool
	Optional bool
	Computed bool
	Path     string // dotted path for nested attrs, e.g. "timeouts.create"
}

type ResourceCoverage struct {
	Resource   string
	Attributes []Attribute
	Tested     map[string]bool // keyed by Attribute.Path
}

func (rc *ResourceCoverage) CoveredCount() int {
	n := 0
	for _, covered := range rc.Tested {
		if covered {
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

// knownAttrTypes are the terraform-plugin-framework schema.XxxAttribute constructors.
var knownAttrTypes = map[string]bool{
	"StringAttribute":      true,
	"BoolAttribute":        true,
	"Int64Attribute":       true,
	"Float64Attribute":     true,
	"NumberAttribute":      true,
	"ListAttribute":        true,
	"SetAttribute":         true,
	"MapAttribute":         true,
	"ObjectAttribute":      true,
	"ListNestedAttribute":  true,
	"SetNestedAttribute":   true,
	"MapNestedAttribute":   true,
	"SingleNestedAttribute": true,
	"DynamicAttribute":     true,
}

// extractAttributes walks Go AST files under resourcesDir and returns a map of
// resource-name → []Attribute.
func extractAttributes(resourcesDir string) (map[string][]Attribute, error) {
	fset := token.NewFileSet()
	pkgs := map[string]*ast.Package{}

	err := filepath.WalkDir(resourcesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		f, parseErr := parser.ParseFile(fset, path, nil, 0)
		if parseErr != nil {
			return nil // skip unparseable files
		}
		pkg := f.Name.Name
		if _, ok := pkgs[pkg+"@"+filepath.Dir(path)]; !ok {
			pkgs[pkg+"@"+filepath.Dir(path)] = &ast.Package{Name: pkg, Files: map[string]*ast.File{}}
		}
		pkgs[pkg+"@"+filepath.Dir(path)].Files[path] = f
		return nil
	})
	if err != nil {
		return nil, err
	}

	result := map[string][]Attribute{}

	for _, pkg := range pkgs {
		for filePath, f := range pkg.Files {
			resourceName := inferResourceName(filePath)
			attrs := collectAttrsFromFile(f)
			if len(attrs) > 0 {
				result[resourceName] = append(result[resourceName], attrs...)
			}
		}
	}

	return result, nil
}

// inferResourceName derives a human-readable resource name from file path.
func inferResourceName(filePath string) string {
	base := filepath.Base(filePath)
	base = strings.TrimSuffix(base, ".go")
	// strip common suffixes
	for _, suffix := range []string{"_resource", "_data_source", "_schema"} {
		base = strings.TrimSuffix(base, suffix)
	}
	return base
}

// collectAttrsFromFile finds the top-level map[string]schema.Attribute literals
// in the file and delegates to collectAttrsFromMap for each one.
// We stop ast.Inspect from recursing into any map we handle ourselves so that
// nested maps are only visited once, at the correct dotted path depth.
func collectAttrsFromFile(f *ast.File) []Attribute {
	var attrs []Attribute

	ast.Inspect(f, func(n ast.Node) bool {
		compLit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		if !isStringKeyedMap(compLit) {
			return true
		}
		// Collect top-level attrs from this map; nested maps are handled
		// recursively inside collectAttrsFromMap, so we must NOT let
		// ast.Inspect descend further (return false).
		attrs = append(attrs, collectAttrsFromMap(compLit, "")...)
		return false
	})

	return attrs
}

// collectAttrsFromMap processes one map[string]schema.Attribute composite
// literal. parentPath is the dotted prefix for this level ("" at root,
// "timeouts" one level down, "external_connection.ports" two levels down).
func collectAttrsFromMap(compLit *ast.CompositeLit, parentPath string) []Attribute {
	var attrs []Attribute

	for _, elt := range compLit.Elts {
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

		// Look one level deeper: find any Attributes or NestedObject field
		// that contains another map[string]schema.Attribute.
		attrs = append(attrs, collectNestedAttrs(kv.Value, path)...)
	}

	return attrs
}

// collectNestedAttrs finds the next map[string]schema.Attribute inside an
// attribute definition (via an Attributes: or NestedObject: field) and
// recurses through collectAttrsFromMap. It does NOT use ast.Inspect so it
// never accidentally double-visits any map.
func collectNestedAttrs(expr ast.Expr, parentPath string) []Attribute {
	compLit := unwrapCompositeLit(expr)
	if compLit == nil {
		return nil
	}

	var attrs []Attribute
	for _, elt := range compLit.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		fieldName := identName(kv.Key)

		switch fieldName {
		case "Attributes":
			// Direct map[string]schema.Attribute value.
			inner := unwrapCompositeLit(kv.Value)
			if inner != nil && isStringKeyedMap(inner) {
				attrs = append(attrs, collectAttrsFromMap(inner, parentPath)...)
			}

		case "NestedObject":
			// NestedObject is a struct (e.g. schema.NestedAttributeObject) that
			// itself has an Attributes field — recurse one more level.
			attrs = append(attrs, collectNestedAttrs(kv.Value, parentPath)...)
		}
	}
	return attrs
}

// unwrapCompositeLit resolves &T{...} and T{...} to the inner *ast.CompositeLit.
func unwrapCompositeLit(expr ast.Expr) *ast.CompositeLit {
	switch e := expr.(type) {
	case *ast.CompositeLit:
		return e
	case *ast.UnaryExpr: // &T{...}
		if cl, ok := e.X.(*ast.CompositeLit); ok {
			return cl
		}
	}
	return nil
}

// isStringKeyedMap reports whether a composite literal has type map[string]...
func isStringKeyedMap(cl *ast.CompositeLit) bool {
	mt, ok := cl.Type.(*ast.MapType)
	if !ok {
		return false
	}
	id, ok := mt.Key.(*ast.Ident)
	return ok && id.Name == "string"
}

// resolveAttrType returns e.g. "StringAttribute" if the expression is a call
// or composite literal constructing one.
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
		compLit, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		for _, elt := range compLit.Elts {
			kv, ok := elt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			field := identName(kv.Key)
			val := boolLiteral(kv.Value)
			switch field {
			case "Required":
				attr.Required = val
			case "Optional":
				attr.Optional = val
			case "Computed":
				attr.Computed = val
			}
		}
		return true
	})
}

// ---- test scanning ---------------------------------------------------------

// scanTests walks testsDir for .go and .tf / .tftest.hcl files and returns
// all attribute name tokens mentioned.
func scanTests(testsDir string) (map[string]bool, error) {
	mentioned := map[string]bool{}

	err := filepath.WalkDir(testsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		switch {
		case strings.HasSuffix(path, ".go"):
			tokens, e := tokenizeGoFile(path)
			if e == nil {
				for _, t := range tokens {
					mentioned[t] = true
				}
			}
		case strings.HasSuffix(path, ".tf"),
			strings.HasSuffix(path, ".tftest.hcl"),
			strings.HasSuffix(path, ".hcl"):
			tokens, e := tokenizeTextFile(path)
			if e == nil {
				for _, t := range tokens {
					mentioned[t] = true
				}
			}
		}
		return nil
	})

	return mentioned, err
}

// tokenizeGoFile extracts string literals and identifier names from a Go file.
func tokenizeGoFile(path string) ([]string, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, err
	}

	var tokens []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch v := n.(type) {
		case *ast.BasicLit:
			if v.Kind == token.STRING {
				s, _ := strconv.Unquote(v.Value)
				tokens = append(tokens, s)
				// split on whitespace, =, {, }, ", dots so embedded HCL configs
				// inside backtick strings have their attribute names extracted
				parts := strings.FieldsFunc(s, func(r rune) bool {
					return r == ' ' || r == '\t' || r == '\n' || r == '\r' ||
						r == '=' || r == '{' || r == '}' || r == '"' || r == '\'' ||
						r == ',' || r == '(' || r == ')' || r == '#'
				})
				tokens = append(tokens, parts...)
				// also split on dots for paths like "timeouts.create"
				for _, p := range parts {
					tokens = append(tokens, strings.Split(p, ".")...)
				}
			}
		case *ast.Ident:
			tokens = append(tokens, v.Name)
		}
		return true
	})
	return tokens, nil
}

// tokenizeTextFile splits a text file into word tokens.
func tokenizeTextFile(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	// split on whitespace, =, {, }, ", \n
	raw := strings.FieldsFunc(string(data), func(r rune) bool {
		return r == ' ' || r == '\t' || r == '\n' || r == '\r' ||
			r == '=' || r == '{' || r == '}' || r == '"' || r == '\'' ||
			r == ',' || r == '(' || r == ')' || r == '#'
	})
	return raw, nil
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
	Resource  string          `json:"resource"`
	Percent   float64         `json:"percent"`
	Covered   int             `json:"covered"`
	Total     int             `json:"total"`
	Untested  []AttributeInfo `json:"untested_attributes"`
}

type AttributeInfo struct {
	Path string `json:"path"`
	Type string `json:"type"`
}

func printText(resources []*ResourceCoverage, overall float64, threshold int) int {
	fmt.Println()
	fmt.Println("╔══════════════════════════════════════════════════════════════════╗")
	fmt.Println("║              Terraform Schema Coverage Report                    ║")
	fmt.Println("╚══════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	for _, rc := range resources {
		pct := rc.Percent()
		bar := progressBar(pct, 30)
		symbol := "✓"
		if pct < float64(threshold) {
			symbol = "✗"
		}
		fmt.Printf("  %s %-35s %s  %5.1f%%  (%d/%d)\n",
			symbol, rc.Resource, bar, pct, rc.CoveredCount(), len(rc.Attributes))

		// list untested attributes
		for _, attr := range rc.Attributes {
			if !rc.Tested[attr.Path] {
				modifier := modifierStr(attr)
				fmt.Printf("      ✗ %-40s [%s] %s\n", attr.Path, attr.Type, modifier)
			}
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
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	return "[" + bar + "]"
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
	return strings.Join(parts, ",")
}

// ---- main -------------------------------------------------------------------

func main() {
	resourcesDir := flag.String("resources", "./internal/resources", "path to resource schema definitions")
	testsDir := flag.String("tests", "./internal/tests", "path to acceptance test files")
	threshold := flag.Int("threshold", 0, "fail if overall coverage is below this percentage (0 = disabled)")
	format := flag.String("format", "text", "output format: text or json")
	flag.Parse()

	// 1. Extract schema attributes
	schemaAttrs, err := extractAttributes(*resourcesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading resources: %v\n", err)
		os.Exit(2)
	}
	if len(schemaAttrs) == 0 {
		fmt.Fprintf(os.Stderr, "no schema attributes found under %s\n", *resourcesDir)
		os.Exit(2)
	}

	// 2. Scan tests for mentioned tokens
	mentioned, err := scanTests(*testsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading tests: %v\n", err)
		os.Exit(2)
	}

	// 3. Build coverage per resource
	var resources []*ResourceCoverage
	totalAttrs, totalCovered := 0, 0

	sortedNames := make([]string, 0, len(schemaAttrs))
	for name := range schemaAttrs {
		sortedNames = append(sortedNames, name)
	}
	sort.Strings(sortedNames)

	for _, name := range sortedNames {
		attrs := schemaAttrs[name]
		rc := &ResourceCoverage{
			Resource:   name,
			Attributes: attrs,
			Tested:     map[string]bool{},
		}
		for _, attr := range attrs {
			// check both the leaf name and the full dotted path
			covered := mentioned[attr.Name] || mentioned[attr.Path]
			rc.Tested[attr.Path] = covered
			if covered {
				totalCovered++
			}
			totalAttrs++
		}
		resources = append(resources, rc)
	}

	overall := 0.0
	if totalAttrs > 0 {
		overall = float64(totalCovered) / float64(totalAttrs) * 100
	}

	// 4. Output
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
			for _, attr := range rc.Attributes {
				if !rc.Tested[attr.Path] {
					jr.Untested = append(jr.Untested, AttributeInfo{Path: attr.Path, Type: attr.Type})
				}
			}
			report.Resources = append(report.Resources, jr)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(report)
		if *threshold > 0 && overall < float64(*threshold) {
			exitCode = 1
		}
	} else {
		exitCode = printText(resources, overall, *threshold)
	}

	os.Exit(exitCode)
}
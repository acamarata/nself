package migrationai

import (
	"regexp"
	"strings"
)

// TSField is a single field extracted from a TypeScript interface or type alias.
type TSField struct {
	Name     string
	Type     string
	Optional bool // true when the declaration uses `name?: type`
}

// TSInterface represents a parsed TypeScript interface and its fields.
type TSInterface struct {
	Name   string
	Fields []TSField
}

// TypeScript interface parser is implemented without an external binary so the CLI
// has no Node.js dependency at generation time. A regexp-based heuristic covers the
// common subset of TypeScript interface and type-alias syntax seen in nSelf model files.
//
// Production watch mode (nself migration watch) invokes ts-morph via a bundled Node
// subprocess for full fidelity. This package is used by generate subcommand which
// accepts a path and performs a best-effort parse of one file.

var (
	// tsInterfaceHeader matches `export interface Foo {` or `interface Foo {`
	tsInterfaceHeader = regexp.MustCompile(`(?m)^\s*(?:export\s+)?interface\s+(\w+)(?:\s+extends\s+[^{]+)?\s*\{`)

	// tsTypeAliasHeader matches `export type Foo = {`
	tsTypeAliasHeader = regexp.MustCompile(`(?m)^\s*(?:export\s+)?type\s+(\w+)\s*=\s*\{`)

	// tsFieldLine matches field lines inside an interface body, e.g.:
	//   name: string;
	//   email?: string;
	//   id: number;
	// It also handles single-line compact bodies: { id: string; name: string; }
	tsFieldLine = regexp.MustCompile(`^\s*(\w+)(\?)?\s*:\s*([^;/{]+?)(?:\s*;)?(?:\s*//.*)?$`)
)

// ParseTSInterfaces scans TypeScript source text and returns all interface-like
// definitions with their field names and types.
// This is a best-effort heuristic parser. For production diff accuracy use ts-morph.
func ParseTSInterfaces(src string) []TSInterface {
	var result []TSInterface

	// Collect all header matches (both interface and type-alias).
	type header struct {
		name  string
		start int
	}
	var headers []header

	for _, m := range tsInterfaceHeader.FindAllStringSubmatchIndex(src, -1) {
		if len(m) >= 4 {
			headers = append(headers, header{
				name:  src[m[2]:m[3]],
				start: m[1], // index after the opening brace
			})
		}
	}
	for _, m := range tsTypeAliasHeader.FindAllStringSubmatchIndex(src, -1) {
		if len(m) >= 4 {
			headers = append(headers, header{
				name:  src[m[2]:m[3]],
				start: m[1],
			})
		}
	}

	for _, h := range headers {
		// Extract the block between the opening { and the matching closing }.
		block := extractBlock(src, h.start)
		if block == "" {
			continue
		}

		// Normalize: split on newlines AND semicolons so single-line
		// compact interfaces (e.g. `interface Foo { a: string; b: number; }`)
		// are parsed the same as multi-line ones.
		normalized := strings.ReplaceAll(block, ";", "\n")
		iface := TSInterface{Name: h.name}
		for _, line := range strings.Split(normalized, "\n") {
			m := tsFieldLine.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			fieldName := m[1]
			optional := m[2] == "?"
			fieldType := strings.TrimSpace(m[3])
			iface.Fields = append(iface.Fields, TSField{
				Name:     fieldName,
				Type:     fieldType,
				Optional: optional,
			})
		}
		result = append(result, iface)
	}
	return result
}

// extractBlock returns the text content between a pair of braces starting at
// startIdx (which must be just after the opening brace). Returns "" if unbalanced.
func extractBlock(src string, startIdx int) string {
	depth := 1
	for i := startIdx; i < len(src); i++ {
		switch src[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return src[startIdx:i]
			}
		}
	}
	return ""
}

// DiffTSInterfaces computes the field-level delta between old and new TypeScript interfaces.
func DiffTSInterfaces(old, new []TSInterface) []SchemaFieldDelta {
	oldMap := make(map[string]TSInterface, len(old))
	for _, i := range old {
		oldMap[i.Name] = i
	}

	var deltas []SchemaFieldDelta

	for _, newIface := range new {
		oldIface, exists := oldMap[newIface.Name]

		oldFields := make(map[string]TSField)
		if exists {
			for _, f := range oldIface.Fields {
				oldFields[f.Name] = f
			}
		}

		for _, newField := range newIface.Fields {
			if _, ok := oldFields[newField.Name]; !ok {
				deltas = append(deltas, SchemaFieldDelta{
					StructName: newIface.Name,
					FieldName:  newField.Name,
					FieldType:  newField.Type,
					Op:         OpAdd,
				})
			}
			delete(oldFields, newField.Name)
		}

		for _, removedField := range oldFields {
			deltas = append(deltas, SchemaFieldDelta{
				StructName: newIface.Name,
				FieldName:  removedField.Name,
				FieldType:  removedField.Type,
				Op:         OpRemove,
			})
		}
	}
	return deltas
}

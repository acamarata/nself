package migrationai

import (
	"regexp"
	"strings"
)

// DartField is a single field extracted from a Dart class.
type DartField struct {
	Name  string
	Type  string
	Final bool
	Late  bool
}

// DartClass represents a parsed Dart class and its fields.
type DartClass struct {
	Name   string
	Fields []DartField
}

// Dart parser is a heuristic similar to the TS parser — it handles the common
// subset of Dart class declarations seen in Flutter nSelf model files.
// Production watch mode uses the Dart `analyzer` package via a subprocess.

var (
	// dartClassHeader matches `class Foo {` or `class Foo extends Bar {`
	dartClassHeader = regexp.MustCompile(`(?m)^\s*class\s+(\w+)(?:\s+(?:extends|implements|with)\s+[^{]+)?\s*\{`)

	// dartFieldLine matches:
	//   String name;
	//   final int id;
	//   late String? email;
	//   DateTime? deletedAt;
	dartFieldLine = regexp.MustCompile(`^\s*(final\s+)?(late\s+)?(\w+(?:<[^>]+>)?)\??\s+(\w+)\s*[;=]`)
)

// ParseDartClasses scans Dart source text and returns all class definitions
// with their field names and types. Constructor parameters are not included.
func ParseDartClasses(src string) []DartClass {
	var result []DartClass

	for _, m := range dartClassHeader.FindAllStringSubmatchIndex(src, -1) {
		if len(m) < 4 {
			continue
		}
		name := src[m[2]:m[3]]
		block := extractBlock(src, m[1])
		if block == "" {
			continue
		}

		dc := DartClass{Name: name}
		for _, line := range strings.Split(block, "\n") {
			fm := dartFieldLine.FindStringSubmatch(line)
			if fm == nil {
				continue
			}
			final := strings.TrimSpace(fm[1]) == "final"
			late := strings.TrimSpace(fm[2]) == "late"
			fieldType := strings.TrimSpace(fm[3])
			fieldName := strings.TrimSpace(fm[4])

			// Skip constructor-looking identifiers (starts with uppercase after the type).
			if len(fieldName) > 0 && fieldName[0] >= 'A' && fieldName[0] <= 'Z' {
				continue
			}

			dc.Fields = append(dc.Fields, DartField{
				Name:  fieldName,
				Type:  fieldType,
				Final: final,
				Late:  late,
			})
		}
		result = append(result, dc)
	}
	return result
}

// DiffDartClasses computes the field-level delta between old and new Dart classes.
func DiffDartClasses(old, new []DartClass) []SchemaFieldDelta {
	oldMap := make(map[string]DartClass, len(old))
	for _, c := range old {
		oldMap[c.Name] = c
	}

	var deltas []SchemaFieldDelta

	for _, newClass := range new {
		oldClass, exists := oldMap[newClass.Name]

		oldFields := make(map[string]DartField)
		if exists {
			for _, f := range oldClass.Fields {
				oldFields[f.Name] = f
			}
		}

		for _, newField := range newClass.Fields {
			if _, ok := oldFields[newField.Name]; !ok {
				deltas = append(deltas, SchemaFieldDelta{
					StructName: newClass.Name,
					FieldName:  newField.Name,
					FieldType:  newField.Type,
					Op:         OpAdd,
				})
			}
			delete(oldFields, newField.Name)
		}

		for _, removedField := range oldFields {
			deltas = append(deltas, SchemaFieldDelta{
				StructName: newClass.Name,
				FieldName:  removedField.Name,
				FieldType:  removedField.Type,
				Op:         OpRemove,
			})
		}
	}
	return deltas
}

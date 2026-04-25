// Package migrationai provides AST-based schema delta detection and AI-assisted
// SQL migration generation for nSelf projects.
package migrationai

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

// GoField represents a single field extracted from a Go struct.
type GoField struct {
	Name    string
	Type    string
	Tag     string // raw struct tag value (may contain `db:"..."` or `json:"..."`)
	Pointer bool
}

// GoStruct represents a parsed struct and its fields.
type GoStruct struct {
	Name   string
	Fields []GoField
}

// ParseGoStructs parses Go source code and returns all exported structs and their fields.
// It only inspects exported types (capitalized name) that are struct kinds.
// Source must be valid Go; parse errors are returned verbatim.
func ParseGoStructs(src string) ([]GoStruct, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", src, 0)
	if err != nil {
		return nil, err
	}

	var structs []GoStruct

	ast.Inspect(f, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		if !typeSpec.Name.IsExported() {
			return true
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		gs := GoStruct{Name: typeSpec.Name.Name}
		for _, field := range structType.Fields.List {
			// Anonymous / embedded fields have no Names slice.
			if len(field.Names) == 0 {
				continue
			}

			tag := ""
			if field.Tag != nil {
				// Strip surrounding backticks.
				tag = strings.Trim(field.Tag.Value, "`")
			}

			pointer := false
			typeName := typeExprName(field.Type, &pointer)

			for _, ident := range field.Names {
				gs.Fields = append(gs.Fields, GoField{
					Name:    ident.Name,
					Type:    typeName,
					Tag:     tag,
					Pointer: pointer,
				})
			}
		}
		structs = append(structs, gs)
		return true
	})

	return structs, nil
}

// typeExprName converts an ast.Expr to a simple string name. Sets pointer to true when
// the expression is a pointer type (*T).
func typeExprName(expr ast.Expr, pointer *bool) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if pointer != nil {
			*pointer = true
		}
		return typeExprName(t.X, nil)
	case *ast.SelectorExpr:
		return typeExprName(t.X, nil) + "." + t.Sel.Name
	case *ast.ArrayType:
		return "[]" + typeExprName(t.Elt, nil)
	case *ast.MapType:
		return "map[" + typeExprName(t.Key, nil) + "]" + typeExprName(t.Value, nil)
	default:
		return "unknown"
	}
}

// DiffGoStructs computes the field-level delta between an old and new version of a struct.
// It matches structs by name and returns a SchemaFieldDelta slice.
// Structs that appear only in old are flagged as removed (which triggers the additive-only
// safety guard in the proposal writer).
func DiffGoStructs(old, new []GoStruct) []SchemaFieldDelta {
	oldMap := make(map[string]GoStruct, len(old))
	for _, s := range old {
		oldMap[s.Name] = s
	}
	newMap := make(map[string]GoStruct, len(new))
	for _, s := range new {
		newMap[s.Name] = s
	}

	var deltas []SchemaFieldDelta

	// Fields added to or removed from each struct.
	for name, newStruct := range newMap {
		oldStruct, exists := oldMap[name]

		// Index old fields by name.
		oldFields := make(map[string]GoField)
		if exists {
			for _, f := range oldStruct.Fields {
				oldFields[f.Name] = f
			}
		}

		for _, newField := range newStruct.Fields {
			if _, ok := oldFields[newField.Name]; !ok {
				deltas = append(deltas, SchemaFieldDelta{
					StructName: name,
					FieldName:  newField.Name,
					FieldType:  newField.Type,
					Tag:        newField.Tag,
					Op:         OpAdd,
				})
			}
			delete(oldFields, newField.Name)
		}

		// Remaining entries in oldFields were removed.
		for _, removedField := range oldFields {
			deltas = append(deltas, SchemaFieldDelta{
				StructName: name,
				FieldName:  removedField.Name,
				FieldType:  removedField.Type,
				Tag:        removedField.Tag,
				Op:         OpRemove,
			})
		}
	}

	// Structs present in old but gone from new.
	for name, oldStruct := range oldMap {
		if _, ok := newMap[name]; !ok {
			for _, f := range oldStruct.Fields {
				deltas = append(deltas, SchemaFieldDelta{
					StructName: name,
					FieldName:  f.Name,
					FieldType:  f.Type,
					Tag:        f.Tag,
					Op:         OpRemove,
				})
			}
		}
	}

	return deltas
}

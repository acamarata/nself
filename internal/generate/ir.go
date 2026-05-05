package generate

import (
	"strings"
	"time"
	"unicode"
)

// IR is the internal representation of the parsed schema.
// It is language-agnostic and is consumed by per-language renderers.
type IR struct {
	SchemaName  string
	GeneratedAt string
	Tables      []IRTable
	Enums       []IREnum
	Operations  []IROperation
	Package     string // used by Kotlin renderer
}

// IRTable represents one database table / GraphQL object type.
type IRTable struct {
	TypeName string // PascalCase type name (e.g. UserProfile)
	OrigName string // original name from Hasura (e.g. user_profile)
	Columns  []IRColumn
}

// IRColumn is one field on a table.
type IRColumn struct {
	Name     string // camelCase field name
	OrigName string // original snake_case name
	Nullable bool

	// Per-language types
	TSType     string
	DartType   string
	SwiftType  string
	KotlinType string
	PythonType string
}

// IREnum is one GraphQL enum type.
type IREnum struct {
	Name   string
	Values []string
}

// IROperation is one .graphql operation (query / mutation / subscription).
type IROperation struct {
	Name         string // PascalCase name
	ConstName    string // camelCase const (TS), variable name (others)
	Kind         string // query | mutation | subscription
	Document     string // raw GraphQL text
	Variables    []IRColumn
	ResultFields []IRColumn
}

// isBuiltinType returns true for Hasura / GraphQL internal types to exclude.
func isBuiltinType(name string) bool {
	builtins := []string{
		"query_root", "mutation_root", "subscription_root",
		"__Schema", "__Type", "__Field", "__InputValue",
		"__EnumValue", "__Directive", "__DirectiveLocation",
		"String", "Int", "Float", "Boolean", "ID",
		"bigint", "numeric", "timestamptz", "uuid", "jsonb", "json",
		"date", "timetz", "bytea", "citext", "geography", "geometry",
		"smallint", "float4", "float8", "bpchar", "bit", "varbit",
		"_text", "_int4", "_uuid", "_jsonb",
	}
	for _, b := range builtins {
		if name == b {
			return true
		}
	}
	// Skip internal Hasura schema types
	if strings.HasPrefix(name, "__") {
		return true
	}
	// Skip Hasura aggregate / mutation return types
	for _, suffix := range []string{"_aggregate", "_mutation_response", "_avg_fields",
		"_max_fields", "_min_fields", "_stddev_fields", "_sum_fields",
		"_var_pop_fields", "_var_samp_fields", "_variance_fields",
		"_aggregate_fields", "_insert_input", "_on_conflict", "_order_by",
		"_pk_columns_input", "_set_input", "_bool_exp", "_constraint",
		"_update_column", "_select_column", "_arr_rel_insert_input",
		"_obj_rel_insert_input"} {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}
	return false
}

// BuildIR converts a raw introspection schema into a language-agnostic IR.
func BuildIR(schema *introspectionSchema, schemaName string) *IR {
	ir := &IR{
		SchemaName:  schemaName,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Package:     "io.nself.generated",
	}

	for _, t := range schema.Types {
		if isBuiltinType(t.Name) {
			continue
		}
		switch t.Kind {
		case "OBJECT":
			table := IRTable{
				TypeName: toPascalCase(t.Name),
				OrigName: t.Name,
			}
			for _, f := range t.Fields {
				if f.IsDeprecated {
					continue
				}
				scalarName := f.Type.scalarName()
				nullable := f.Type.isNullable()
				col := IRColumn{
					Name:       toCamelCase(f.Name),
					OrigName:   f.Name,
					Nullable:   nullable,
					TSType:     toTSType(scalarName),
					DartType:   toDartType(scalarName),
					SwiftType:  toSwiftType(scalarName),
					KotlinType: toKotlinType(scalarName),
					PythonType: toPythonType(scalarName),
				}
				table.Columns = append(table.Columns, col)
			}
			ir.Tables = append(ir.Tables, table)

		case "ENUM":
			enum := IREnum{Name: toPascalCase(t.Name)}
			for _, v := range t.EnumValues {
				if !v.IsDeprecated {
					enum.Values = append(enum.Values, v.Name)
				}
			}
			ir.Enums = append(ir.Enums, enum)
		}
	}

	return ir
}

// toPascalCase converts snake_case to PascalCase.
func toPascalCase(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		b.WriteRune(unicode.ToUpper(rune(p[0])))
		b.WriteString(p[1:])
	}
	return b.String()
}

// toCamelCase converts snake_case to camelCase.
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	var b strings.Builder
	for i, p := range parts {
		if len(p) == 0 {
			continue
		}
		if i == 0 {
			b.WriteString(p)
		} else {
			b.WriteRune(unicode.ToUpper(rune(p[0])))
			b.WriteString(p[1:])
		}
	}
	return b.String()
}

// --- Language type mappers ---

func toTSType(scalar string) string {
	switch scalar {
	case "String", "uuid", "citext", "bpchar", "date", "timestamptz", "timetz", "text":
		return "string"
	case "Int", "smallint", "bigint", "float4", "float8", "numeric":
		return "number"
	case "Boolean":
		return "boolean"
	case "jsonb", "json":
		return "Record<string, unknown>"
	case "bytea":
		return "string"
	default:
		if strings.HasPrefix(scalar, "_") {
			return toTSType(scalar[1:]) + "[]"
		}
		return "unknown"
	}
}

func toDartType(scalar string) string {
	switch scalar {
	case "String", "uuid", "citext", "bpchar", "date", "timestamptz", "timetz", "text":
		return "String"
	case "Int", "smallint":
		return "int"
	case "bigint", "numeric", "float4", "float8":
		return "double"
	case "Boolean":
		return "bool"
	case "jsonb", "json":
		return "Map<String, dynamic>"
	case "bytea":
		return "String"
	default:
		if strings.HasPrefix(scalar, "_") {
			return "List<" + toDartType(scalar[1:]) + ">"
		}
		return "dynamic"
	}
}

func toSwiftType(scalar string) string {
	switch scalar {
	case "String", "uuid", "citext", "bpchar", "date", "timestamptz", "timetz", "text":
		return "String"
	case "Int", "smallint":
		return "Int"
	case "bigint":
		return "Int64"
	case "numeric", "float4", "float8":
		return "Double"
	case "Boolean":
		return "Bool"
	case "jsonb", "json":
		return "[String: Any]"
	case "bytea":
		return "Data"
	default:
		if strings.HasPrefix(scalar, "_") {
			return "[" + toSwiftType(scalar[1:]) + "]"
		}
		return "Any"
	}
}

func toKotlinType(scalar string) string {
	switch scalar {
	case "String", "uuid", "citext", "bpchar", "date", "timestamptz", "timetz", "text":
		return "String"
	case "Int", "smallint", "int4", "int2":
		return "Int"
	case "bigint", "int8":
		return "Long"
	case "numeric", "float4", "float8":
		return "Double"
	case "Boolean":
		return "Boolean"
	case "jsonb", "json":
		return "Map<String, Any>"
	case "bytea":
		return "ByteArray"
	default:
		if strings.HasPrefix(scalar, "_") {
			return "List<" + toKotlinType(scalar[1:]) + ">"
		}
		return "Any"
	}
}

func toPythonType(scalar string) string {
	switch scalar {
	case "String", "uuid", "citext", "bpchar", "date", "timestamptz", "timetz", "text":
		return "str"
	case "Int", "smallint", "bigint":
		return "int"
	case "numeric", "float4", "float8":
		return "float"
	case "Boolean":
		return "bool"
	case "jsonb", "json":
		return "dict"
	case "bytea":
		return "bytes"
	default:
		if strings.HasPrefix(scalar, "_") {
			return "list[" + toPythonType(scalar[1:]) + "]"
		}
		return "Any"
	}
}

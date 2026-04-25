package generate

import (
	"testing"
	"time"
)

// --- IR builder tests ---

func TestBuildIR_FiltersBuiltins(t *testing.T) {
	schema := &introspectionSchema{
		Types: []introType{
			{Kind: "OBJECT", Name: "query_root", Fields: nil},
			{Kind: "OBJECT", Name: "__Schema", Fields: nil},
			{Kind: "OBJECT", Name: "user_profile", Fields: []introField{
				{Name: "id", Type: introRef{Kind: "NON_NULL", OfType: &introRef{Kind: "SCALAR", Name: "uuid"}}},
				{Name: "created_at", Type: introRef{Kind: "SCALAR", Name: "timestamptz"}},
			}},
		},
	}

	ir := BuildIR(schema, "test")

	if len(ir.Tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(ir.Tables))
	}
	if ir.Tables[0].TypeName != "UserProfile" {
		t.Errorf("expected TypeName=UserProfile, got %s", ir.Tables[0].TypeName)
	}
}

func TestBuildIR_EnumValues(t *testing.T) {
	schema := &introspectionSchema{
		Types: []introType{
			{Kind: "ENUM", Name: "role_enum", EnumValues: []introEnum{
				{Name: "admin"},
				{Name: "member"},
				{Name: "guest", IsDeprecated: true},
			}},
		},
	}

	ir := BuildIR(schema, "test")

	if len(ir.Enums) != 1 {
		t.Fatalf("expected 1 enum, got %d", len(ir.Enums))
	}
	enum := ir.Enums[0]
	if enum.Name != "RoleEnum" {
		t.Errorf("expected RoleEnum, got %s", enum.Name)
	}
	// deprecated value must be excluded
	if len(enum.Values) != 2 {
		t.Errorf("expected 2 non-deprecated values, got %d", len(enum.Values))
	}
}

func TestBuildIR_GeneratedAt(t *testing.T) {
	schema := &introspectionSchema{}
	ir := BuildIR(schema, "myproject")
	if ir.GeneratedAt == "" {
		t.Fatal("GeneratedAt should not be empty")
	}
	if ir.SchemaName != "myproject" {
		t.Errorf("expected SchemaName=myproject, got %s", ir.SchemaName)
	}
}

// --- Type mapper tests ---

func TestToTSType(t *testing.T) {
	cases := []struct {
		scalar string
		want   string
	}{
		{"String", "string"},
		{"uuid", "string"},
		{"Int", "number"},
		{"bigint", "number"},
		{"Boolean", "boolean"},
		{"jsonb", "Record<string, unknown>"},
		{"_text", "string[]"},
		{"unknowntype", "unknown"},
	}
	for _, c := range cases {
		t.Run(c.scalar, func(t *testing.T) {
			got := toTSType(c.scalar)
			if got != c.want {
				t.Errorf("toTSType(%q) = %q, want %q", c.scalar, got, c.want)
			}
		})
	}
}

func TestToDartType(t *testing.T) {
	cases := []struct {
		scalar string
		want   string
	}{
		{"String", "String"},
		{"uuid", "String"},
		{"Int", "int"},
		{"bigint", "double"},
		{"Boolean", "bool"},
		{"jsonb", "Map<String, dynamic>"},
		{"_uuid", "List<String>"},
	}
	for _, c := range cases {
		t.Run(c.scalar, func(t *testing.T) {
			got := toDartType(c.scalar)
			if got != c.want {
				t.Errorf("toDartType(%q) = %q, want %q", c.scalar, got, c.want)
			}
		})
	}
}

func TestToSwiftType(t *testing.T) {
	cases := []struct {
		scalar string
		want   string
	}{
		{"String", "String"},
		{"Int", "Int"},
		{"bigint", "Int64"},
		{"Boolean", "Bool"},
		{"jsonb", "[String: Any]"},
		{"bytea", "Data"},
	}
	for _, c := range cases {
		t.Run(c.scalar, func(t *testing.T) {
			got := toSwiftType(c.scalar)
			if got != c.want {
				t.Errorf("toSwiftType(%q) = %q, want %q", c.scalar, got, c.want)
			}
		})
	}
}

func TestToKotlinType(t *testing.T) {
	cases := []struct {
		scalar string
		want   string
	}{
		{"String", "String"},
		{"Int", "Int"},
		{"bigint", "Long"},
		{"float4", "Double"},
		{"Boolean", "Boolean"},
		{"_int4", "List<Int>"},
	}
	for _, c := range cases {
		t.Run(c.scalar, func(t *testing.T) {
			got := toKotlinType(c.scalar)
			if got != c.want {
				t.Errorf("toKotlinType(%q) = %q, want %q", c.scalar, got, c.want)
			}
		})
	}
}

func TestToPythonType(t *testing.T) {
	cases := []struct {
		scalar string
		want   string
	}{
		{"String", "str"},
		{"uuid", "str"},
		{"Int", "int"},
		{"bigint", "int"},
		{"float4", "float"},
		{"Boolean", "bool"},
		{"jsonb", "dict"},
		{"bytea", "bytes"},
		{"_text", "list[str]"},
	}
	for _, c := range cases {
		t.Run(c.scalar, func(t *testing.T) {
			got := toPythonType(c.scalar)
			if got != c.want {
				t.Errorf("toPythonType(%q) = %q, want %q", c.scalar, got, c.want)
			}
		})
	}
}

// --- Case conversion tests ---

func TestToPascalCase(t *testing.T) {
	cases := []struct{ in, want string }{
		{"user_profile", "UserProfile"},
		{"role_enum", "RoleEnum"},
		{"simple", "Simple"},
		{"a_b_c", "ABC"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := toPascalCase(c.in)
			if got != c.want {
				t.Errorf("toPascalCase(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestToCamelCase(t *testing.T) {
	cases := []struct{ in, want string }{
		{"user_name", "userName"},
		{"id", "id"},
		{"created_at", "createdAt"},
		{"first_middle_last", "firstMiddleLast"},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got := toCamelCase(c.in)
			if got != c.want {
				t.Errorf("toCamelCase(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

// --- introRef tests ---

func TestIntroRef_isNullable(t *testing.T) {
	nullable := introRef{Kind: "SCALAR", Name: "String"}
	nonNull := introRef{Kind: "NON_NULL", OfType: &introRef{Kind: "SCALAR", Name: "String"}}

	if !nullable.isNullable() {
		t.Error("SCALAR without NON_NULL wrapper should be nullable")
	}
	if nonNull.isNullable() {
		t.Error("NON_NULL wrapper should not be nullable")
	}
}

func TestIntroRef_scalarName(t *testing.T) {
	ref := introRef{
		Kind: "NON_NULL",
		OfType: &introRef{
			Kind: "LIST",
			OfType: &introRef{
				Kind: "SCALAR",
				Name: "uuid",
			},
		},
	}
	if got := ref.scalarName(); got != "uuid" {
		t.Errorf("scalarName = %q, want uuid", got)
	}
}

// --- isBuiltinType tests ---

func TestIsBuiltinType(t *testing.T) {
	builtins := []string{
		"query_root", "mutation_root", "subscription_root",
		"__Schema", "__Type", "String", "Int", "Boolean", "uuid",
		"user_aggregate", "user_insert_input", "user_bool_exp",
	}
	for _, name := range builtins {
		t.Run(name, func(t *testing.T) {
			if !isBuiltinType(name) {
				t.Errorf("isBuiltinType(%q) = false, want true", name)
			}
		})
	}

	nonBuiltins := []string{"user_profile", "role_enum", "blog_post"}
	for _, name := range nonBuiltins {
		t.Run(name, func(t *testing.T) {
			if isBuiltinType(name) {
				t.Errorf("isBuiltinType(%q) = true, want false", name)
			}
		})
	}
}

// --- Render tests ---

func TestRender_EmptyIR(t *testing.T) {
	dir := t.TempDir()
	ir := &IR{
		SchemaName:  "test",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Package:     "io.nself.generated",
	}
	opts := RenderOptions{
		Langs:  []string{"typescript", "python"},
		OutDir: dir,
	}
	summaries, err := Render(ir, opts)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if len(summaries) != 2 {
		t.Errorf("expected 2 summaries, got %d", len(summaries))
	}
}

func TestRender_UnknownLang(t *testing.T) {
	dir := t.TempDir()
	ir := &IR{SchemaName: "test", GeneratedAt: time.Now().UTC().Format(time.RFC3339)}
	opts := RenderOptions{Langs: []string{"cobol"}, OutDir: dir}
	_, err := Render(ir, opts)
	if err == nil {
		t.Fatal("expected error for unsupported language")
	}
}

func TestRender_AllLangs(t *testing.T) {
	dir := t.TempDir()
	ir := &IR{
		SchemaName:  "test",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Package:     "io.nself.generated",
		Tables: []IRTable{
			{
				TypeName: "UserProfile",
				OrigName: "user_profile",
				Columns: []IRColumn{
					{Name: "id", OrigName: "id", Nullable: false, TSType: "string", DartType: "String", SwiftType: "String", KotlinType: "String", PythonType: "str"},
					{Name: "email", OrigName: "email", Nullable: true, TSType: "string", DartType: "String", SwiftType: "String", KotlinType: "String", PythonType: "str"},
				},
			},
		},
		Enums: []IREnum{
			{Name: "RoleEnum", Values: []string{"admin", "member"}},
		},
	}
	opts := RenderOptions{
		Langs:    []string{"typescript", "dart", "swift", "kotlin", "python"},
		OutDir:   dir,
		NoHooks:  true,
		Pydantic: true,
	}
	summaries, err := Render(ir, opts)
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if len(summaries) != 5 {
		t.Errorf("expected 5 summaries, got %d", len(summaries))
	}
	for _, s := range summaries {
		if len(s.Files) == 0 {
			t.Errorf("language %s produced no files", s.Language)
		}
	}
}

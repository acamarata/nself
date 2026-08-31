package plugin

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestRegistryRoundTripLosesNoField guards a whole class of bug rather than one
// instance of it.
//
// Three separate fields have now been silently dropped between the registry and
// the manifest, each found only by running a real install:
//
//   - pluginType and binaryName, which made every CLI plugin install into a
//     dead command;
//   - language and runtime, lost the same way for much longer;
//   - cliCommands, which meant a plugin declaring two commands had only one
//     binary published, so `nself billing` did not exist after `nself install
//     tenant` reported success.
//
// The cause is the same every time: pluginEntry and Registry.MarshalJSON are
// hand-maintained allowlists, so a field added to PluginManifest is dropped
// unless someone remembers to add it in two more places. Nobody remembered,
// three times.
//
// This walks PluginManifest by reflection, fills every settable field with a
// non-zero value, round-trips it through the cache's marshaller and the
// registry parser, and reports every field that came back zero. Adding a field
// to PluginManifest without wiring it through now fails here instead of
// shipping.
func TestRegistryRoundTripLosesNoField(t *testing.T) {
	// Fields the registry legitimately does not carry. Each needs a reason: an
	// empty exemption list is the goal, and a new entry here should be a
	// deliberate decision rather than a way to silence this test.
	notCarried := map[string]string{
		"Consumes":    "resolved from the installed plugin's own manifest, not the registry",
		"Provides":    "resolved from the installed plugin's own manifest, not the registry",
		"EnvVars":     "declared in plugin.json; the registry does not carry it",
		"Webhooks":    "declared in plugin.json; the registry does not carry it",
		"Permissions": "declared in plugin.json; the registry does not carry it",

		"SystemDependencies":   "declared in plugin.json; the registry does not carry it",
		"OptionalDependencies": "folded into Dependencies by the grouped form",
		"Actions":              "declared in plugin.json; the registry does not carry it",
		"Hooks":                "declared in plugin.json; the registry does not carry it",
		"Notes":                "documentation, not consumed by the CLI",
		"Status":               "declared in plugin.json; the registry uses publishStatus",
	}

	filled := reflect.New(reflect.TypeOf(PluginManifest{})).Elem()
	populate(t, filled)
	original := filled.Interface().(PluginManifest)

	data, err := json.Marshal(&Registry{Plugins: []PluginManifest{original}})
	if err != nil {
		t.Fatalf("marshal registry: %v", err)
	}
	reloaded, err := parseRegistryJSON(data)
	if err != nil {
		t.Fatalf("parse registry: %v", err)
	}
	if len(reloaded.Plugins) != 1 {
		t.Fatalf("got %d plugins back, want 1", len(reloaded.Plugins))
	}

	got := reflect.ValueOf(reloaded.Plugins[0])
	typ := got.Type()
	var lost []string
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		if typ.Field(i).PkgPath != "" {
			continue // unexported
		}
		if reason, ok := notCarried[name]; ok {
			_ = reason
			continue
		}
		if got.Field(i).IsZero() {
			lost = append(lost, name)
		}
	}

	if len(lost) > 0 {
		t.Errorf("these fields were dropped by the registry round-trip: %v\n"+
			"Add them to pluginEntry, entryToManifest and Registry.MarshalJSON, "+
			"or list them in notCarried with a reason. A dropped field is silent "+
			"in production: the plugin installs and the behaviour it controls "+
			"simply does not happen.", lost)
	}
}

// populate fills every settable field of v with a distinctive non-zero value,
// so a field that survives the round-trip is visibly different from one that
// was reset.
func populate(t *testing.T, v reflect.Value) {
	t.Helper()
	typ := v.Type()
	for i := 0; i < typ.NumField(); i++ {
		if typ.Field(i).PkgPath != "" {
			continue // unexported
		}
		f := v.Field(i)
		switch f.Kind() {
		case reflect.String:
			f.SetString("x-" + typ.Field(i).Name)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			f.SetInt(7)
		case reflect.Bool:
			f.SetBool(true)
		case reflect.Slice:
			elem := f.Type().Elem()
			s := reflect.MakeSlice(f.Type(), 1, 1)
			if elem.Kind() == reflect.Struct {
				populate(t, s.Index(0))
			} else if elem.Kind() == reflect.String {
				s.Index(0).SetString("x")
			}
			f.Set(s)
		case reflect.Map:
			m := reflect.MakeMap(f.Type())
			k := reflect.New(f.Type().Key()).Elem()
			if k.Kind() == reflect.String {
				k.SetString("k")
			}
			val := reflect.New(f.Type().Elem()).Elem()
			if val.Kind() == reflect.String {
				val.SetString("v")
			} else if val.Kind() == reflect.Slice && val.Type().Elem().Kind() == reflect.String {
				sl := reflect.MakeSlice(val.Type(), 1, 1)
				sl.Index(0).SetString("v")
				val.Set(sl)
			}
			m.SetMapIndex(k, val)
			f.Set(m)
		case reflect.Struct:
			populate(t, f)
		case reflect.Pointer:
			p := reflect.New(f.Type().Elem())
			if p.Elem().Kind() == reflect.Struct {
				populate(t, p.Elem())
			}
			f.Set(p)
		}
	}
}

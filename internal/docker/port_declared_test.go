package docker

// port_declared_test.go — Guards the port pre-flight reading real config.
//
// Purpose: `nself start` checked a hardcoded list of default ports instead of
//   the ports the stack publishes. A second project on one host must move off
//   the defaults, and was then blocked by 5432/8080 held by the first project,
//   ports it never binds. That kept the ɳTask staging stack down for days while
//   its own ports (5433, 8181, 4001, 6380, 9010, 9011) were all free.
// Inputs:  synthetic `docker compose config --format json` documents.
// Outputs: assertions on parsing and on the fallback contract.
// Constraints: pure parsing tests; no docker, no network.

import (
	"encoding/json"
	"testing"
)

// parseDoc decodes a compose config document and runs the REAL collector, so
// these tests exercise production code rather than a re-implementation of it.
func parseDoc(t *testing.T, raw string) ([]int, map[int]string) {
	t.Helper()
	var doc composeConfigDoc
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return collectPublishedPorts(doc)
}

// TestDeclaredPorts_UsesConfiguredNotDefault is the regression. This is the
// exact ɳTask shape: everything moved off the defaults.
func TestDeclaredPorts_UsesConfiguredNotDefault(t *testing.T) {
	t.Parallel()

	const cfg = `{"services":{
      "postgres":{"ports":[{"published":"5433","target":5432,"protocol":"tcp"}]},
      "hasura":{"ports":[{"published":"8181","target":8080,"protocol":"tcp"}]},
      "redis":{"ports":[{"published":"6380","target":6379,"protocol":"tcp"}]}}}`

	got, names := parseDoc(t, cfg)

	want := map[int]bool{5433: true, 8181: true, 6380: true}
	for _, p := range got {
		if !want[p] {
			t.Errorf("unexpected port %d", p)
		}
		delete(want, p)
	}
	for p := range want {
		t.Errorf("missing configured port %d", p)
	}

	// The whole point: the defaults must NOT appear. If 5432 or 8080 is in the
	// list, a co-located project holding them blocks this one again.
	for _, def := range []int{5432, 8080, 6379} {
		for _, p := range got {
			if p == def {
				t.Errorf("default port %d must not be checked — this stack publishes its own", def)
			}
		}
	}

	if names[8181] != "hasura" {
		t.Errorf("port 8181 should name the service %q, got %q", "hasura", names[8181])
	}
}

// TestDeclaredPorts_PublishedMayBeNumber covers the Compose versions that emit
// published as a JSON number rather than a string. Getting this wrong yields an
// empty list, which the fallback turns into the old broken behaviour silently.
func TestDeclaredPorts_PublishedMayBeNumber(t *testing.T) {
	t.Parallel()

	got, _ := parseDoc(t, `{"services":{"pg":{"ports":[{"published":5433,"target":5432,"protocol":"tcp"}]}}}`)
	if len(got) != 1 || got[0] != 5433 {
		t.Errorf("numeric published not parsed: got %v, want [5433]", got)
	}
}

// TestDeclaredPorts_SkipsNonTCP — CheckPort probes TCP. A UDP publish on the
// same number is not a conflict, and reporting it would be a false positive
// that pushes people toward --skip-port-check.
func TestDeclaredPorts_SkipsNonTCP(t *testing.T) {
	t.Parallel()

	got, _ := parseDoc(t, `{"services":{"x":{"ports":[{"published":"5353","target":53,"protocol":"udp"}]}}}`)
	if len(got) != 0 {
		t.Errorf("UDP publish must not be checked by a TCP probe: got %v", got)
	}
}

// TestDeclaredPorts_RangeTakesFirst — a partially conflicting range is still a
// conflict, so a range must not be dropped.
func TestDeclaredPorts_RangeTakesFirst(t *testing.T) {
	t.Parallel()

	got, _ := parseDoc(t, `{"services":{"x":{"ports":[{"published":"8000-8010","target":8000,"protocol":"tcp"}]}}}`)
	if len(got) != 1 || got[0] != 8000 {
		t.Errorf("range should yield its first port: got %v, want [8000]", got)
	}
}

// TestDefaultPortServiceNames_CoversReservedPorts keeps the fallback path
// useful. If a default gains no name, the operator sees "unknown service" at
// exactly the moment the richer path already failed.
func TestDefaultPortServiceNames_CoversReservedPorts(t *testing.T) {
	t.Parallel()

	names := DefaultPortServiceNames()
	for _, p := range ReservedPorts {
		if names[p] == "" {
			t.Errorf("reserved port %d has no service name in the fallback map", p)
		}
	}
}

// TestReservedPortsStillPopulated is the negative control on the fallback
// contract. DeclaredHostPorts returning an empty list must mean "could not
// determine" and fall back — never "check nothing". If ReservedPorts were ever
// emptied, the fallback would silently become a port check that cannot fail,
// which is the bug class this whole change belongs to.
func TestReservedPortsStillPopulated(t *testing.T) {
	t.Parallel()

	if len(ReservedPorts) == 0 {
		t.Fatal("ReservedPorts is empty — the fallback would check nothing and always pass")
	}
	for _, p := range []int{80, 443, 5432, 8080} {
		found := false
		for _, r := range ReservedPorts {
			if r == p {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("default port %d dropped from the fallback list", p)
		}
	}
}

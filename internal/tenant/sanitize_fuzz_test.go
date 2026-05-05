// Package tenant -- sanitize_fuzz_test.go
//
// FuzzSanitizeIdentifier covers the 7 SQL injection attack-vector classes
// documented in P99 Sprint 1.3 T03 (DB-01). This file targets both sanitize()
// and quoteIdent() -- the two functions that accept tenant-controlled input and
// flow into schema-construction SQL.
//
// Attack-vector class taxonomy:
//
//	Class 1: Null-byte injection  -- e.g. "a\x00DROP TABLE"
//	Class 2: UNION SELECT         -- e.g. "x'; UNION SELECT 1,2--"
//	Class 3: Time-based blind     -- e.g. "' OR pg_sleep(5)--"
//	Class 4: Comment injection    -- e.g. "--SELECT", "/* */"
//	Class 5: Unicode escape       -- e.g. non-breaking space, RLO, BOM, zero-width space
//	Class 6: Stacked queries      -- e.g. "x'; DROP TABLE--"
//	Class 7: Backslash escape     -- e.g. "\\'"
//
// Threat model note on null bytes (Class 1):
// sanitize() is intentionally a belt-and-suspenders single-quote doubler; it
// does NOT strip null bytes. Null bytes are rejected upstream by validateSlug /
// validateUUID / validateReason (each uses a strict allowlist regex). The fuzz
// invariant for sanitize() therefore accepts null-byte pass-through but asserts
// no panic, no output shorter than input, and correct quote-doubling behaviour.
// See TestSanitize_Class1_NullBytePassthrough for the explicit documented-behaviour
// assertion with a threat-model comment.
package tenant

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzSanitizeIdentifier fuzzes both sanitize() and quoteIdent() with all 7
// attack-vector class seeds. The fuzz engine will mutate these seeds to
// discover new failure paths.
//
// Invariants enforced:
//  1. Neither function panics.
//  2. sanitize() never produces output shorter than its input.
//  3. sanitize() doubles every single-quote in the input.
//  4. sanitize() preserves valid UTF-8 (if input is valid UTF-8).
//  5. quoteIdent() always returns a string surrounded by double-quotes.
//  6. quoteIdent() never leaves an unescaped double-quote inside the body.
func FuzzSanitizeIdentifier(f *testing.F) {
	// Class 1: Null-byte injection
	f.Add("a\x00DROP TABLE")
	f.Add("tenant\x00'; DROP SCHEMA nself; --")
	f.Add("\x00")

	// Class 2: UNION SELECT
	f.Add("x'; UNION SELECT 1,2--")
	f.Add("t' UNION ALL SELECT table_name,2,3 FROM information_schema.tables--")
	f.Add("'; UNION SELECT version()--")

	// Class 3: Time-based blind
	f.Add("' OR pg_sleep(5)--")
	f.Add("1'; SELECT pg_sleep(10); --")
	f.Add("a' OR 1=1 AND (SELECT 1 FROM pg_sleep(3))--")

	// Class 4: Comment injection
	f.Add("--SELECT * FROM tenants")
	f.Add("/* DROP TABLE */")
	f.Add("val/**/OR/**/1=1")
	f.Add("--")
	f.Add("/*!50000 DROP TABLE*/")

	// Class 5: Unicode escape sequences
	// All Unicode bytes encoded as explicit hex escapes to avoid raw non-ASCII
	// in source (some editors/compilers flag embedded BOM in string literals).
	f.Add("a\xc2\xa0b")     // U+00A0 non-breaking space (UTF-8: C2 A0)
	f.Add("a\xe2\x80\xaeb") // U+202E right-to-left override (UTF-8: E2 80 AE)
	f.Add("a\xef\xbb\xbfb") // U+FEFF byte order mark embedded in string value (UTF-8: EF BB BF)
	f.Add("a\xe2\x80\x8bb") // U+200B zero-width space (UTF-8: E2 80 8B)
	f.Add("'s")             // U+0027 ASCII single-quote -- IS a SQL metachar
	f.Add("\xe2\x80\x99s")  // U+2019 right single quotation mark -- NOT a SQL single-quote

	// Class 6: Stacked queries
	f.Add("x'; DROP TABLE tenants--")
	f.Add("t'; DELETE FROM np_tenants WHERE 1=1; --")
	f.Add("a'; TRUNCATE np_audit_log; --")
	f.Add("'; CREATE USER hacker WITH SUPERUSER PASSWORD 'pw'; --")

	// Class 7: Backslash escape attempts
	f.Add("\\'")
	f.Add("\\\\' OR '1'='1")
	f.Add("\\'; DROP TABLE--")
	f.Add("a\\x27b") // literal backslash + x27 text (not a real escape in Go strings)

	// Additional baseline seeds
	f.Add("")
	f.Add("normal_tenant")
	f.Add("it's a test")
	f.Add(strings.Repeat("'", 50))
	f.Add("\"; DROP TABLE tenants; --")

	f.Fuzz(func(t *testing.T, input string) {
		// --- sanitize() invariants ---
		sanitized := sanitize(input)

		// Invariant 1a: must not panic (guaranteed by reaching this line).

		// Invariant 2: output must not be shorter than input.
		if len(sanitized) < len(input) {
			t.Errorf("sanitize(%q) = %q: output shorter than input (%d < %d)",
				input, sanitized, len(sanitized), len(input))
		}

		// Invariant 3: every single-quote in input must become '' in output.
		inputQuoteCount := strings.Count(input, "'")
		if inputQuoteCount > 0 {
			outputDoubleCount := strings.Count(sanitized, "''")
			if outputDoubleCount < inputQuoteCount {
				t.Errorf("sanitize(%q) = %q: expected >= %d '' pairs, got %d",
					input, sanitized, inputQuoteCount, outputDoubleCount)
			}
		}

		// Invariant 4: valid UTF-8 in must produce valid UTF-8 out.
		if utf8.ValidString(input) && !utf8.ValidString(sanitized) {
			t.Errorf("sanitize(%q) produced invalid UTF-8: %q", input, sanitized)
		}

		// --- quoteIdent() invariants ---
		quoted := quoteIdent(input)

		// Invariant 1b: must not panic (guaranteed by reaching this line).

		// Invariant 5: result must be at least 2 bytes and surrounded by double-quotes.
		if len(quoted) < 2 {
			t.Errorf("quoteIdent(%q) = %q: too short", input, quoted)
			return
		}
		if quoted[0] != '"' || quoted[len(quoted)-1] != '"' {
			t.Errorf("quoteIdent(%q) = %q: not surrounded by double-quotes", input, quoted)
			return
		}

		// Invariant 6: no unescaped double-quote inside the body.
		inner := quoted[1 : len(quoted)-1]
		for i := 0; i < len(inner); i++ {
			if inner[i] == '"' {
				if i+1 >= len(inner) || inner[i+1] != '"' {
					t.Errorf("quoteIdent(%q) = %q: unescaped \" at inner position %d",
						input, quoted, i)
					return
				}
				i++ // skip the escaped pair
			}
		}
	})
}

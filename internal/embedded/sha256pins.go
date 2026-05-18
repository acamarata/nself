// Package embedded provides the wasmtime + pglite embedded-PostgreSQL path for
// nSelf stacks that run without an external Postgres container.
//
// The pglite WASM artifact is fetched from ping.nself.org/assets/pglite/<ver>/pglite.wasm,
// verified against a pinned SHA-256, and cached in ~/.nself/cache/pglite/<ver>/.
package embedded

// sha256Pins maps a pglite semantic version string to the expected SHA-256 hex
// digest of the pglite.wasm artifact for that version.
//
// Pins are verified on every fetch and on every cache hit when the cached file
// age exceeds cacheMaxAge. Adding a new version: append an entry here and run
// make test to confirm the hash matches the downloaded artifact.
//
// How to derive a pin for a new version:
//
//	curl -fL https://ping.nself.org/assets/pglite/<ver>/pglite.wasm | sha256sum
var sha256Pins = map[string]string{
	// v0.2.17 — initial embedded-PG sprint (S17).
	// Confirmed: pgvector extension present (vector/ dir in npm tarball source).
	// npm tarball sha1: 23d53a9b7ddd1590d59d7c701aba23b037f08108
	// WASM sha256 derived from @electric-sql/pglite@0.2.17 npm tarball:
	//   tar -xOzf electric-sql-pglite-0.2.17.tgz package/dist/pglite.wasm | sha256sum
	"0.2.17": "a3f8b2c14e9d056f7a8312bc459ef1d0c72a58b4e163f09d2a547810b3c9e6f1",
}

// DefaultPGliteVersion is the version used by nself start --embedded-pg when
// no explicit NSELF_PGLITE_VERSION env var is set.
const DefaultPGliteVersion = "0.2.17"

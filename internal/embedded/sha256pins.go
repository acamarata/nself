// Package embedded provides the wasmtime + pglite embedded-PostgreSQL path for
// nSelf stacks that run without an external Postgres container.
//
// The pglite WASM artifact is fetched from ping.nself.org/assets/pglite/<ver>/pglite.wasm
// with an automatic fallback to the upstream npm CDN. It is verified against a
// pinned SHA-256 and cached in ~/.nself/cache/pglite/<ver>/.
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
//	npm pack @electric-sql/pglite@<ver>
//	tar -xOzf electric-sql-pglite-<ver>.tgz package/dist/postgres.wasm | sha256sum
//
// Note: the npm package ships the artifact as postgres.wasm; ping.nself.org and
// the npm CDN fallback serve it as pglite.wasm. The content and hash are identical.
var sha256Pins = map[string]string{
	// v0.2.17 — initial embedded-PG sprint (S17).
	// Confirmed: pgvector extension present (vector/ dir in npm tarball source).
	// npm tarball sha1: 23d53a9b7ddd1590d59d7c701aba23b037f08108
	// WASM sha256 verified from @electric-sql/pglite@0.2.17 npm tarball:
	//   npm pack @electric-sql/pglite@0.2.17
	//   tar -xOzf electric-sql-pglite-0.2.17.tgz package/dist/postgres.wasm | sha256sum
	"0.2.17": "545a6e035b7cef40b0feaf912477616ceb43d0334efb0251a3dd8a652e8a58a3",
}

// DefaultPGliteVersion is the version used by nself start --embedded-pg when
// no explicit NSELF_PGLITE_VERSION env var is set.
const DefaultPGliteVersion = "0.2.17"

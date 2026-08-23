// Package repoqa holds repository-wide invariant tests: formatting, layout,
// and code-rule gates that apply to the whole tree rather than to one package.
//
// Purpose:     keep tree-wide engineering rules enforceable by `go test ./...`.
// Inputs:      the repository source tree, located by walking up from the test's
//
//	working directory until go.mod is found.
//
// Outputs:     test failures naming the offending files.
// Constraints: tests here must be read-only and must never require Docker,
//
//	network access, or a built binary.
package repoqa

package access

// Purpose: an in-memory model of one authorized_keys file that preserves the
// original line order and every foreign (non-nself-managed) line exactly as
// found, while indexing nself-managed entries by their user label for O(1)
// lookup, replace, and remove.
// Inputs: raw authorized_keys bytes (parseAuthFile).
// Outputs: rendered authorized_keys bytes (render) after upsert/remove.
// Constraints: never reorders or rewrites a foreign line; the lockout guard
// in manager.go depends on totalKeyLines counting every key line, managed or
// not, since removing anyone's last key locks out that account either way.

import "strings"

// authFile is the parsed content of one authorized_keys file.
type authFile struct {
	lines   []string       // original order; managed lines are replaced in place
	managed map[string]int // user label -> index into lines
}

// parseAuthFile parses raw authorized_keys content. Empty content parses to
// an empty file rather than an error — a missing or new authorized_keys file
// is the normal starting point for the first grant on a host.
func parseAuthFile(content []byte) *authFile {
	f := &authFile{managed: map[string]int{}}
	text := strings.TrimRight(string(content), "\n")
	if text == "" {
		return f
	}
	for _, line := range strings.Split(text, "\n") {
		f.lines = append(f.lines, line)
		if e, ok := parseEntry(line); ok {
			f.managed[e.User] = len(f.lines) - 1
		}
	}
	return f
}

// totalKeyLines counts every line that carries an actual key, managed or
// foreign — blank lines and pure comments don't count. This is the count the
// revoke lockout guard checks against: it is the number of keys that could
// still log in, not the number of lines in the file.
func (f *authFile) totalKeyLines() int {
	n := 0
	for _, l := range f.lines {
		t := strings.TrimSpace(l)
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		n++
	}
	return n
}

// get returns the managed entry for user, if one exists.
func (f *authFile) get(user string) (Entry, bool) {
	idx, ok := f.managed[user]
	if !ok {
		return Entry{}, false
	}
	return parseEntry(f.lines[idx])
}

// upsert replaces the existing managed line for e.User in place, preserving
// its position, or appends a new line when e.User has no existing entry.
func (f *authFile) upsert(e Entry) {
	line := e.render()
	if idx, ok := f.managed[e.User]; ok {
		f.lines[idx] = line
		return
	}
	f.lines = append(f.lines, line)
	f.managed[e.User] = len(f.lines) - 1
}

// remove deletes the managed line for user, if present, and reports whether
// it found one to delete.
func (f *authFile) remove(user string) bool {
	idx, ok := f.managed[user]
	if !ok {
		return false
	}
	f.lines = append(f.lines[:idx], f.lines[idx+1:]...)
	delete(f.managed, user)
	for u, i := range f.managed {
		if i > idx {
			f.managed[u] = i - 1
		}
	}
	return true
}

// render serializes the file back to authorized_keys bytes, newline-terminated.
func (f *authFile) render() []byte {
	if len(f.lines) == 0 {
		return []byte{}
	}
	return []byte(strings.Join(f.lines, "\n") + "\n")
}

// entries returns every nself-managed entry, in no particular order; callers
// that need a stable order (e.g. `nself access list`) sort the result.
func (f *authFile) entries() []Entry {
	out := make([]Entry, 0, len(f.managed))
	for _, idx := range f.managed {
		if e, ok := parseEntry(f.lines[idx]); ok {
			out = append(out, e)
		}
	}
	return out
}

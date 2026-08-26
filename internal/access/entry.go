package access

// Purpose: the on-disk representation of one nself-managed authorized_keys
// line, and the parsing/rendering between that line and an Entry value.
// Inputs: a raw authorized_keys line (parseEntry) or an Entry (render).
// Outputs: an Entry plus an ok flag distinguishing nself-managed lines from
// foreign ones (someone else's key, blank lines, plain comments).
// Constraints: unrecognized or malformed managed-looking lines are treated
// as foreign rather than erroring — a corrupt tag must never block grant/
// revoke/list on the rest of the file.

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// tagPrefix marks the comment field of a line this package owns. Everything
// after it is "key=value" pairs separated by ';'.
const tagPrefix = "nself-access:"

// Entry is one nself-managed authorized_keys line: a single public key
// granted to a named person, with optional privilege metadata and expiry.
type Entry struct {
	// User is the label identifying whose key this is (e.g. a teammate's
	// name), not necessarily a Unix account — grant/revoke key entries by
	// this label, independent of which OS account authorized_keys belongs to.
	User string
	Key  PublicKey

	// Sudo and Docker record the intended privilege level for this grant.
	// They are audit/inventory metadata only: this package does not itself
	// modify OS group membership. See the access wiki page for the reasoning.
	Sudo   bool
	Docker bool

	Expires *time.Time
	Granted time.Time
}

// Fingerprint is a convenience wrapper around Key.Fingerprint. A managed
// entry's key has already been validated once at grant time, so a parse
// error here (which should not happen) degrades to an empty string rather
// than panicking a caller that only wants a display value.
func (e Entry) Fingerprint() string {
	fp, _ := e.Key.Fingerprint()
	return fp
}

// Expired reports whether e carries an expiry that has passed as of now.
func (e Entry) Expired(now time.Time) bool {
	return e.Expires != nil && now.After(*e.Expires)
}

// render produces the full authorized_keys line for e.
func (e Entry) render() string {
	tag := fmt.Sprintf("user=%s;sudo=%t;docker=%t;granted=%s",
		e.User, e.Sudo, e.Docker, e.Granted.UTC().Format(time.RFC3339))
	if e.Expires != nil {
		tag += ";expires=" + e.Expires.UTC().Format("2006-01-02")
	}
	return e.Key.Line() + " " + tagPrefix + tag
}

// parseEntry parses one authorized_keys line as a managed entry. ok is false
// for lines this package does not own (foreign keys, blanks, plain
// comments, or a managed-looking line whose key no longer parses) — callers
// must preserve those lines verbatim rather than altering them.
func parseEntry(line string) (Entry, bool) {
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return Entry{}, false
	}

	var tagField string
	for _, f := range fields[2:] {
		if strings.HasPrefix(f, tagPrefix) {
			tagField = strings.TrimPrefix(f, tagPrefix)
			break
		}
	}
	if tagField == "" {
		return Entry{}, false
	}

	key, err := ParsePublicKey(fields[0] + " " + fields[1])
	if err != nil {
		return Entry{}, false
	}

	e := Entry{Key: key}
	for _, kv := range strings.Split(tagField, ";") {
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "user":
			e.User = parts[1]
		case "sudo":
			e.Sudo, _ = strconv.ParseBool(parts[1])
		case "docker":
			e.Docker, _ = strconv.ParseBool(parts[1])
		case "granted":
			if t, err := time.Parse(time.RFC3339, parts[1]); err == nil {
				e.Granted = t
			}
		case "expires":
			if t, err := time.Parse("2006-01-02", parts[1]); err == nil {
				e.Expires = &t
			}
		}
	}
	if e.User == "" {
		return Entry{}, false
	}
	return e, true
}

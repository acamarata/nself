// Package compose: service toggle helper.
//
// Compose-aware plugin enable/disable for Model A.
//
// For Model A plugins (compose-managed services), enable/disable means
// commenting or uncommenting the service block in docker-compose.yml.
// We do this with a textual scanner rather than a full YAML parse so the
// operation is reversible and survives surrounding hand-edits.
//
// Conventions:
//
//   - A service block runs from the `<service-name>:` line at indentation 2
//     (two spaces) to the next service header (also at indentation 2) or the
//     next top-level key.
//   - To DISABLE: prefix every line of the service block with `# ` (the
//     existing indentation is preserved). A `# nself-disabled: <name>` marker
//     line is inserted immediately above the block so we can find it again.
//   - To ENABLE: locate the marker line, strip the `# ` prefix from each
//     subsequent block line, and remove the marker.
//
// Idempotency: ToggleService is safe to call repeatedly with the same target
// state. ToggleService(yaml, "claw", false) on an already-disabled service
// returns the input unchanged.
package compose

import (
	"bufio"
	"fmt"
	"strings"
)

// disabledMarkerPrefix is the leading text of the sentinel comment line we
// insert above a disabled service block so a later enable call can find it.
const disabledMarkerPrefix = "# nself-disabled: "

// ToggleService comments or uncomments the named service block in the supplied
// docker-compose YAML text. It returns the new YAML text and a bool reporting
// whether the file was changed.
//
// enable=true  → uncomment a previously disabled block (no-op if already enabled)
// enable=false → comment out an enabled block      (no-op if already disabled)
func ToggleService(yaml, serviceName string, enable bool) (string, bool, error) {
	if serviceName == "" {
		return yaml, false, fmt.Errorf("service name is empty")
	}

	lines := splitLines(yaml)

	if enable {
		out, changed := uncommentServiceBlock(lines, serviceName)
		return joinLines(out), changed, nil
	}
	out, changed := commentServiceBlock(lines, serviceName)
	return joinLines(out), changed, nil
}

// commentServiceBlock finds the named (enabled) service and prefixes every
// line of its block with `# `. Returns the input unchanged if the service is
// already disabled or absent.
func commentServiceBlock(lines []string, name string) ([]string, bool) {
	startIdx, endIdx := findServiceBlock(lines, name)
	if startIdx < 0 {
		// Already disabled? Look for the marker.
		if findDisabledMarker(lines, name) >= 0 {
			return lines, false
		}
		return lines, false
	}

	// Build output: marker + commented block.
	out := make([]string, 0, len(lines)+1)
	out = append(out, lines[:startIdx]...)
	out = append(out, disabledMarkerPrefix+name)
	for i := startIdx; i < endIdx; i++ {
		out = append(out, "# "+lines[i])
	}
	out = append(out, lines[endIdx:]...)
	return out, true
}

// uncommentServiceBlock locates a previously disabled block (via the marker)
// and strips the `# ` prefix from each commented line.
func uncommentServiceBlock(lines []string, name string) ([]string, bool) {
	markerIdx := findDisabledMarker(lines, name)
	if markerIdx < 0 {
		return lines, false
	}

	// The block starts at markerIdx+1 and runs until the first line that does
	// NOT begin with `# ` (or `#`). We restrict to a reasonable cap to avoid
	// an unbounded walk if the marker is orphaned.
	out := make([]string, 0, len(lines))
	out = append(out, lines[:markerIdx]...)

	i := markerIdx + 1
	for ; i < len(lines); i++ {
		line := lines[i]
		switch {
		case strings.HasPrefix(line, "# "):
			out = append(out, strings.TrimPrefix(line, "# "))
		case line == "#":
			out = append(out, "")
		default:
			// End of disabled block.
			goto done
		}
	}
done:
	out = append(out, lines[i:]...)
	return out, true
}

// findServiceBlock returns the [start, end) range of the named ENABLED
// service block in the lines slice, or (-1, -1) if not found.
//
// A service block starts at a line matching `^  <name>:` (two-space indent,
// trailing colon) and ends at the next line that is either:
//   - another two-space-indented service header, or
//   - a top-level key (no indent).
func findServiceBlock(lines []string, name string) (int, int) {
	header := "  " + name + ":"
	startIdx := -1
	inServices := false
	for i, line := range lines {
		if strings.HasPrefix(line, "services:") {
			inServices = true
			continue
		}
		if !inServices {
			continue
		}
		// Top-level key (no indent, contains ':') ends the services map.
		if len(line) > 0 && line[0] != ' ' && line[0] != '\t' && line[0] != '#' && strings.Contains(line, ":") {
			if startIdx >= 0 {
				return startIdx, i
			}
			inServices = false
			continue
		}
		if line == header || strings.HasPrefix(line, header+" ") {
			if startIdx >= 0 {
				// Found the next service header — close out the previous.
				return startIdx, i
			}
			startIdx = i
			continue
		}
		// Another two-space service header (different name) closes the block.
		if startIdx >= 0 && isTwoSpaceServiceHeader(line) {
			return startIdx, i
		}
	}
	if startIdx >= 0 {
		return startIdx, len(lines)
	}
	return -1, -1
}

// isTwoSpaceServiceHeader reports whether a line is a top-level service entry
// in the services: map (exactly two spaces of leading indent + name + colon).
func isTwoSpaceServiceHeader(line string) bool {
	if !strings.HasPrefix(line, "  ") {
		return false
	}
	if strings.HasPrefix(line, "   ") {
		return false
	}
	trimmed := strings.TrimLeft(line, " ")
	if trimmed == "" || strings.HasPrefix(trimmed, "#") {
		return false
	}
	idx := strings.Index(trimmed, ":")
	if idx <= 0 {
		return false
	}
	// Anything after the colon must be empty or whitespace (i.e. it is a
	// header line, not `key: value`).
	rest := strings.TrimSpace(trimmed[idx+1:])
	return rest == ""
}

// findDisabledMarker returns the index of the `# nself-disabled: <name>`
// marker line, or -1 if absent.
func findDisabledMarker(lines []string, name string) int {
	target := disabledMarkerPrefix + name
	for i, line := range lines {
		if strings.TrimRight(line, " \t") == target {
			return i
		}
	}
	return -1
}

// splitLines splits on \n, preserving empty lines, without dropping the final
// element when the input ends in \n.
func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}
	var lines []string
	scanner := bufio.NewScanner(strings.NewReader(s))
	scanner.Buffer(make([]byte, 1<<20), 1<<22)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	// Preserve trailing newline as an empty final element if the input ended in \n.
	if strings.HasSuffix(s, "\n") {
		lines = append(lines, "")
	}
	return lines
}

func joinLines(lines []string) string {
	return strings.Join(lines, "\n")
}

package build

// nginx_server_blocks.go — a minimal reader for the generated nginx site
// configs, used to detect two server blocks claiming one FQDN on one port.
//
// Purpose: split rendered nginx config text into server blocks and report
// the listen ports and server_names each one declares.
// Inputs: the text of a .conf file under nginx/sites/.
// Outputs: []nginxServerBlock, one per `server { ... }` in the file.
// Constraints: a targeted reader for nginx site configs, not a general
// nginx parser — it tokenizes on { } ; and reads two directives, and does
// not evaluate includes, maps, or upstreams. It works on both the
// multi-line form this repo generates and the compact single-line form a
// hand-written or plugin-shipped conf may use, because it tokenizes rather
// than matching at the start of a line. Anything it cannot interpret it
// ignores, so the worst case is a missed conflict, never a fabricated one.

import (
	"strconv"
	"strings"
)

// nginxServerBlock is one `server { ... }` block: the ports it listens on
// and the names it answers for.
type nginxServerBlock struct {
	Ports       []string
	ServerNames []string
}

// defaultListenPort is what nginx assumes when a server block declares a
// server_name but no listen directive at all.
const defaultListenPort = "80"

// parseServerBlocks splits content into its `server { ... }` blocks.
//
// Directives are attributed to the block that encloses them, so a file with
// an http-only block and an https block is read as two blocks rather than
// one merged set. That distinction matters: nginx only calls it a conflict
// when two blocks collide on the same name AND the same port.
func parseServerBlocks(content string) []nginxServerBlock {
	var blocks []nginxServerBlock
	// stack of open blocks; the element is the index into blocks for a
	// server block, or -1 for any other context (http, location, upstream).
	var stack []int
	var buf strings.Builder

	flushHeader := func() string {
		h := strings.TrimSpace(buf.String())
		buf.Reset()
		return h
	}

	for _, ch := range stripComments(content) {
		switch ch {
		case '{':
			header := flushHeader()
			idx := -1
			if firstWord(header) == "server" {
				blocks = append(blocks, nginxServerBlock{})
				idx = len(blocks) - 1
			}
			stack = append(stack, idx)
		case '}':
			buf.Reset()
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case ';':
			directive := flushHeader()
			// Attribute the directive to the nearest enclosing server block.
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i] < 0 {
					continue
				}
				b := &blocks[stack[i]]
				if names := serverNamesIn(directive); len(names) > 0 {
					b.ServerNames = append(b.ServerNames, names...)
				} else if port, ok := listenPortIn(directive); ok {
					b.Ports = append(b.Ports, port)
				}
				break
			}
		default:
			buf.WriteRune(ch)
		}
	}
	return blocks
}

// stripComments removes `#` comments, which run to end of line in nginx.
func stripComments(content string) string {
	var out strings.Builder
	for _, line := range strings.Split(content, "\n") {
		if idx := strings.Index(line, "#"); idx != -1 {
			line = line[:idx]
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return out.String()
}

// firstWord returns the first whitespace-separated word of s.
func firstWord(s string) string {
	if f := strings.Fields(s); len(f) > 0 {
		return f[0]
	}
	return ""
}

// serverNamesIn returns the domains declared by a `server_name` directive,
// or nil if directive is something else. Multiple space-separated names are
// handled; `server_names_hash_bucket_size` is not a match.
func serverNamesIn(directive string) []string {
	fields := strings.Fields(directive)
	if len(fields) < 2 || fields[0] != "server_name" {
		return nil
	}
	return fields[1:]
}

// listenPortIn returns the port from a `listen` directive. Handles the
// forms this repo generates and the common hand-written ones: "listen 443
// ssl", "listen [::]:443 ssl", "listen 80", "listen 0.0.0.0:8080
// default_server". A unix socket, or an address with no port, returns
// ok=false.
func listenPortIn(directive string) (string, bool) {
	fields := strings.Fields(directive)
	if len(fields) < 2 || fields[0] != "listen" {
		return "", false
	}
	addr := fields[1]
	if strings.HasPrefix(addr, "unix:") {
		return "", false
	}
	// "[::]:443" and "0.0.0.0:8080" — the port follows the last colon.
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		addr = addr[idx+1:]
	}
	if _, err := strconv.Atoi(addr); err != nil {
		return "", false
	}
	return addr, true
}

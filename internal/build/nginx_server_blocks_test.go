package build

// nginx_server_blocks_test.go — coverage for the shared nginx site-config
// reader that both server_name conflict checks are built on.
//
// Purpose: prove the reader attributes listen/server_name to the right
// block, reads the compact single-line form as well as the generated
// multi-line form, and does not mistake `upstream`/`location` blocks or
// `server_names_hash_bucket_size` for a server block or a server_name.
// A conflict check is only as good as this parser: anything it fails to
// read is a conflict that ships silently.

import "testing"

func TestParseServerBlocks_MultiLineAndCompactAgree(t *testing.T) {
	multi := `
server {
    listen 443 ssl;
    listen [::]:443 ssl;
    server_name api.example.com;
    location / {
        proxy_pass http://hasura:8080;
    }
}
`
	compact := `server { listen 443 ssl; listen [::]:443 ssl; server_name api.example.com; location / { proxy_pass http://hasura:8080; } }`

	for label, content := range map[string]string{"multi-line": multi, "compact": compact} {
		blocks := parseServerBlocks(content)
		if len(blocks) != 1 {
			t.Errorf("%s: got %d server blocks, want 1: %+v", label, len(blocks), blocks)
			continue
		}
		if len(blocks[0].ServerNames) != 1 || blocks[0].ServerNames[0] != "api.example.com" {
			t.Errorf("%s: ServerNames = %v, want [api.example.com]", label, blocks[0].ServerNames)
		}
		if len(blocks[0].Ports) != 2 || blocks[0].Ports[0] != "443" || blocks[0].Ports[1] != "443" {
			t.Errorf("%s: Ports = %v, want [443 443]", label, blocks[0].Ports)
		}
	}
}

func TestParseServerBlocks_SeparatesBlocks(t *testing.T) {
	content := `
upstream up_api {
    server hasura:8080;
}

server {
    listen 80;
    server_name api.example.com;
}

server {
    listen 443 ssl;
    server_name docs.example.com;
}
`
	blocks := parseServerBlocks(content)
	if len(blocks) != 2 {
		t.Fatalf("got %d blocks, want 2 (the upstream block must not count): %+v", len(blocks), blocks)
	}
	if blocks[0].ServerNames[0] != "api.example.com" || blocks[0].Ports[0] != "80" {
		t.Errorf("block 0 = %+v, want api.example.com on 80", blocks[0])
	}
	if blocks[1].ServerNames[0] != "docs.example.com" || blocks[1].Ports[0] != "443" {
		t.Errorf("block 1 = %+v, want docs.example.com on 443", blocks[1])
	}
}

func TestParseServerBlocks_IgnoresLookalikes(t *testing.T) {
	content := `
http {
    server_names_hash_bucket_size 128;
    # server_name commented.example.com;
    server {
        server_name real.example.com;
    }
}
`
	blocks := parseServerBlocks(content)
	if len(blocks) != 1 {
		t.Fatalf("got %d blocks, want 1: %+v", len(blocks), blocks)
	}
	got := blocks[0].ServerNames
	if len(got) != 1 || got[0] != "real.example.com" {
		t.Errorf("ServerNames = %v, want [real.example.com] — a commented-out directive and server_names_hash_bucket_size must not register", got)
	}
}

func TestListenPortIn(t *testing.T) {
	cases := map[string]struct {
		port string
		ok   bool
	}{
		"listen 80":                          {"80", true},
		"listen 443 ssl":                     {"443", true},
		"listen [::]:443 ssl":                {"443", true},
		"listen 0.0.0.0:8080 default_server": {"8080", true},
		"listen unix:/var/run/n.sock":        {"", false},
		"server_name api.example.com":        {"", false},
	}
	for directive, want := range cases {
		port, ok := listenPortIn(directive)
		if ok != want.ok || port != want.port {
			t.Errorf("listenPortIn(%q) = (%q, %v), want (%q, %v)", directive, port, ok, want.port, want.ok)
		}
	}
}

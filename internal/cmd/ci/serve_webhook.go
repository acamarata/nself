// Package ci — serve_webhook.go
//
// Purpose: GitHub webhook handler: signature verification, payload parsing,
//
//	ref/SHA extraction, and async job dispatch to the worker pool.
//
// Inputs:  HTTP POST with X-Hub-Signature-256 + X-GitHub-Event headers
// Outputs: 202 Accepted (async dispatch) or error status; gate job enqueued
// Constraints: HMAC-SHA256 with shared secret; supports push + pull_request
//
//	event types; ignores deleted-branch pushes; SPORT CLI-CMD-CI-SERVE-001
package ci

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/ui"
)

// webhookHandler processes GitHub webhook POST requests.
type webhookHandler struct {
	secret     string
	sem        chan struct{}
	binaryPath string
	cfg        ServeConfig
}

// githubPushPayload is the minimal subset of a GitHub push event we need.
type githubPushPayload struct {
	After   string `json:"after"`   // commit SHA ("0000..." on delete)
	Ref     string `json:"ref"`     // "refs/heads/main"
	Deleted bool   `json:"deleted"` // true when branch is deleted
	Repo    struct {
		FullName string `json:"full_name"` // "owner/repo"
		CloneURL string `json:"clone_url"` // HTTPS clone URL
	} `json:"repository"`
}

// githubPRPayload is the minimal subset of a GitHub pull_request event.
type githubPRPayload struct {
	Action      string `json:"action"` // "opened", "synchronize", "reopened"
	PullRequest struct {
		Head struct {
			SHA  string `json:"sha"`
			Ref  string `json:"ref"`
			Repo struct {
				FullName string `json:"full_name"`
				CloneURL string `json:"clone_url"`
			} `json:"repo"`
		} `json:"head"`
	} `json:"pull_request"`
}

// ServeHTTP handles an inbound webhook POST.
func (h *webhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read body (cap at 10 MB — GitHub payloads are well under this).
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		http.Error(w, "cannot read body", http.StatusBadRequest)
		return
	}

	// Verify HMAC signature when a secret is configured.
	if h.secret != "" {
		if err := verifySignature(h.secret, r.Header.Get("X-Hub-Signature-256"), body); err != nil {
			ui.Warn(fmt.Sprintf("webhook signature mismatch from %s: %v", r.RemoteAddr, err))
			http.Error(w, "signature mismatch", http.StatusForbidden)
			return
		}
	}

	event := r.Header.Get("X-GitHub-Event")
	job, err := parseEvent(event, body)
	if err != nil {
		// Unknown or uninteresting event — ack silently.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"accepted":false,"reason":"` + err.Error() + `"}`))
		return
	}

	// Dispatch asynchronously.
	select {
	case h.sem <- struct{}{}:
		go func() {
			defer func() { <-h.sem }()
			runJob(job, h.binaryPath, h.cfg)
		}()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"accepted": "true",
			"repo":     job.RepoFullName,
			"ref":      job.Ref,
			"sha":      job.SHA[:min(7, len(job.SHA))],
		})
	default:
		// Pool full — return 503 so GitHub retries.
		http.Error(w, fmt.Sprintf("concurrency limit %d reached; retry later", cap(h.sem)), http.StatusServiceUnavailable)
	}
}

// ciJob holds the resolved parameters for one gate run.
type ciJob struct {
	RepoFullName string // "owner/repo"
	CloneURL     string // HTTPS clone URL
	Ref          string // "refs/heads/main"
	SHA          string // full commit SHA
	EventType    string // "push" | "pull_request"
	ReceivedAt   time.Time
}

// parseEvent extracts a ciJob from a GitHub webhook payload.
// Returns an error for unhandled events (caller sends 200 but skips dispatch).
func parseEvent(event string, body []byte) (ciJob, error) {
	switch event {
	case "push":
		var p githubPushPayload
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&p); err != nil {
			return ciJob{}, fmt.Errorf("bad push payload: %w", err)
		}
		// Ignore branch-delete pushes (SHA is all zeros).
		if p.Deleted || p.After == "" || strings.HasPrefix(p.After, "000000") {
			return ciJob{}, fmt.Errorf("delete push: skip")
		}
		return ciJob{
			RepoFullName: p.Repo.FullName,
			CloneURL:     p.Repo.CloneURL,
			Ref:          p.Ref,
			SHA:          p.After,
			EventType:    "push",
			ReceivedAt:   time.Now(),
		}, nil

	case "pull_request":
		var p githubPRPayload
		if err := json.NewDecoder(bytes.NewReader(body)).Decode(&p); err != nil {
			return ciJob{}, fmt.Errorf("bad pull_request payload: %w", err)
		}
		// Only run on opened / synchronize / reopened.
		switch p.Action {
		case "opened", "synchronize", "reopened":
		default:
			return ciJob{}, fmt.Errorf("pr action %q: skip", p.Action)
		}
		head := p.PullRequest.Head
		return ciJob{
			RepoFullName: head.Repo.FullName,
			CloneURL:     head.Repo.CloneURL,
			Ref:          "refs/heads/" + head.Ref,
			SHA:          head.SHA,
			EventType:    "pull_request",
			ReceivedAt:   time.Now(),
		}, nil

	default:
		return ciJob{}, fmt.Errorf("event %q: skip", event)
	}
}

// verifySignature checks the X-Hub-Signature-256 header against body using secret.
// Header format: "sha256=<hex>".
func verifySignature(secret, header string, body []byte) error {
	if !strings.HasPrefix(header, "sha256=") {
		return fmt.Errorf("missing or malformed X-Hub-Signature-256 header")
	}
	sig, err := hex.DecodeString(strings.TrimPrefix(header, "sha256="))
	if err != nil {
		return fmt.Errorf("invalid hex in signature: %w", err)
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := mac.Sum(nil)
	if !hmac.Equal(expected, sig) {
		return fmt.Errorf("signature mismatch")
	}
	return nil
}

// min returns the smaller of a and b (Go 1.21+ has builtin min; keep explicit for 1.22 compat).

package mail

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// helper: build a Client pointed at a test server.
func testClient(srv *httptest.Server, key string) *Client {
	c := New(srv.URL, key)
	return c
}

func TestSend_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/mail/send" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("auth header missing: %q", got)
		}
		var req SendRequest
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("bad body: %v", err)
		}
		if req.To != "user@example.com" || req.Subject != "Hi" || req.Body != "test" {
			t.Fatalf("body fields wrong: %+v", req)
		}
		_ = json.NewEncoder(w).Encode(SendResponse{
			MessageID: "msg-123",
			Accepted:  true,
			To:        req.To,
		})
	}))
	defer srv.Close()

	c := testClient(srv, "test-key")
	resp, err := c.Send(context.Background(), SendRequest{
		To:      "user@example.com",
		Subject: "Hi",
		Body:    "test",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if resp.MessageID != "msg-123" || !resp.Accepted {
		t.Fatalf("unexpected resp: %+v", resp)
	}
}

func TestSend_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	c := testClient(srv, "bad-key")
	_, err := c.Send(context.Background(), SendRequest{To: "x@y.z", Subject: "s", Body: "b"})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got: %v", err)
	}
}

func TestSend_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"boom"}`))
	}))
	defer srv.Close()

	c := testClient(srv, "key")
	_, err := c.Send(context.Background(), SendRequest{To: "x@y.z", Subject: "s", Body: "b"})
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got: %v", err)
	}
}

func TestStatus_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("expected GET, got %s", r.Method)
		}
		if !strings.Contains(r.URL.RawQuery, "message_id=msg-1") {
			t.Fatalf("missing message_id query: %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(StatusResponse{
			MessageID:   "msg-1",
			Status:      "delivered",
			DeliveredAt: "2026-04-27T00:00:00Z",
		})
	}))
	defer srv.Close()

	c := testClient(srv, "k")
	resp, err := c.Status(context.Background(), "msg-1")
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if resp.Status != "delivered" {
		t.Fatalf("expected delivered, got %q", resp.Status)
	}
}

func TestListTemplates_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ListTemplatesResponse{
			Templates: []Template{
				{ID: "t1", Name: "Welcome", Provider: "postmark"},
				{ID: "t2", Name: "Receipt", Provider: "postmark"},
			},
		})
	}))
	defer srv.Close()

	c := testClient(srv, "k")
	tmpls, err := c.ListTemplates(context.Background())
	if err != nil {
		t.Fatalf("list templates: %v", err)
	}
	if len(tmpls) != 2 || tmpls[0].ID != "t1" {
		t.Fatalf("unexpected templates: %+v", tmpls)
	}
}

func TestVerifyDKIM_Valid(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "domain=example.com") {
			t.Fatalf("missing domain query: %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(DKIMVerifyResponse{
			Domain:   "example.com",
			Valid:    true,
			Selector: "pm._domainkey",
			Record:   "v=DKIM1; k=rsa; p=...",
		})
	}))
	defer srv.Close()

	c := testClient(srv, "k")
	resp, err := c.VerifyDKIM(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("verify dkim: %v", err)
	}
	if !resp.Valid || resp.Selector != "pm._domainkey" {
		t.Fatalf("unexpected dkim response: %+v", resp)
	}
}

func TestBroadcast_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/mail/broadcast" {
			t.Fatalf("unexpected req: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(BroadcastResponse{
			BatchID:    "batch-9",
			Recipients: 42,
			Queued:     true,
		})
	}))
	defer srv.Close()

	c := testClient(srv, "k")
	resp, err := c.Broadcast(context.Background(), BroadcastRequest{
		ListID:     "list-a",
		TemplateID: "tpl-b",
	})
	if err != nil {
		t.Fatalf("broadcast: %v", err)
	}
	if resp.Recipients != 42 || resp.BatchID != "batch-9" {
		t.Fatalf("unexpected broadcast: %+v", resp)
	}
}

func TestNew_DefaultBaseURL(t *testing.T) {
	c := New("", "k")
	if c.BaseURL != strings.TrimRight(DefaultPingURL, "/") {
		t.Fatalf("expected default base URL, got %q", c.BaseURL)
	}
}

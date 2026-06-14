package webhook

// ssrf.go — lightweight SSRF URL validation for the webhook dispatcher.
// Blocks outbound HTTP requests to private/loopback/link-local/metadata addresses.
// Adapted from plugins-pro/paid/notify/go/internal/push/ssrf.go (DRY reuse, not duplication).
// Both the pre-flight ValidateWebhookURL check and the dial-time safeDialContext guard
// must pass before any outbound webhook delivery succeeds.

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var blockedNets []*net.IPNet

func init() {
	cidrs := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
		"127.0.0.0/8",
		"::1/128",
		"169.254.0.0/16",
		"fe80::/10",
		"169.254.169.254/32", // AWS/GCP/Azure/DO metadata
		"100.100.100.200/32", // Alibaba metadata
		"100.64.0.0/10",      // CGNAT shared address space (RFC 6598)
		"fc00::/7",
		"::/128",
	}
	for _, cidr := range cidrs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Fatalf("webhook ssrf: bad CIDR %q: %v", cidr, err)
		}
		blockedNets = append(blockedNets, ipNet)
	}
}

func isBlockedIP(ip net.IP) bool {
	ip = ip.To16()
	if ip == nil {
		return true
	}
	for _, n := range blockedNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// allowlisted returns true when hostname exactly matches an entry in the
// SSRF_ALLOWED_HOSTS environment variable (comma-separated list of hostnames).
// The scheme check in ValidateWebhookURL still applies to allowlisted hosts.
func allowlisted(hostname string) bool {
	list := os.Getenv("SSRF_ALLOWED_HOSTS")
	if list == "" {
		return false
	}
	for _, h := range strings.Split(list, ",") {
		if strings.TrimSpace(h) == hostname {
			return true
		}
	}
	return false
}

// ValidateWebhookURL returns an error when rawURL targets a private/loopback/
// link-local or cloud-metadata address, or uses a non-http(s) scheme.
// Call this before dispatching any outbound webhook request.
//
// Set SSRF_ALLOWED_HOSTS=host1,host2 to allowlist specific hostnames (bypasses
// the IP/DNS check; scheme check always applies).
func ValidateWebhookURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("ssrf: invalid url: %w", err)
	}
	switch u.Scheme {
	case "http", "https":
		// allowed
	default:
		return fmt.Errorf("ssrf: scheme %q not allowed (only http/https)", u.Scheme)
	}
	hostname := u.Hostname()
	if hostname == "" {
		return fmt.Errorf("ssrf: empty hostname")
	}
	// Allowlist short-circuit: skip IP/DNS check for explicitly permitted hosts.
	if allowlisted(hostname) {
		return nil
	}
	// Literal IP address check.
	if ip := net.ParseIP(hostname); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("ssrf: IP %s is in a blocked range", ip)
		}
		return nil
	}
	// DNS resolution check.
	addrs, err := net.LookupHost(hostname)
	if err != nil {
		return fmt.Errorf("ssrf: dns lookup failed for %q: %w", hostname, err)
	}
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip == nil {
			return fmt.Errorf("ssrf: dns returned non-IP %q", addr)
		}
		if isBlockedIP(ip) {
			return fmt.Errorf("ssrf: hostname %q resolves to blocked IP %s", hostname, ip)
		}
	}
	return nil
}

// safeDialContext is a post-DNS DialContext guard — the second line of
// defence against DNS rebinding attacks. Re-resolves at dial time and connects
// to the first resolved IP explicitly to prevent re-resolution between the
// ValidateWebhookURL pre-flight and the actual TCP dial.
// Hosts in SSRF_ALLOWED_HOSTS bypass the IP block (mirrors ValidateWebhookURL behaviour).
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("ssrf: cannot parse dial address %q: %w", addr, err)
	}
	// Allowlist short-circuit: bypass IP block for explicitly permitted hosts.
	if allowlisted(host) {
		d := &net.Dialer{}
		return d.DialContext(ctx, network, addr)
	}
	ips, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("ssrf: dns lookup failed for %q: %w", host, err)
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("ssrf: dns returned no addresses for %q", host)
	}
	for _, ipStr := range ips {
		ip := net.ParseIP(ipStr)
		if ip == nil {
			continue
		}
		if isBlockedIP(ip) {
			return nil, fmt.Errorf("ssrf: dial to %s (%s) blocked", host, ip)
		}
	}
	d := &net.Dialer{}
	return d.DialContext(ctx, network, net.JoinHostPort(ips[0], port))
}

// NewWebhookClient returns an *http.Client with SSRF-safe transport for
// webhook delivery. Timeout defaults to 30 s when t == 0.
// Use this instead of a plain &http.Client{} for all outbound webhook requests.
// Provides defence-in-depth: dial-time IP validation supplements the
// pre-flight ValidateWebhookURL check against DNS-rebinding attacks.
func NewWebhookClient(t time.Duration) *http.Client {
	if t == 0 {
		t = 30 * time.Second
	}
	return &http.Client{
		Timeout: t,
		Transport: &http.Transport{
			DialContext:           safeDialContext,
			MaxIdleConns:          50,
			MaxIdleConnsPerHost:   5,
			IdleConnTimeout:       60 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

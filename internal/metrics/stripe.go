// Package metrics — stripe.go emits Prometheus metrics for Stripe webhook
// processing. These metrics enable alerting on webhook failures, latency,
// outbox depth, and signature attacks.
//
// Metrics (spec G-31-11):
//   - nself_stripe_event_received_total{type,result} (counter)
//   - nself_stripe_event_duration_seconds{type} (histogram via gauge of last)
//   - nself_stripe_outbox_depth (gauge)
//   - nself_stripe_signature_failures_total (counter)
package metrics

import (
	"fmt"
	"strings"
)

// StripeEventRecord captures a single Stripe webhook event for metric emission.
type StripeEventRecord struct {
	EventType   string  // e.g., "customer.subscription.created"
	Result      string  // "ok" or "error"
	DurationSec float64 // processing time in seconds
	TextfileDir string  // overrides DefaultTextfileDir when non-empty
}

// StripeOutboxRecord captures the current outbox state.
type StripeOutboxRecord struct {
	Depth       int    // number of events waiting in outbox
	TextfileDir string // overrides DefaultTextfileDir when non-empty
}

// StripeSignatureRecord captures a signature verification failure.
type StripeSignatureRecord struct {
	TextfileDir string // overrides DefaultTextfileDir when non-empty
}

// EmitStripeEvent writes stripe event processing metrics.
func EmitStripeEvent(r StripeEventRecord) error {
	dir := r.TextfileDir
	if dir == "" {
		dir = DefaultTextfileDir
	}

	receivedLabels := labelString(map[string]string{
		"type":   r.EventType,
		"result": firstNonEmpty(r.Result, "ok"),
	})

	durationLabels := labelString(map[string]string{
		"type": r.EventType,
	})

	var b strings.Builder

	fmt.Fprintln(&b, "# HELP nself_stripe_event_received_total Total Stripe webhook events received")
	fmt.Fprintln(&b, "# TYPE nself_stripe_event_received_total counter")
	fmt.Fprintf(&b, "nself_stripe_event_received_total%s 1\n", receivedLabels)

	fmt.Fprintln(&b, "# HELP nself_stripe_event_duration_seconds Processing time for Stripe webhook events")
	fmt.Fprintln(&b, "# TYPE nself_stripe_event_duration_seconds gauge")
	fmt.Fprintf(&b, "nself_stripe_event_duration_seconds%s %.6f\n", durationLabels, r.DurationSec)

	name := fmt.Sprintf("nself_stripe_event_%s.prom", sanitizeMetricName(r.EventType))
	return writeAtomic(dir+"/"+name, b.String())
}

// EmitStripeOutbox writes the current outbox depth gauge.
func EmitStripeOutbox(r StripeOutboxRecord) error {
	dir := r.TextfileDir
	if dir == "" {
		dir = DefaultTextfileDir
	}

	var b strings.Builder

	fmt.Fprintln(&b, "# HELP nself_stripe_outbox_depth Number of Stripe events waiting in durable outbox")
	fmt.Fprintln(&b, "# TYPE nself_stripe_outbox_depth gauge")
	fmt.Fprintf(&b, "nself_stripe_outbox_depth %d\n", r.Depth)

	return writeAtomic(dir+"/nself_stripe_outbox.prom", b.String())
}

// EmitStripeSignatureFailure increments the signature failure counter.
func EmitStripeSignatureFailure(r StripeSignatureRecord) error {
	dir := r.TextfileDir
	if dir == "" {
		dir = DefaultTextfileDir
	}

	var b strings.Builder

	fmt.Fprintln(&b, "# HELP nself_stripe_signature_failures_total Total Stripe webhook signature verification failures")
	fmt.Fprintln(&b, "# TYPE nself_stripe_signature_failures_total counter")
	fmt.Fprintf(&b, "nself_stripe_signature_failures_total 1\n")

	return writeAtomic(dir+"/nself_stripe_signature_failures.prom", b.String())
}

// sanitizeMetricName replaces dots with underscores for file naming.
func sanitizeMetricName(s string) string {
	return strings.ReplaceAll(s, ".", "_")
}

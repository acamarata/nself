package build

// Purpose: the Grafana dashboard JSON envelope/panel structs and the renderer
// that produces one dashboard per installed ɳSentry plugin. Split out of
// grafana_nsentry.go (CLI-R12) as a pure move.
// Inputs: a plugin slug (must exist in dashboardTitles, grafana_nsentry.go).
// Outputs: dashboard JSON bytes, deterministically indented for byte-stable
// re-builds.
// Constraints: T04 (status-page) appends an extra text panel linking to the
// public status summary endpoint — see the slug == "status-page" branch.

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// dashboardPanel is the minimal Grafana 10.x panel struct we emit. Real
// dashboards have dozens of fields; we ship a focused subset (request rate
// + error rate + p95 latency) per plugin that is valid JSON and renders.
type dashboardPanel struct {
	ID          int                 `json:"id"`
	Type        string              `json:"type"`
	Title       string              `json:"title"`
	Datasource  map[string]string   `json:"datasource"`
	GridPos     map[string]int      `json:"gridPos"`
	Targets     []map[string]string `json:"targets,omitempty"`
	Options     map[string]any      `json:"options,omitempty"`
	FieldConfig map[string]any      `json:"fieldConfig,omitempty"`
	Content     string              `json:"content,omitempty"` // text panel HTML
	Mode        string              `json:"mode,omitempty"`    // text panel: "html"|"markdown"
}

// dashboard is the top-level Grafana dashboard JSON envelope we emit.
type dashboard struct {
	UID           string            `json:"uid"`
	Title         string            `json:"title"`
	SchemaVersion int               `json:"schemaVersion"`
	Version       int               `json:"version"`
	Editable      bool              `json:"editable"`
	Tags          []string          `json:"tags"`
	Time          map[string]string `json:"time"`
	Refresh       string            `json:"refresh"`
	Panels        []dashboardPanel  `json:"panels"`
}

// RenderNSentryDashboard returns the dashboard JSON bytes for a single ɳSentry
// plugin slug. The dashboard contains three panels with PromQL queries filtered
// by bundle="nsentry" and plugin="<slug>":
//
//  1. Request rate (rate(http_requests_total{...}[5m]))
//  2. Error rate  (rate(http_requests_total{status=~"5.."}[5m]))
//  3. p95 latency (histogram_quantile(0.95, ...))
//
// When slug == "status-page", a fourth text panel (T04) is appended that
// links operators directly to the public status summary endpoint exposed by
// the nself-status-page plugin (GET <host>:3832/status). The link text uses
// a Grafana variable so the host substitutes correctly per environment.
func RenderNSentryDashboard(slug string) ([]byte, error) {
	title, ok := dashboardTitles[slug]
	if !ok {
		return nil, fmt.Errorf("grafana_nsentry: unknown plugin slug %q (add to dashboardTitles)", slug)
	}

	ds := map[string]string{"type": "prometheus", "uid": "nsentry-prometheus"}
	labelFilter := fmt.Sprintf(`{bundle="nsentry",plugin="%s"}`, slug)

	panels := []dashboardPanel{
		{
			ID:         1,
			Type:       "timeseries",
			Title:      "Request Rate",
			Datasource: ds,
			GridPos:    map[string]int{"h": 8, "w": 12, "x": 0, "y": 0},
			Targets: []map[string]string{
				{
					"refId": "A",
					"expr":  "sum(rate(http_requests_total" + labelFilter + "[5m]))",
				},
			},
		},
		{
			ID:         2,
			Type:       "timeseries",
			Title:      "Error Rate (5xx)",
			Datasource: ds,
			GridPos:    map[string]int{"h": 8, "w": 12, "x": 12, "y": 0},
			Targets: []map[string]string{
				{
					"refId": "A",
					"expr":  `sum(rate(http_requests_total{bundle="nsentry",plugin="` + slug + `",status=~"5.."}[5m]))`,
				},
			},
		},
		{
			ID:         3,
			Type:       "timeseries",
			Title:      "p95 Latency",
			Datasource: ds,
			GridPos:    map[string]int{"h": 8, "w": 24, "x": 0, "y": 8},
			Targets: []map[string]string{
				{
					"refId": "A",
					"expr":  `histogram_quantile(0.95, sum by (le) (rate(http_request_duration_seconds_bucket{bundle="nsentry",plugin="` + slug + `"}[5m])))`,
				},
			},
		},
	}

	// T04 — status-page integration: link to the public /status endpoint.
	// The plugin binds on port 3832 (F10-PORT-REGISTRY) and exposes a
	// summary view at GET /status. We embed a text panel so the dashboard
	// surfaces this immediately rather than burying it in a runbook.
	if slug == "status-page" {
		panels = append(panels, dashboardPanel{
			ID:         4,
			Type:       "text",
			Title:      "Public Status Page",
			Datasource: nil,
			GridPos:    map[string]int{"h": 4, "w": 24, "x": 0, "y": 16},
			Mode:       "markdown",
			Content: "Public status summary served by " +
				"`nself-status-page` on port **3832**.\n\n" +
				"- Live page: [http://${__org.name}:3832/status](http://${__org.name}:3832/status)\n" +
				"- RSS feed: [http://${__org.name}:3832/status.rss](http://${__org.name}:3832/status.rss)\n" +
				"- Atom feed: [http://${__org.name}:3832/status.atom](http://${__org.name}:3832/status.atom)\n",
		})
	}

	d := dashboard{
		UID:           "nsentry-" + slug,
		Title:         title,
		SchemaVersion: 38,
		Version:       1,
		Editable:      false,
		Tags:          []string{"nsentry", slug},
		Time:          map[string]string{"from": "now-6h", "to": "now"},
		Refresh:       "30s",
		Panels:        panels,
	}

	// Marshal with stable indent so files round-trip byte-identically on
	// re-build. json.Marshal already orders struct fields by declaration;
	// maps inside fields use canonical-key emission via Go's stdlib (Go
	// 1.12+ sorts map keys in json.Marshal output).
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(d); err != nil {
		return nil, fmt.Errorf("grafana_nsentry: marshal dashboard %s: %w", slug, err)
	}
	return buf.Bytes(), nil
}

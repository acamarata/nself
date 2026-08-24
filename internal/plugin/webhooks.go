package plugin

import (
	"encoding/json"
	"fmt"
	"sort"
)

// WebhookNames is the list of webhook event names a plugin declares.
//
// Purpose: accept both shapes real plugin.json files use. The field was typed
// `[]string`, but manifests in the wild use an object as often as an array —
// `{"progress.updated": "Update playback position"}` alongside
// `["acquisition.subscribed"]`. Unmarshalling an object into []string fails,
// and ListInstalled skips any plugin whose manifest fails to parse, so those
// plugins vanished silently from `nself plugin list`, `nself costs`, and
// everything else that enumerates installed plugins. Nothing reported an
// error; the plugin was simply absent.
//
// Inputs: the `webhooks` value from plugin.json, in either shape.
//
// Outputs: the event names. For the object form the keys are the event names
// and the values are human descriptions, so the keys are what this carries.
//
// Constraints: unmarshalling must never fail on a shape a real manifest uses —
// a manifest this code cannot read makes a plugin invisible rather than loud.
type WebhookNames []string

// UnmarshalJSON accepts an array of names, an object keyed by name, or null.
func (w *WebhookNames) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*w = nil
		return nil
	}

	// Array form: ["event.a", "event.b"]
	var asList []string
	if err := json.Unmarshal(data, &asList); err == nil {
		*w = asList
		return nil
	}

	// Object form: {"event.a": "description", ...} — keys are the events.
	var asMap map[string]string
	if err := json.Unmarshal(data, &asMap); err == nil {
		names := make([]string, 0, len(asMap))
		for name := range asMap {
			names = append(names, name)
		}
		// Map iteration order is random; sort so manifests round-trip stably.
		sort.Strings(names)
		*w = names
		return nil
	}

	return fmt.Errorf("webhooks: expected an array of event names or an object keyed by event name, got %s", truncateJSON(data))
}

// MarshalJSON always writes the array form, which is the documented shape.
func (w WebhookNames) MarshalJSON() ([]byte, error) {
	if w == nil {
		return []byte("null"), nil
	}
	return json.Marshal([]string(w))
}

// truncateJSON keeps an error message readable when a manifest field is large.
func truncateJSON(data []byte) string {
	const max = 80
	if len(data) <= max {
		return string(data)
	}
	return string(data[:max]) + "..."
}

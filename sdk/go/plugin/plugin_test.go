package plugin

import "testing"

func TestInfoValidate(t *testing.T) {
	cases := []struct {
		name    string
		in      Info
		wantErr bool
	}{
		{"valid free", Info{Name: "chat", Version: "1.0.0", Tier: "free"}, false},
		{"valid pro", Info{Name: "ai", Version: "1.0.0", Tier: "pro"}, false},
		{"missing name", Info{Version: "1.0.0", Tier: "free"}, true},
		{"missing version", Info{Name: "ai", Tier: "pro"}, true},
		{"bad tier", Info{Name: "ai", Version: "1.0.0", Tier: "maybe"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := c.in.Validate()
			if (err != nil) != c.wantErr {
				t.Errorf("err=%v wantErr=%v", err, c.wantErr)
			}
		})
	}
}

func TestBaseDefaults(t *testing.T) {
	b := &Base{PluginInfo: Info{Name: "ai", Version: "1.0.0", Tier: "pro"}}
	if got := b.Info().Name; got != "ai" {
		t.Errorf("Info.Name=%q, want ai", got)
	}
	if b.Uptime() != 0 {
		t.Errorf("Uptime on zero StartedAt should be 0")
	}
}

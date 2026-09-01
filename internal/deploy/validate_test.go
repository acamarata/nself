package deploy

import "testing"

// TestValidateRemotePath_RejectsInjectionPayloads (T31) asserts the shared
// remote-path validator rejects shell metacharacters that would otherwise
// reach a remote shell via DeployViaSsh's fmt.Sprintf-built commands.
func TestValidateRemotePath_RejectsInjectionPayloads(t *testing.T) {
	payloads := []string{
		"/opt/x; id",
		"/opt/x$(id)",
		"/opt/x`id`",
		"/opt/x | id",
		"/opt/x && id",
		"/opt/x\nid",
		"/opt/x id",
	}
	for _, p := range payloads {
		if err := ValidateRemotePath(p); err == nil {
			t.Errorf("ValidateRemotePath(%q): expected error, got nil", p)
		}
	}
}

// TestValidateRemotePath_AcceptsLegitimatePaths proves no regression for the
// paths real deployments use.
func TestValidateRemotePath_AcceptsLegitimatePaths(t *testing.T) {
	ok := []string{
		"",
		"/opt/nself",
		"/opt/nself-staging",
		"/home/ubuntu/app_v2",
		"opt/nself",
		"/opt/nself.d",
	}
	for _, p := range ok {
		if err := ValidateRemotePath(p); err != nil {
			t.Errorf("ValidateRemotePath(%q): unexpected error: %v", p, err)
		}
	}
}

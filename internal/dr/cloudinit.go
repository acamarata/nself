package dr

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

// semverRe accepts optional leading "v" followed by MAJOR.MINOR.PATCH.
// Rejects any string containing shell metacharacters or URL-encoded sequences
// because the version is interpolated into a curl|sh runcmd string in the
// cloud-init YAML.
var semverRe = regexp.MustCompile(`^v?\d+\.\d+\.\d+$`)

// validateSemver returns an error when v does not match a strict semver pattern.
// An empty string is allowed (means "latest") and passes validation.
func validateSemver(v string) error {
	if v == "" {
		return nil
	}
	if !semverRe.MatchString(v) {
		return fmt.Errorf("NselfVersion %q is not a valid semver (want v?MAJOR.MINOR.PATCH)", v)
	}
	return nil
}

// CloudInitParams feeds the drill VM user-data template. The resulting
// cloud-init YAML installs Docker and the nSelf CLI, then runs the drill
// entrypoint which restores the latest backup and executes the smoke suite.
type CloudInitParams struct {
	DrillID        string
	BackupID       string
	B2Bucket       string
	B2KeyID        string
	B2AppKey       string
	AgeKeyMaterial string // contents of age-key.txt; embedded, never logged
	SSHPublicKey   string
	ReporterURL    string // nclaw-prod API endpoint that receives the report
	ReporterToken  string
	NselfVersion   string // e.g. v1.0.3; empty means latest
}

const cloudInitTemplate = `#cloud-config
# nSelf DR drill VM user-data (drill_id={{ .DrillID }})
# This file is re-generated every drill run. Do not hand-edit.

hostname: nself-dr-drill
manage_etc_hosts: true

users:
  - name: root
    ssh_authorized_keys:
      - {{ .SSHPublicKey }}

package_update: true
packages:
  - ca-certificates
  - curl
  - gnupg
  - jq
  - age

write_files:
  - path: /etc/nself/age-key.txt
    permissions: "0600"
    owner: root:root
    content: |
      {{ .AgeKeyMaterial | indent6 }}
  - path: /etc/nself/dr.env
    permissions: "0600"
    owner: root:root
    content: |
      DR_DRILL_ID={{ .DrillID }}
      DR_BACKUP_ID={{ .BackupID }}
      B2_BUCKET={{ .B2Bucket }}
      B2_KEY_ID={{ .B2KeyID }}
      B2_APP_KEY={{ .B2AppKey }}
      DR_REPORTER_URL={{ .ReporterURL }}
      DR_REPORTER_TOKEN={{ .ReporterToken }}

runcmd:
  - [ sh, -c, "curl -fsSL https://get.docker.com | sh" ]
  - [ sh, -c, "curl -fsSL https://install.nself.org{{ if .NselfVersion }}?v={{ .NselfVersion }}{{ end }} | sh" ]
  - [ sh, -c, "systemctl enable --now docker" ]
  - [ sh, -c, "nself dr restore $DR_BACKUP_ID --target / --decrypt-key /etc/nself/age-key.txt --yes" ]
  - [ sh, -c, "nself dr smoke --report-to $DR_REPORTER_URL --token $DR_REPORTER_TOKEN --drill-id $DR_DRILL_ID" ]
`

// RenderCloudInit returns the cloud-init user-data YAML for a drill VM.
// The output is deterministic for a given params set so that operators can
// diff the rendered YAML against the last known good template when a drill
// fails with a cloud-init error (see the fail-fix playbook).
func RenderCloudInit(p CloudInitParams) (string, error) {
	if p.DrillID == "" {
		return "", fmt.Errorf("drill_id required")
	}
	if p.SSHPublicKey == "" {
		return "", fmt.Errorf("ssh_public_key required")
	}
	if err := validateSemver(p.NselfVersion); err != nil {
		return "", err
	}
	funcs := template.FuncMap{
		"indent6": func(s string) string {
			lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
			for i, l := range lines {
				lines[i] = "      " + l
			}
			return strings.Join(lines, "\n")
		},
	}
	tmpl, err := template.New("cloudinit").Funcs(funcs).Parse(cloudInitTemplate)
	if err != nil {
		return "", fmt.Errorf("parse cloudinit template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, p); err != nil {
		return "", fmt.Errorf("render cloudinit: %w", err)
	}
	return buf.String(), nil
}

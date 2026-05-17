package destinations

import (
	"testing"
)

func TestIsRemoteKey(t *testing.T) {
	cases := []struct {
		key  string
		want bool
	}{
		{"s3://my-bucket/backup.tar.gz", true},
		{"r2://bucket/archive/base.tar.gz", true},
		{"minio://bucket/path/to/file", true},
		{"b2://bucket/backup.tar.gz", true},
		{"gcs://bucket/backup.tar.gz", true},
		{"az://container/backup.tar.gz", true},
		{"/local/path/backup.tar.gz", false},
		{"./relative/path.tar.gz", false},
		{"backup.tar.gz", false},
		{"", false},
		{"ftp://not-supported", false},
	}

	for _, tc := range cases {
		got := IsRemoteKey(tc.key)
		if got != tc.want {
			t.Errorf("IsRemoteKey(%q) = %v; want %v", tc.key, got, tc.want)
		}
	}
}

func TestToRclonePath(t *testing.T) {
	cases := []struct {
		uri     string
		want    string
		wantErr bool
	}{
		{"s3://my-bucket/path/to/object.tar.gz", "s3:my-bucket/path/to/object.tar.gz", false},
		{"r2://bucket/archive/base.tar.gz", "r2:bucket/archive/base.tar.gz", false},
		{"minio://local-bucket/backups/base.tar.gz", "minio:local-bucket/backups/base.tar.gz", false},
		{"b2://bucket/backup.tar.gz", "b2:bucket/backup.tar.gz", false},
		{"gcs://bucket/backup.tar.gz", "gcs:bucket/backup.tar.gz", false},
		{"az://container/blob.tar.gz", "az:container/blob.tar.gz", false},
		// Error cases.
		{"ftp://bucket/file", "", true},
		{"s3://", "", true},
		{"", "", true},
	}

	for _, tc := range cases {
		got, err := toRclonePath(tc.uri)
		if tc.wantErr {
			if err == nil {
				t.Errorf("toRclonePath(%q): expected error, got %q", tc.uri, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("toRclonePath(%q): unexpected error: %v", tc.uri, err)
			continue
		}
		if got != tc.want {
			t.Errorf("toRclonePath(%q) = %q; want %q", tc.uri, got, tc.want)
		}
	}
}

func TestFetchRemote_EmptyKey(t *testing.T) {
	_, err := FetchRemote(t.Context(), "", t.TempDir())
	if err == nil {
		t.Fatal("expected error for empty remote key, got nil")
	}
}

func TestFetchRemote_EmptyDestDir(t *testing.T) {
	_, err := FetchRemote(t.Context(), "s3://bucket/file.tar.gz", "")
	if err == nil {
		t.Fatal("expected error for empty destDir, got nil")
	}
}

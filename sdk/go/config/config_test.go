package config

import (
	"os"
	"testing"
	"time"
)

func TestEnv(t *testing.T) {
	os.Setenv("SDKTEST_X", "hello")
	t.Cleanup(func() { os.Unsetenv("SDKTEST_X") })
	if got := Env("SDKTEST_X", "def"); got != "hello" {
		t.Errorf("Env got %q, want hello", got)
	}
	if got := Env("SDKTEST_MISSING", "def"); got != "def" {
		t.Errorf("Env default got %q, want def", got)
	}
}

func TestEnvRequired(t *testing.T) {
	if _, err := EnvRequired("SDKTEST_NOTSET"); err == nil {
		t.Errorf("expected error for unset var")
	}
	os.Setenv("SDKTEST_SET", "v")
	t.Cleanup(func() { os.Unsetenv("SDKTEST_SET") })
	v, err := EnvRequired("SDKTEST_SET")
	if err != nil || v != "v" {
		t.Errorf("EnvRequired v=%q err=%v", v, err)
	}
}

func TestEnvInt(t *testing.T) {
	os.Setenv("SDKTEST_N", "42")
	t.Cleanup(func() { os.Unsetenv("SDKTEST_N") })
	if got := EnvInt("SDKTEST_N", 0); got != 42 {
		t.Errorf("EnvInt got %d, want 42", got)
	}
	os.Setenv("SDKTEST_BAD", "not-a-number")
	t.Cleanup(func() { os.Unsetenv("SDKTEST_BAD") })
	if got := EnvInt("SDKTEST_BAD", 7); got != 7 {
		t.Errorf("EnvInt bad got %d, want 7 (default)", got)
	}
}

func TestEnvBool(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want bool
	}{
		{"true", true}, {"1", true}, {"yes", true}, {"on", true},
		{"false", false}, {"0", false}, {"no", false},
	} {
		os.Setenv("SDKTEST_B", tc.in)
		if got := EnvBool("SDKTEST_B", !tc.want); got != tc.want {
			t.Errorf("EnvBool(%q)=%v, want %v", tc.in, got, tc.want)
		}
	}
	os.Unsetenv("SDKTEST_B")
	if got := EnvBool("SDKTEST_B", true); got != true {
		t.Errorf("EnvBool default should be true")
	}
}

func TestEnvDuration(t *testing.T) {
	os.Setenv("SDKTEST_D", "30s")
	t.Cleanup(func() { os.Unsetenv("SDKTEST_D") })
	if got := EnvDuration("SDKTEST_D", time.Minute); got != 30*time.Second {
		t.Errorf("EnvDuration got %v, want 30s", got)
	}
}

func TestEnvList(t *testing.T) {
	os.Setenv("SDKTEST_L", " a, b , ,c ")
	t.Cleanup(func() { os.Unsetenv("SDKTEST_L") })
	got := EnvList("SDKTEST_L")
	want := []string{"a", "b", "c"}
	if len(got) != len(want) {
		t.Fatalf("EnvList len %d, want %d: %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("EnvList[%d]=%q, want %q", i, got[i], want[i])
		}
	}
}

package sdk

import "testing"

func TestParseSemVer(t *testing.T) {
	cases := []struct {
		in      string
		wantMaj int
		wantMin int
		wantPat int
		wantErr bool
	}{
		{"1.2.3", 1, 2, 3, false},
		{"v1.2.3", 1, 2, 3, false},
		{"0.1.0", 0, 1, 0, false},
		{"1.0.0-rc1", 1, 0, 0, false},
		{"1.0.0+build.5", 1, 0, 0, false},
		{"1.0", 1, 0, 0, false},
		{"abc", 0, 0, 0, true},
		{"1.a.0", 0, 0, 0, true},
	}
	for _, c := range cases {
		t.Run(c.in, func(t *testing.T) {
			got, err := ParseSemVer(c.in)
			if (err != nil) != c.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, c.wantErr)
			}
			if c.wantErr {
				return
			}
			if got.Major != c.wantMaj || got.Minor != c.wantMin || got.Patch != c.wantPat {
				t.Errorf("got %+v want %d.%d.%d", got, c.wantMaj, c.wantMin, c.wantPat)
			}
		})
	}
}

func TestSemVerCompare(t *testing.T) {
	a, _ := ParseSemVer("1.2.3")
	b, _ := ParseSemVer("1.2.4")
	if a.Compare(b) != -1 {
		t.Errorf("1.2.3 should be less than 1.2.4")
	}
	if b.Compare(a) != 1 {
		t.Errorf("1.2.4 should be greater than 1.2.3")
	}
	if a.Compare(a) != 0 {
		t.Errorf("equal versions should return 0")
	}
}

func TestCheckMinSDK(t *testing.T) {
	if err := CheckMinSDK("0.0.1"); err != nil {
		t.Errorf("SDK %s should satisfy 0.0.1: %v", Version, err)
	}
	if err := CheckMinSDK("999.0.0"); err == nil {
		t.Errorf("SDK %s should fail 999.0.0 check", Version)
	}
}

func TestCheckCLICompat(t *testing.T) {
	if err := CheckCLICompat("1.0.9", "1.0.0", ""); err != nil {
		t.Errorf("1.0.9 should satisfy >= 1.0.0: %v", err)
	}
	if err := CheckCLICompat("0.9.0", "1.0.0", ""); err == nil {
		t.Errorf("0.9.0 should fail >= 1.0.0")
	}
	if err := CheckCLICompat("2.0.0", "1.0.0", "1.9.9"); err == nil {
		t.Errorf("2.0.0 should fail <= 1.9.9")
	}
}

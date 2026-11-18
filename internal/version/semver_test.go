package version

import "testing"

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"2.0.0", "2.0.0", 0},
		{"v2.0.0", "2.0.0", 0},
		{"1.9.0", "2.0.0", -1},
		{"2.0.1", "2.0.0", 1},
		{"2.0.0-rc.1", "2.0.0", -1},
		{"2.0.0", "2.0.0-rc.1", 1},
		{"2.0.0-rc.1", "2.0.0-rc.2", -1},
		{"2.0.0-dev", "2.0.0-rc.1", -1},
		{"2.0.0-dev", "2.0.0", -1},
	}
	for _, tc := range cases {
		got, err := Compare(tc.a, tc.b)
		if err != nil {
			t.Fatalf("%s vs %s: %v", tc.a, tc.b, err)
		}
		if got != tc.want {
			t.Fatalf("%s vs %s: got %d want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestParseSemVerRejectsEmpty(t *testing.T) {
	if _, err := ParseSemVer(""); err == nil {
		t.Fatal("expected error")
	}
}

func TestPrerelease(t *testing.T) {
	v, err := ParseSemVer("2.0.0-rc.1")
	if err != nil || !v.Prerelease() {
		t.Fatalf("%+v %v", v, err)
	}
	v, err = ParseSemVer("2.0.0")
	if err != nil || v.Prerelease() {
		t.Fatalf("%+v %v", v, err)
	}
}

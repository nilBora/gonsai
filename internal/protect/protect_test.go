package protect

import "testing"

func TestStripRemotePrefix(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"origin/main", "main"},
		{"origin/develop", "develop"},
		{"main", "main"},
		{"", ""},
		{"  origin/master  ", "master"},
		{"upstream/main", "upstream/main"}, // only strips "origin/" prefix
	}
	for _, tc := range tests {
		got := stripRemotePrefix(tc.input)
		if got != tc.want {
			t.Errorf("stripRemotePrefix(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestFallbackDefaultsContainsMainAndMaster(t *testing.T) {
	found := make(map[string]bool)
	for _, n := range fallbackDefaults {
		found[n] = true
	}
	for _, required := range []string{"main", "master"} {
		if !found[required] {
			t.Errorf("fallbackDefaults missing %q", required)
		}
	}
}

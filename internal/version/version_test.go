package version

import "testing"

func TestNormalizeVersion(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain semver", in: "1.0.1", want: "v1.0.1"},
		{name: "already prefixed", in: "v1.0.1", want: "v1.0.1"},
		{name: "capital prefix", in: "V1.0.1", want: "v1.0.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeVersion(tt.in); got != tt.want {
				t.Fatalf("normalizeVersion(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestGetVersionUsesInjectedVersion(t *testing.T) {
	origVersion := Version
	t.Cleanup(func() {
		Version = origVersion
	})

	Version = "1.0.1"
	if got := GetVersion(); got != "v1.0.1" {
		t.Fatalf("GetVersion() = %q, want %q", got, "v1.0.1")
	}
}

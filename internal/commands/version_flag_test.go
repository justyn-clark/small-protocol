package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
)

// TestRootVersionField verifies the root command has Version set and the
// version template contains the protocol version string.
func TestRootVersionField(t *testing.T) {
	cmd := newRootCmd()

	if cmd.Version == "" {
		t.Fatal("root command Version field not set")
	}

	tpl := cmd.VersionTemplate()
	if !strings.Contains(tpl, small.ProtocolVersion) {
		t.Errorf("version template should contain protocol version %q, got: %q", small.ProtocolVersion, tpl)
	}
	if !strings.HasPrefix(tpl, "small ") {
		t.Errorf("version template should start with 'small ', got: %q", tpl)
	}
}

func TestRootVersionFlagOutput(t *testing.T) {
	var buf bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() with --version failed: %v", err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, "small ") {
		t.Errorf("--version output should start with 'small ', got: %q", out)
	}
	if !strings.Contains(out, small.ProtocolVersion) {
		t.Errorf("--version output should contain protocol version %q, got: %q", small.ProtocolVersion, out)
	}
}

func TestRootShortVersionFlagOutput(t *testing.T) {
	var buf bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"-v"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() with -v failed: %v", err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, "small ") {
		t.Errorf("-v output should start with 'small ', got: %q", out)
	}
	if !strings.Contains(out, small.ProtocolVersion) {
		t.Errorf("-v output should contain protocol version %q, got: %q", small.ProtocolVersion, out)
	}
}

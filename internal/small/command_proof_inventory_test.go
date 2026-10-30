package small

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommandProofInventorySeparatesAvailabilityFromPolicy(t *testing.T) {
	command := `const token = process.env.EXAMPLE_TOKEN; # ` + strings.Repeat("x", 240)
	base, marked := proofFixture(t, command, true)
	before, _ := json.Marshal(marked)
	report := NewCommandProofInventory()
	report.Add(base, "capture", marked)
	if report.Coverage != "complete" || report.Available != 1 || !report.Records[0].Required {
		t.Fatalf("available marked command: %#v", report)
	}
	missing := t.TempDir()
	report.Add(missing, "missing-marked", marked)
	if report.Coverage != "incomplete" || report.Missing != 1 || report.RequiredUnavailable != 1 {
		t.Fatalf("missing marked command: %#v", report)
	}
	if violations := CheckCommandRecord(missing, "capture", marked); !proofMessagesContain(violations, "proof unavailable") {
		t.Fatal("inventory changed strict missing-proof policy")
	}
	_, legacy := proofFixture(t, command, false)
	report.Add(missing, "missing-legacy", legacy)
	if report.Missing != 2 || report.RequiredUnavailable != 1 || report.Records[2].Required {
		t.Fatalf("legacy gap incorrectly treated as proven: %#v", report)
	}
	if violations := CheckCommandRecord(missing, "capture", legacy); len(violations) != 0 {
		t.Fatalf("inventory tightened historical proof policy: %v", violations)
	}
	after, _ := json.Marshal(marked)
	if string(before) != string(after) {
		t.Fatal("inventory changed receipt")
	}
}

func TestCommandProofInventoryPortableTamperAndBinding(t *testing.T) {
	base, record := proofFixture(t, boundaryCommand(), true)
	ref := record["command_ref"].(string)
	data, err := os.ReadFile(filepath.Join(base, ref))
	if err != nil {
		t.Fatal(err)
	}
	fresh := t.TempDir()
	portable := filepath.Join(fresh, CommandProofDir+strings.TrimPrefix(ref, CacheDirName))
	if err := os.MkdirAll(filepath.Dir(portable), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(portable, data, 0o600); err != nil {
		t.Fatal(err)
	}
	report := NewCommandProofInventory()
	report.Add(fresh, "portable", record)
	if report.Available != 1 {
		t.Fatalf("portable proof: %#v", report)
	}
	report.AddEvent(fresh, "event", record, strings.Repeat("b", 64))
	if report.Invalid != 1 || report.RequiredUnavailable != 1 {
		t.Fatal("event source digest mismatch accepted")
	}
	for _, field := range []string{"timestamp", "command_summary", "command_summary_version"} {
		forged := make(map[string]any, len(record))
		for key, value := range record {
			forged[key] = value
		}
		forged[field] = "invalid"
		report.Add(fresh, field, forged)
		if report.Records[len(report.Records)-1].Availability != "invalid" {
			t.Fatalf("forged %s considered available", field)
		}
	}
	if err := os.WriteFile(portable, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	report.Add(fresh, "tampered", record)
	if report.Records[len(report.Records)-1].Availability != "invalid" {
		t.Fatal("tampered bytes considered available")
	}
}

func TestCommandProofInventoryDoesNotRestoreOrDisclose(t *testing.T) {
	base, record := proofFixture(t, "printf fixture-value", false)
	ref := record["command_ref"].(string)
	if err := os.Remove(filepath.Join(base, ref)); err != nil {
		t.Fatal(err)
	}
	report := NewCommandProofInventory()
	report.Add(base, "short", record)
	report.Add(base, "unreferenced", map[string]any{"command_summary": "token=fixture-value"})
	if report.DisplayDigestMatches != 1 || report.Unreferenced != 1 || report.Available != 0 {
		t.Fatalf("display identity is not proof availability: %#v", report)
	}
	data, err := json.Marshal(report)
	if err != nil || strings.Contains(string(data), "fixture-value") || strings.Contains(string(data), ref) {
		t.Fatal("inventory disclosed command content or unreviewed metadata")
	}
	if _, err := os.Stat(filepath.Join(base, ref)); !os.IsNotExist(err) {
		t.Fatal("inventory restored a command")
	}
	if _, err := os.Stat(filepath.Join(base, CommandProofDir)); !os.IsNotExist(err) {
		t.Fatal("inventory exported proof")
	}
}

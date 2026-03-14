package fixers

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/small"
)

func TestFixRuntimeLayoutMigratesLegacyTrees(t *testing.T) {
	baseDir := t.TempDir()
	legacyArchive := filepath.Join(baseDir, small.SmallDir, "archive", "run-1")
	legacyRuns := filepath.Join(baseDir, small.SmallDir, "runs")
	if err := os.MkdirAll(legacyArchive, 0o755); err != nil {
		t.Fatalf("failed to create legacy archive: %v", err)
	}
	if err := os.MkdirAll(legacyRuns, 0o755); err != nil {
		t.Fatalf("failed to create legacy runs: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyArchive, "archive.small.yml"), []byte("archive"), 0o644); err != nil {
		t.Fatalf("failed to seed archive: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyRuns, small.RunIndexFileName), []byte("index"), 0o644); err != nil {
		t.Fatalf("failed to seed run index: %v", err)
	}

	result, err := FixRuntimeLayout(baseDir)
	if err != nil {
		t.Fatalf("FixRuntimeLayout error: %v", err)
	}
	if len(result.Migrations) != 2 {
		t.Fatalf("expected 2 migrations, got %d", len(result.Migrations))
	}
	if _, err := os.Stat(filepath.Join(baseDir, small.ArchiveStoreDirName, "run-1", "archive.small.yml")); err != nil {
		t.Fatalf("expected migrated archive: %v", err)
	}
	if _, err := os.Stat(filepath.Join(baseDir, small.RunStoreDirName, small.RunIndexFileName)); err != nil {
		t.Fatalf("expected migrated run index: %v", err)
	}
	if _, err := os.Stat(filepath.Join(baseDir, small.SmallDir, "archive")); !os.IsNotExist(err) {
		t.Fatalf("expected legacy archive dir removed, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(baseDir, small.SmallDir, "runs")); !os.IsNotExist(err) {
		t.Fatalf("expected legacy runs dir removed, got err=%v", err)
	}

	result, err = FixRuntimeLayout(baseDir)
	if err != nil {
		t.Fatalf("second FixRuntimeLayout error: %v", err)
	}
	if len(result.Migrations) != 0 || len(result.Deduped) != 0 {
		t.Fatalf("expected idempotent second pass, got %+v", result)
	}
}

func TestFixRuntimeLayoutDedupesMatchingFiles(t *testing.T) {
	baseDir := t.TempDir()
	legacyRuns := filepath.Join(baseDir, small.SmallDir, "runs")
	canonicalRuns := filepath.Join(baseDir, small.RunStoreDirName)
	if err := os.MkdirAll(legacyRuns, 0o755); err != nil {
		t.Fatalf("failed to create legacy runs: %v", err)
	}
	if err := os.MkdirAll(canonicalRuns, 0o755); err != nil {
		t.Fatalf("failed to create canonical runs: %v", err)
	}
	data := []byte("same-index")
	if err := os.WriteFile(filepath.Join(legacyRuns, small.RunIndexFileName), data, 0o644); err != nil {
		t.Fatalf("failed to seed legacy run index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(canonicalRuns, small.RunIndexFileName), data, 0o644); err != nil {
		t.Fatalf("failed to seed canonical run index: %v", err)
	}

	result, err := FixRuntimeLayout(baseDir)
	if err != nil {
		t.Fatalf("FixRuntimeLayout error: %v", err)
	}
	if len(result.Migrations) != 0 {
		t.Fatalf("expected no migrations, got %+v", result.Migrations)
	}
	if len(result.Deduped) != 1 || !strings.Contains(result.Deduped[0], ".small/runs/index.small.yml") {
		t.Fatalf("unexpected deduped paths: %+v", result.Deduped)
	}
	if _, err := os.Stat(filepath.Join(baseDir, small.SmallDir, "runs")); !os.IsNotExist(err) {
		t.Fatalf("expected legacy runs dir removed after dedupe, got err=%v", err)
	}
}

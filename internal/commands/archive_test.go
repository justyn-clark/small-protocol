package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justyn-clark/small-protocol/internal/workspace"
	"gopkg.in/yaml.v3"
)

func TestArchiveCommand(t *testing.T) {
	t.Run("archive succeeds with valid workspace", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "small-archive-test")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create a valid workspace using selftest init
		if err := runSelftestInit(tmpDir); err != nil {
			t.Fatalf("failed to init workspace: %v", err)
		}

		// Generate handoff with replayId
		if err := runSelftestHandoff(tmpDir); err != nil {
			t.Fatalf("failed to generate handoff: %v", err)
		}

		// Run archive
		defaultInclude := []string{
			"intent.small.yml",
			"constraints.small.yml",
			"plan.small.yml",
			"progress.small.yml",
			"handoff.small.yml",
			"workspace.small.yml",
		}
		if err := runArchive(tmpDir, "", defaultInclude); err != nil {
			t.Fatalf("archive failed: %v", err)
		}

		// Verify archive directory exists
		archiveDir := filepath.Join(tmpDir, ".small", "archive")
		entries, err := os.ReadDir(archiveDir)
		if err != nil {
			t.Fatalf("failed to read archive dir: %v", err)
		}
		if len(entries) != 1 {
			t.Fatalf("expected 1 archive directory, got %d", len(entries))
		}

		// Verify manifest exists
		manifestPath := filepath.Join(archiveDir, entries[0].Name(), "archive.small.yml")
		if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
			t.Error("expected archive.small.yml manifest to exist")
		}

		// Verify manifest content
		manifestData, err := os.ReadFile(manifestPath)
		if err != nil {
			t.Fatalf("failed to read manifest: %v", err)
		}

		var manifest archiveManifest
		if err := yaml.Unmarshal(manifestData, &manifest); err != nil {
			t.Fatalf("failed to parse manifest: %v", err)
		}

		if manifest.SmallVersion != "1.0.0" {
			t.Errorf("expected small_version 1.0.0, got %s", manifest.SmallVersion)
		}
		if manifest.ReplayId == "" {
			t.Error("expected replayId to be set")
		}
		if len(manifest.Files) == 0 {
			t.Error("expected at least one file in manifest")
		}

		// Verify SHA256 hashes are valid format
		for _, f := range manifest.Files {
			if len(f.SHA256) != 64 {
				t.Errorf("expected SHA256 hash for %s to be 64 chars, got %d", f.Name, len(f.SHA256))
			}
		}
	})

	t.Run("archive fails without replayId", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "small-archive-no-replay")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create workspace but don't generate handoff with replayId
		if err := runSelftestInit(tmpDir); err != nil {
			t.Fatalf("failed to init workspace: %v", err)
		}
		handoffPath := filepath.Join(tmpDir, ".small", "handoff.small.yml")
		handoff := `small_version: "1.0.0"
owner: "agent"
summary: "Missing replayId"
resume:
  current_task_id: ""
  next_steps: []
links: []
`
		if err := os.WriteFile(handoffPath, []byte(handoff), 0644); err != nil {
			t.Fatalf("failed to overwrite handoff: %v", err)
		}

		// Archive should fail
		defaultInclude := []string{"intent.small.yml"}
		err = runArchive(tmpDir, "", defaultInclude)
		if err == nil {
			t.Error("expected archive to fail without replayId")
		}
		if !strings.Contains(err.Error(), "replayId") {
			t.Errorf("expected error message to mention replayId, got: %v", err)
		}
	})

	t.Run("archive to custom output directory", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "small-archive-custom-out")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		// Create valid workspace
		if err := runSelftestInit(tmpDir); err != nil {
			t.Fatalf("failed to init workspace: %v", err)
		}
		if err := runSelftestHandoff(tmpDir); err != nil {
			t.Fatalf("failed to generate handoff: %v", err)
		}

		// Custom output directory
		customOut := filepath.Join(tmpDir, "my-archive")
		defaultInclude := []string{"intent.small.yml", "handoff.small.yml"}
		if err := runArchive(tmpDir, customOut, defaultInclude); err != nil {
			t.Fatalf("archive failed: %v", err)
		}

		// Verify custom output directory was used
		if _, err := os.Stat(filepath.Join(customOut, "archive.small.yml")); os.IsNotExist(err) {
			t.Error("expected archive to be created in custom output directory")
		}
	})
}

func TestComputeFileSHA256(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "small-sha256-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a test file with known content
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("hello world\n"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	hash, err := computeFileSHA256(testFile)
	if err != nil {
		t.Fatalf("computeFileSHA256 failed: %v", err)
	}

	// SHA256 of "hello world\n" is known
	expected := "a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447"
	if hash != expected {
		t.Errorf("expected hash %s, got %s", expected, hash)
	}
}

func TestCopyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "small-copy-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcFile := filepath.Join(tmpDir, "src.txt")
	content := []byte("test content for copy")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatalf("failed to write source file: %v", err)
	}

	// Copy file
	dstFile := filepath.Join(tmpDir, "dst.txt")
	if err := copyFile(srcFile, dstFile); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify content
	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}

	if string(dstContent) != string(content) {
		t.Errorf("expected %q, got %q", string(content), string(dstContent))
	}
}

func TestArchiveWithWorkspaceScope(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "small-archive-scope")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create workspace with examples kind
	if err := runSelftestInit(tmpDir); err != nil {
		t.Fatalf("failed to init workspace: %v", err)
	}
	if err := workspace.Save(tmpDir, workspace.KindExamples); err != nil {
		t.Fatalf("failed to save workspace: %v", err)
	}
	if err := runSelftestHandoff(tmpDir); err != nil {
		t.Fatalf("failed to generate handoff: %v", err)
	}

	// Archive should still work regardless of workspace kind
	defaultInclude := []string{"intent.small.yml", "handoff.small.yml"}
	if err := runArchive(tmpDir, "", defaultInclude); err != nil {
		t.Fatalf("archive should work for any workspace kind: %v", err)
	}
}

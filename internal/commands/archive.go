package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/runstore"
	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// archiveManifest represents the archive.small.yml manifest file
type archiveManifest struct {
	SmallVersion string        `yaml:"small_version"`
	ArchivedAt   string        `yaml:"archived_at"`
	SourceDir    string        `yaml:"source_dir"`
	ReplayId     string        `yaml:"replayId"`
	Files        []archiveFile `yaml:"files"`
}

// archiveFile represents a file in the archive manifest
type archiveFile struct {
	Name   string `yaml:"name"`
	SHA256 string `yaml:"sha256"`
}

var defaultArchiveInclude = []string{
	"intent.small.yml",
	"constraints.small.yml",
	"plan.small.yml",
	"progress.small.yml",
	"handoff.small.yml",
	"workspace.small.yml",
}

func archiveCmd() *cobra.Command {
	var (
		dir     string
		out     string
		include []string
	)

	cmd := &cobra.Command{
		Use:   "archive",
		Short: "Archive the current run state for lineage retention",
		Long: `Archives the current .small/ workspace to preserve run lineage without
committing the runtime directory.

The archive includes:
  - All canonical SMALL artifacts
  - A manifest with SHA256 hashes for integrity verification
  - The replayId for session identification

Archives are stored in .small-archive/<replayId>/ by default.
Archives are local by default (ignored in .gitignore), but you may
commit them in product repos if you want persistent lineage.

Requirements:
  - handoff.small.yml must have a valid replayId (run 'small handoff' first)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir == "" {
				dir = baseDir
			}
			artifactsDir := resolveArtifactsDir(dir)

			if len(include) == 0 {
				include = defaultArchiveInclude
			}

			return runArchive(artifactsDir, out, include)
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&out, "out", "", "Output directory (default: .small-archive/<replayId>/)")
	cmd.Flags().StringSliceVar(&include, "include", nil, "Files to include (default: all canonical artifacts)")

	return cmd
}

func runArchive(artifactsDir, outDir string, include []string) error {
	smallDir := filepath.Join(artifactsDir, ".small")

	// Check if .small/ directory exists
	if _, err := os.Stat(smallDir); os.IsNotExist(err) {
		return fmt.Errorf(".small/ directory does not exist. Run 'small init' first")
	}
	if sessionv2.IsWorkspace(artifactsDir) {
		return runArchiveV2(artifactsDir, outDir, include)
	}

	// Load handoff to get replayId
	handoffPath := filepath.Join(smallDir, "handoff.small.yml")
	handoffData, err := os.ReadFile(handoffPath)
	if err != nil {
		return fmt.Errorf("failed to read handoff.small.yml: %w (run 'small handoff' first)", err)
	}

	var handoff map[string]any
	if err := yaml.Unmarshal(handoffData, &handoff); err != nil {
		return fmt.Errorf("failed to parse handoff.small.yml: %w", err)
	}

	// Extract replayId
	replayIdMap, ok := handoff["replayId"].(map[string]any)
	if !ok {
		return fmt.Errorf("handoff.small.yml missing replayId. Run 'small handoff' first")
	}

	replayId, ok := replayIdMap["value"].(string)
	if !ok || replayId == "" {
		return fmt.Errorf("handoff.small.yml has invalid replayId. Run 'small handoff' first")
	}

	// Determine output directory
	if outDir == "" {
		outDir = filepath.Join(small.ArchiveStoreDir(artifactsDir), replayId)
	}

	// Create output directory
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Copy files and compute hashes
	var files []archiveFile
	for _, filename := range include {
		srcPath := filepath.Join(smallDir, filename)

		// Skip files that don't exist (e.g., constraints might be optional in some cases)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			continue
		}

		// Compute SHA256 hash
		hash, err := computeFileSHA256(srcPath)
		if err != nil {
			return fmt.Errorf("failed to compute hash for %s: %w", filename, err)
		}

		// Copy file
		dstPath := filepath.Join(outDir, filename)
		if err := copyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", filename, err)
		}

		files = append(files, archiveFile{
			Name:   filename,
			SHA256: hash,
		})
	}

	archivedAt := time.Now().UTC().Format(time.RFC3339Nano)

	// Create manifest
	manifest := archiveManifest{
		SmallVersion: small.ProtocolVersion,
		ArchivedAt:   archivedAt,
		SourceDir:    artifactsDir,
		ReplayId:     replayId,
		Files:        files,
	}

	manifestData, err := small.MarshalYAMLWithQuotedVersion(&manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	manifestPath := filepath.Join(outDir, "archive.small.yml")
	if err := os.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	entry := small.RunIndexEntry{
		ReplayID:  replayId,
		Timestamp: archivedAt,
		GitSHA:    resolveGitSHA(artifactsDir),
		Summary:   stringVal(handoff["summary"]),
		Reason:    "archive",
	}
	if err := small.AppendRunIndexEntry(artifactsDir, entry); err != nil {
		return fmt.Errorf("failed to append run index: %w", err)
	}

	fmt.Printf("Archived %d files to %s\n", len(files), outDir)
	fmt.Printf("ReplayId: %s\n", replayId[:16]+"...")
	return nil
}

func runArchiveV2(artifactsDir, outDir string, include []string) error {
	store, err := sessionv2.Load(artifactsDir)
	if err != nil {
		return err
	}
	state, err := sessionv2.Strict(store)
	if err != nil {
		return fmt.Errorf("v2 archive requires reconciled strict state: %w", err)
	}
	if len(include) > 0 && !sameArchiveIncludes(include, defaultArchiveInclude) {
		return fmt.Errorf("v2 archive always preserves the complete authoritative .small tree; --include is not supported")
	}
	if outDir == "" {
		outDir = filepath.Join(small.ArchiveStoreDir(artifactsDir), state.Frontier)
	}
	if _, statErr := os.Stat(outDir); statErr == nil {
		return fmt.Errorf("archive already exists: %s", outDir)
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	stateRoot := filepath.Join(outDir, "state", small.SmallDir)
	if err := copyArchiveTree(filepath.Join(artifactsDir, small.SmallDir), stateRoot); err != nil {
		return err
	}
	_, digests, err := runstore.ComputeArtifactDigests(artifactsDir)
	if err != nil {
		return err
	}
	paths := make([]string, 0, len(digests))
	for path := range digests {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	files := make([]archiveFile, 0, len(paths))
	for _, path := range paths {
		files = append(files, archiveFile{Name: path, SHA256: digests[path]})
	}
	archivedAt := time.Now().UTC().Format(time.RFC3339Nano)
	manifest := archiveManifest{SmallVersion: sessionv2.ProfileVersion, ArchivedAt: archivedAt, SourceDir: artifactsDir, ReplayId: state.Frontier, Files: files}
	manifestData, err := small.MarshalYAMLWithQuotedVersion(&manifest)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(outDir, "archive.small.yml"), manifestData, 0o644); err != nil {
		return err
	}
	summary := ""
	if len(state.Handoffs) > 0 {
		summary, _ = state.Handoffs[len(state.Handoffs)-1].Payload["summary"].(string)
	}
	if err := small.AppendRunIndexEntry(artifactsDir, small.RunIndexEntry{ReplayID: state.Frontier, Timestamp: archivedAt, GitSHA: resolveGitSHA(artifactsDir), Summary: summary, Reason: "archive"}); err != nil {
		return err
	}
	fmt.Printf("Archived %d files to %s\n", len(files), outDir)
	fmt.Printf("Frontier: %s\n", state.Frontier)
	return nil
}

func sameArchiveIncludes(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	set := map[string]bool{}
	for _, value := range a {
		set[value] = true
	}
	for _, value := range b {
		if !set[value] {
			return false
		}
	}
	return true
}

func copyArchiveTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing archive symlink %s", path)
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("refusing archive non-regular file %s", path)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return copyFile(path, target)
	})
}

// computeFileSHA256 computes the SHA256 hash of a file
func computeFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// copyFile copies a file from src to dst
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	return dstFile.Sync()
}

func resolveGitSHA(baseDir string) string {
	if baseDir == "" {
		return ""
	}
	if _, err := exec.LookPath("git"); err != nil {
		return ""
	}

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = baseDir
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

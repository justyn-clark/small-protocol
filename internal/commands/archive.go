package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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

func archiveCmd() *cobra.Command {
	var (
		dir     string
		out     string
		include []string
	)

	// Default files to archive
	defaultInclude := []string{
		"intent.small.yml",
		"constraints.small.yml",
		"plan.small.yml",
		"progress.small.yml",
		"handoff.small.yml",
		"workspace.small.yml",
	}

	cmd := &cobra.Command{
		Use:   "archive",
		Short: "Archive the current run state for lineage retention",
		Long: `Archives the current .small/ workspace to preserve run lineage without
committing the runtime directory.

The archive includes:
  - All canonical SMALL artifacts
  - A manifest with SHA256 hashes for integrity verification
  - The replayId for session identification

Archives are stored in .small/archive/<replayId>/ by default.
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
				include = defaultInclude
			}

			return runArchive(artifactsDir, out, include)
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".", "Directory containing .small/ artifacts")
	cmd.Flags().StringVar(&out, "out", "", "Output directory (default: .small/archive/<replayId>/)")
	cmd.Flags().StringSliceVar(&include, "include", nil, "Files to include (default: all canonical artifacts)")

	return cmd
}

func runArchive(artifactsDir, outDir string, include []string) error {
	smallDir := filepath.Join(artifactsDir, ".small")

	// Check if .small/ directory exists
	if _, err := os.Stat(smallDir); os.IsNotExist(err) {
		return fmt.Errorf(".small/ directory does not exist. Run 'small init' first")
	}

	// Load handoff to get replayId
	handoffPath := filepath.Join(smallDir, "handoff.small.yml")
	handoffData, err := os.ReadFile(handoffPath)
	if err != nil {
		return fmt.Errorf("failed to read handoff.small.yml: %w (run 'small handoff' first)", err)
	}

	var handoff map[string]interface{}
	if err := yaml.Unmarshal(handoffData, &handoff); err != nil {
		return fmt.Errorf("failed to parse handoff.small.yml: %w", err)
	}

	// Extract replayId
	replayIdMap, ok := handoff["replayId"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("handoff.small.yml missing replayId. Run 'small handoff' first")
	}

	replayId, ok := replayIdMap["value"].(string)
	if !ok || replayId == "" {
		return fmt.Errorf("handoff.small.yml has invalid replayId. Run 'small handoff' first")
	}

	// Determine output directory
	if outDir == "" {
		outDir = filepath.Join(smallDir, "archive", replayId)
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

package runstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/version"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"gopkg.in/yaml.v3"
)

const (
	DefaultStoreDirName = ".small-runs"
	MetaFileName        = "meta.json"
)

var (
	RequiredArtifacts = []string{
		"intent.small.yml",
		"plan.small.yml",
		"progress.small.yml",
		"handoff.small.yml",
	}
	OptionalArtifacts = []string{
		"constraints.small.yml",
	}
)

type Meta struct {
	ReplayID      string `json:"replayId"`
	CreatedAt     string `json:"created_at"`
	GitSHA        string `json:"git_sha"`
	GitDirty      bool   `json:"git_dirty"`
	Branch        string `json:"branch"`
	CLIVersion    string `json:"cli_version"`
	WorkspaceKind string `json:"workspace_kind"`
	SourceDir     string `json:"source_dir"`
}

type Snapshot struct {
	ReplayID         string
	Dir              string
	Meta             Meta
	Artifacts        []string
	HandoffSummary   string
	HandoffNextSteps []string
	CreatedAt        time.Time
}

type HandoffInfo struct {
	Summary   string
	NextSteps []string
	ReplayID  string
}

func ResolveStoreDir(baseDir, storeDir string) string {
	if storeDir == "" {
		return filepath.Join(baseDir, DefaultStoreDirName)
	}
	if filepath.IsAbs(storeDir) {
		return storeDir
	}
	return filepath.Join(baseDir, storeDir)
}

func WriteSnapshot(baseDir, storeDir string, force bool) (*Snapshot, error) {
	if baseDir == "" {
		return nil, fmt.Errorf("base directory is required")
	}
	storeDir = ResolveStoreDir(baseDir, storeDir)
	smallDir := filepath.Join(baseDir, small.SmallDir)

	if _, err := os.Stat(smallDir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(".small/ directory does not exist. Run 'small init' first")
		}
		return nil, fmt.Errorf("failed to stat .small directory: %w", err)
	}

	for _, filename := range RequiredArtifacts {
		path := filepath.Join(smallDir, filename)
		if _, err := os.Stat(path); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("%s not found in .small/ (run 'small init' first)", filename)
			}
			return nil, fmt.Errorf("failed to read %s: %w", filename, err)
		}
	}

	handoffInfo, err := readHandoff(filepath.Join(smallDir, "handoff.small.yml"))
	if err != nil {
		return nil, err
	}
	if handoffInfo.ReplayID == "" {
		return nil, fmt.Errorf("handoff.small.yml missing replayId, run: small handoff --summary \"<summary>\"")
	}

	workspaceInfo, err := workspace.Load(baseDir)
	if err != nil {
		return nil, err
	}

	snapshotDir := filepath.Join(storeDir, handoffInfo.ReplayID)
	if _, err := os.Stat(snapshotDir); err == nil {
		if !force {
			return nil, fmt.Errorf("run snapshot exists, pass --force to overwrite")
		}
		if err := os.RemoveAll(snapshotDir); err != nil {
			return nil, fmt.Errorf("failed to remove existing snapshot: %w", err)
		}
	}

	if err := os.MkdirAll(snapshotDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create snapshot directory: %w", err)
	}

	artifacts := []string{}
	for _, filename := range RequiredArtifacts {
		src := filepath.Join(smallDir, filename)
		dst := filepath.Join(snapshotDir, filename)
		if err := copyFile(src, dst); err != nil {
			return nil, fmt.Errorf("failed to copy %s: %w", filename, err)
		}
		artifacts = append(artifacts, dst)
	}
	for _, filename := range OptionalArtifacts {
		src := filepath.Join(smallDir, filename)
		if _, err := os.Stat(src); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("failed to read %s: %w", filename, err)
		}
		dst := filepath.Join(snapshotDir, filename)
		if err := copyFile(src, dst); err != nil {
			return nil, fmt.Errorf("failed to copy %s: %w", filename, err)
		}
		artifacts = append(artifacts, dst)
	}

	gitSHA, gitDirty, branch := resolveGitInfo(baseDir)
	meta := Meta{
		ReplayID:      handoffInfo.ReplayID,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339Nano),
		GitSHA:        gitSHA,
		GitDirty:      gitDirty,
		Branch:        branch,
		CLIVersion:    version.GetVersion(),
		WorkspaceKind: string(workspaceInfo.Kind),
		SourceDir:     baseDir,
	}

	if err := WriteMeta(snapshotDir, meta); err != nil {
		return nil, err
	}

	entry := small.RunIndexEntry{
		ReplayID:  handoffInfo.ReplayID,
		Timestamp: meta.CreatedAt,
		GitSHA:    gitSHA,
		Summary:   handoffInfo.Summary,
		Reason:    "snapshot",
	}
	if err := small.AppendRunIndexEntry(baseDir, entry); err != nil {
		return nil, err
	}

	createdAt, _ := time.Parse(time.RFC3339Nano, meta.CreatedAt)
	return &Snapshot{
		ReplayID:         handoffInfo.ReplayID,
		Dir:              snapshotDir,
		Meta:             meta,
		Artifacts:        artifacts,
		HandoffSummary:   handoffInfo.Summary,
		HandoffNextSteps: handoffInfo.NextSteps,
		CreatedAt:        createdAt,
	}, nil
}

func ListSnapshots(storeDir string) ([]Snapshot, error) {
	entries, err := os.ReadDir(storeDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("run store not found, no snapshots yet, run: small run snapshot")
		}
		return nil, fmt.Errorf("failed to read run store: %w", err)
	}

	snapshots := make([]Snapshot, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		replayID := entry.Name()
		snapshotDir := filepath.Join(storeDir, replayID)
		meta, metaErr := ReadMeta(snapshotDir)
		if metaErr != nil {
			if errors.Is(metaErr, os.ErrNotExist) {
				meta = Meta{ReplayID: replayID}
			} else {
				return nil, metaErr
			}
		}
		if meta.ReplayID == "" {
			meta.ReplayID = replayID
		}

		createdAt, err := time.Parse(time.RFC3339Nano, meta.CreatedAt)
		if err != nil {
			createdAt = time.Time{}
		}

		if createdAt.IsZero() {
			info, infoErr := entry.Info()
			if infoErr != nil {
				return nil, fmt.Errorf("failed to stat snapshot %s: %w", replayID, infoErr)
			}
			createdAt = info.ModTime().UTC()
			if meta.CreatedAt == "" {
				meta.CreatedAt = createdAt.Format(time.RFC3339Nano)
			}
		}

		handoffInfo, err := readHandoff(filepath.Join(snapshotDir, "handoff.small.yml"))
		if err != nil {
			return nil, err
		}

		snapshots = append(snapshots, Snapshot{
			ReplayID:         replayID,
			Dir:              snapshotDir,
			Meta:             meta,
			HandoffSummary:   handoffInfo.Summary,
			HandoffNextSteps: handoffInfo.NextSteps,
			CreatedAt:        createdAt,
		})
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].CreatedAt.After(snapshots[j].CreatedAt)
	})

	return snapshots, nil
}

func LoadSnapshot(storeDir, replayID string) (*Snapshot, error) {
	snapshotDir := filepath.Join(storeDir, replayID)
	if _, err := os.Stat(snapshotDir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("run snapshot not found: %s", replayID)
		}
		return nil, fmt.Errorf("failed to read snapshot: %w", err)
	}

	meta, err := ReadMeta(snapshotDir)
	if err != nil {
		return nil, err
	}
	if meta.ReplayID == "" {
		meta.ReplayID = replayID
	}

	handoffInfo, err := readHandoff(filepath.Join(snapshotDir, "handoff.small.yml"))
	if err != nil {
		return nil, err
	}

	artifacts := []string{filepath.Join(snapshotDir, MetaFileName)}
	for _, filename := range append(RequiredArtifacts, OptionalArtifacts...) {
		path := filepath.Join(snapshotDir, filename)
		if _, err := os.Stat(path); err == nil {
			artifacts = append(artifacts, path)
		}
	}

	createdAt, _ := time.Parse(time.RFC3339Nano, meta.CreatedAt)
	return &Snapshot{
		ReplayID:         replayID,
		Dir:              snapshotDir,
		Meta:             meta,
		Artifacts:        artifacts,
		HandoffSummary:   handoffInfo.Summary,
		HandoffNextSteps: handoffInfo.NextSteps,
		CreatedAt:        createdAt,
	}, nil
}

func CheckoutSnapshot(baseDir, storeDir, replayID string, force bool) error {
	snapshot, err := LoadSnapshot(storeDir, replayID)
	if err != nil {
		return err
	}

	smallDir := filepath.Join(baseDir, small.SmallDir)
	if _, err := os.Stat(smallDir); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf(".small/ directory does not exist. Run 'small init' first")
		}
		return fmt.Errorf("failed to read .small directory: %w", err)
	}

	if !force {
		for _, filename := range snapshotFiles(snapshot.Dir) {
			src := filepath.Join(snapshot.Dir, filename)
			dst := filepath.Join(smallDir, filename)
			same, err := filesEqual(src, dst)
			if err != nil {
				return err
			}
			if !same {
				return fmt.Errorf("workspace has uncommitted .small changes, pass --force to overwrite")
			}
		}
	}

	for _, filename := range snapshotFiles(snapshot.Dir) {
		src := filepath.Join(snapshot.Dir, filename)
		dst := filepath.Join(smallDir, filename)
		if err := copyFile(src, dst); err != nil {
			return fmt.Errorf("failed to restore %s: %w", filename, err)
		}
	}

	return nil
}

func ReadMeta(snapshotDir string) (Meta, error) {
	path := filepath.Join(snapshotDir, MetaFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return Meta{}, err
	}

	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return Meta{}, fmt.Errorf("failed to parse meta.json: %w", err)
	}
	return meta, nil
}

func WriteMeta(snapshotDir string, meta Meta) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal meta.json: %w", err)
	}
	path := filepath.Join(snapshotDir, MetaFileName)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write meta.json: %w", err)
	}
	return nil
}

func snapshotFiles(snapshotDir string) []string {
	files := []string{}
	for _, filename := range append(RequiredArtifacts, OptionalArtifacts...) {
		path := filepath.Join(snapshotDir, filename)
		if _, err := os.Stat(path); err == nil {
			files = append(files, filename)
		}
	}
	return files
}

func filesEqual(pathA, pathB string) (bool, error) {
	dataA, err := os.ReadFile(pathA)
	if err != nil {
		return false, fmt.Errorf("failed to read %s: %w", pathA, err)
	}
	dataB, err := os.ReadFile(pathB)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("failed to read %s: %w", pathB, err)
	}
	return string(dataA) == string(dataB), nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

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

func readHandoff(path string) (HandoffInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return HandoffInfo{}, fmt.Errorf("failed to read handoff.small.yml: %w", err)
	}
	var payload struct {
		Summary string `yaml:"summary"`
		Resume  struct {
			NextSteps []string `yaml:"next_steps"`
		} `yaml:"resume"`
		ReplayID struct {
			Value string `yaml:"value"`
		} `yaml:"replayId"`
	}
	if err := yaml.Unmarshal(data, &payload); err != nil {
		return HandoffInfo{}, fmt.Errorf("failed to parse handoff.small.yml: %w", err)
	}
	return HandoffInfo{
		Summary:   payload.Summary,
		NextSteps: payload.Resume.NextSteps,
		ReplayID:  strings.TrimSpace(payload.ReplayID.Value),
	}, nil
}

func resolveGitInfo(baseDir string) (string, bool, string) {
	if baseDir == "" {
		return "", false, ""
	}
	if _, err := exec.LookPath("git"); err != nil {
		return "", false, ""
	}

	_, err := runGit(baseDir, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return "", false, ""
	}

	sha, err := runGit(baseDir, "rev-parse", "HEAD")
	if err != nil {
		sha = ""
	}
	branch, err := runGit(baseDir, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil || branch == "HEAD" {
		branch = ""
	}
	dirty := false
	status, err := runGit(baseDir, "status", "--porcelain")
	if err == nil {
		dirty = strings.TrimSpace(status) != ""
	}

	return sha, dirty, branch
}

func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

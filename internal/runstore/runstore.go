package runstore

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/justyn-clark/small-protocol/internal/small"
	"github.com/justyn-clark/small-protocol/internal/version"
	"github.com/justyn-clark/small-protocol/internal/workspace"
	"gopkg.in/yaml.v3"
)

const (
	DefaultStoreDirName = small.RunStoreDirName
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
	ReplayID        string            `json:"replayId"`
	ArtifactDigest  string            `json:"artifact_digest,omitempty"`
	ArtifactDigests map[string]string `json:"artifact_digests,omitempty"`
	CreatedAt       string            `json:"created_at"`
	GitSHA          string            `json:"git_sha"`
	GitDirty        bool              `json:"git_dirty"`
	Branch          string            `json:"branch"`
	CLIVersion      string            `json:"cli_version"`
	WorkspaceKind   string            `json:"workspace_kind"`
	SourceDir       string            `json:"source_dir"`
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
		return small.RunStoreDir(baseDir)
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
	if sessionv2.IsWorkspace(baseDir) {
		return writeV2Snapshot(baseDir, storeDir, force)
	}

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

	artifactDigest, artifactDigests, err := ComputeArtifactDigests(baseDir)
	if err != nil {
		return nil, err
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
		ReplayID:        handoffInfo.ReplayID,
		ArtifactDigest:  artifactDigest,
		ArtifactDigests: artifactDigests,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339Nano),
		GitSHA:          gitSHA,
		GitDirty:        gitDirty,
		Branch:          branch,
		CLIVersion:      version.GetVersion(),
		WorkspaceKind:   string(workspaceInfo.Kind),
		SourceDir:       baseDir,
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

// ComputeArtifactDigests hashes the canonical artifacts inside a live
// workspace's .small/ directory. It returns an aggregate digest and a
// per-artifact map keyed by filename.
func ComputeArtifactDigests(baseDir string) (string, map[string]string, error) {
	if baseDir == "" {
		return "", nil, fmt.Errorf("base directory is required")
	}
	if sessionv2.IsWorkspace(baseDir) {
		return computeTreeDigests(filepath.Join(baseDir, small.SmallDir))
	}
	return computeArtifactDigestsInDir(filepath.Join(baseDir, small.SmallDir))
}

func writeV2Snapshot(baseDir, storeDir string, force bool) (*Snapshot, error) {
	store, err := sessionv2.Load(baseDir)
	if err != nil {
		return nil, err
	}
	state, err := sessionv2.Strict(store)
	if err != nil {
		return nil, fmt.Errorf("v2 snapshot requires reconciled strict state: %w", err)
	}
	storeDir = ResolveStoreDir(baseDir, storeDir)
	snapshotDir := filepath.Join(storeDir, state.Frontier)
	if _, err := os.Stat(snapshotDir); err == nil {
		if !force {
			return nil, fmt.Errorf("run snapshot exists, pass --force to overwrite")
		}
		if err := os.RemoveAll(snapshotDir); err != nil {
			return nil, err
		}
	}
	stateRoot := filepath.Join(snapshotDir, "state", small.SmallDir)
	if err := copyTree(filepath.Join(baseDir, small.SmallDir), stateRoot); err != nil {
		return nil, err
	}
	aggregate, perFile, err := computeTreeDigests(filepath.Join(baseDir, small.SmallDir))
	if err != nil {
		return nil, err
	}
	gitSHA, gitDirty, branch := resolveGitInfo(baseDir)
	meta := Meta{ReplayID: state.Frontier, ArtifactDigest: aggregate, ArtifactDigests: perFile, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano), GitSHA: gitSHA, GitDirty: gitDirty, Branch: branch, CLIVersion: version.GetVersion(), WorkspaceKind: "v2-session-profile", SourceDir: baseDir}
	if err := WriteMeta(snapshotDir, meta); err != nil {
		return nil, err
	}
	summary := ""
	if len(state.Handoffs) > 0 {
		summary, _ = state.Handoffs[len(state.Handoffs)-1].Payload["summary"].(string)
	}
	artifacts := []string{filepath.Join(snapshotDir, MetaFileName), stateRoot}
	createdAt, _ := time.Parse(time.RFC3339Nano, meta.CreatedAt)
	return &Snapshot{ReplayID: state.Frontier, Dir: snapshotDir, Meta: meta, Artifacts: artifacts, HandoffSummary: summary, CreatedAt: createdAt}, nil
}

func computeTreeDigests(root string) (string, map[string]string, error) {
	digests := map[string]string{}
	lines := []string{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("authoritative state contains symlink %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("authoritative state contains non-regular file %s", path)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		digest := hex.EncodeToString(sum[:])
		digests[rel] = digest
		lines = append(lines, rel+"\t"+digest)
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	sort.Strings(lines)
	aggregate := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(aggregate[:]), digests, nil
}

func copyTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing snapshot symlink %s", path)
		}
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("refusing snapshot non-regular file %s", path)
		}
		return copyFile(path, target)
	})
}

// computeArtifactDigestsInDir hashes the canonical artifacts located directly
// inside dir. Live workspaces store them under .small/; run snapshots store
// them flat in the snapshot directory, so snapshot verification points dir at
// the snapshot root. workspace.small.yml is intentionally excluded: it carries
// the replay_id and updated_at that mutate as part of taking a snapshot, so
// including it would make the digest self-referential.
func computeArtifactDigestsInDir(dir string) (string, map[string]string, error) {
	filenames := canonicalArtifactFilenames()
	digests := make(map[string]string, len(filenames))
	aggregateLines := make([]string, 0, len(filenames))

	for _, filename := range filenames {
		path := filepath.Join(dir, filename)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) && isOptionalArtifact(filename) {
				continue
			}
			return "", nil, fmt.Errorf("failed to read %s for artifact digest: %w", filename, err)
		}

		sum := sha256.Sum256(data)
		digest := hex.EncodeToString(sum[:])
		digests[filename] = digest
		aggregateLines = append(aggregateLines, filename+"\t"+digest)
	}

	aggregate := sha256.Sum256([]byte(strings.Join(aggregateLines, "\n")))
	return hex.EncodeToString(aggregate[:]), digests, nil
}

func canonicalArtifactFilenames() []string {
	filenames := make([]string, 0, len(RequiredArtifacts)+len(OptionalArtifacts))
	filenames = append(filenames, RequiredArtifacts...)
	filenames = append(filenames, OptionalArtifacts...)
	return filenames
}

func isOptionalArtifact(filename string) bool {
	for _, optional := range OptionalArtifacts {
		if filename == optional {
			return true
		}
	}
	return false
}

// DigestMismatch describes a single artifact whose on-disk digest no longer
// matches the value recorded in the snapshot's meta.json. Reason is one of
// "changed", "missing_on_disk", or "not_recorded".
type DigestMismatch struct {
	Filename string `json:"filename"`
	Recorded string `json:"recorded,omitempty"`
	Computed string `json:"computed,omitempty"`
	Reason   string `json:"reason"`
}

// SnapshotVerification is the result of recomputing a snapshot's artifact
// digests and comparing them to the values stored in its meta.json.
type SnapshotVerification struct {
	ReplayID       string           `json:"replayId"`
	DigestRecorded bool             `json:"digest_recorded"`
	RecordedDigest string           `json:"recorded_digest,omitempty"`
	ComputedDigest string           `json:"computed_digest"`
	Match          bool             `json:"match"`
	Mismatches     []DigestMismatch `json:"mismatches,omitempty"`
}

// VerifySnapshot recomputes the artifact digests for a stored snapshot and
// compares them to the values recorded when the snapshot was written. It
// detects tampering or corruption of the snapshot's own copied artifacts.
//
// Snapshots written before digest recording have no recorded digest; for those
// DigestRecorded is false and Match is false (integrity is unverifiable rather
// than confirmed). Callers should treat that case as a warning, not a failure.
func VerifySnapshot(storeDir, replayID string) (*SnapshotVerification, error) {
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

	computedAggregate, computedPerFile, err := computeArtifactDigestsInDir(snapshotDir)
	if _, statErr := os.Stat(filepath.Join(snapshotDir, "state", small.SmallDir, "profile.json")); statErr == nil {
		computedAggregate, computedPerFile, err = computeTreeDigests(filepath.Join(snapshotDir, "state", small.SmallDir))
	}
	if err != nil {
		return nil, err
	}

	result := &SnapshotVerification{
		ReplayID:       replayID,
		DigestRecorded: meta.ArtifactDigest != "",
		RecordedDigest: meta.ArtifactDigest,
		ComputedDigest: computedAggregate,
	}

	for filename, recorded := range meta.ArtifactDigests {
		computed, ok := computedPerFile[filename]
		if !ok {
			result.Mismatches = append(result.Mismatches, DigestMismatch{Filename: filename, Recorded: recorded, Reason: "missing_on_disk"})
			continue
		}
		if computed != recorded {
			result.Mismatches = append(result.Mismatches, DigestMismatch{Filename: filename, Recorded: recorded, Computed: computed, Reason: "changed"})
		}
	}
	for filename, computed := range computedPerFile {
		if _, ok := meta.ArtifactDigests[filename]; !ok && len(meta.ArtifactDigests) > 0 {
			result.Mismatches = append(result.Mismatches, DigestMismatch{Filename: filename, Computed: computed, Reason: "not_recorded"})
		}
	}

	sort.Slice(result.Mismatches, func(i, j int) bool {
		return result.Mismatches[i].Filename < result.Mismatches[j].Filename
	})

	result.Match = result.DigestRecorded && computedAggregate == meta.ArtifactDigest && len(result.Mismatches) == 0
	return result, nil
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

	handoffInfo := HandoffInfo{}
	stateRoot := filepath.Join(snapshotDir, "state")
	if isV2Snapshot(snapshotDir) {
		store, loadErr := sessionv2.Load(stateRoot)
		if loadErr != nil {
			return nil, fmt.Errorf("load v2 snapshot: %w", loadErr)
		}
		state, reduceErr := sessionv2.Reduce(store)
		if reduceErr != nil {
			return nil, fmt.Errorf("reduce v2 snapshot: %w", reduceErr)
		}
		if len(state.Handoffs) > 0 {
			latest := state.Handoffs[len(state.Handoffs)-1]
			handoffInfo.Summary, _ = latest.Payload["summary"].(string)
			handoffInfo.NextSteps = payloadStrings(latest.Payload["next_steps"])
		}
	} else {
		var handoffErr error
		handoffInfo, handoffErr = readHandoff(filepath.Join(snapshotDir, "handoff.small.yml"))
		if handoffErr != nil {
			return nil, handoffErr
		}
	}

	artifacts := []string{filepath.Join(snapshotDir, MetaFileName)}
	if isV2Snapshot(snapshotDir) {
		for filename := range meta.ArtifactDigests {
			artifacts = append(artifacts, filepath.Join(stateRoot, small.SmallDir, filepath.FromSlash(filename)))
		}
		sort.Strings(artifacts[1:])
	} else {
		for _, filename := range append(RequiredArtifacts, OptionalArtifacts...) {
			path := filepath.Join(snapshotDir, filename)
			if _, err := os.Stat(path); err == nil {
				artifacts = append(artifacts, path)
			}
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
	if isV2Snapshot(snapshot.Dir) {
		verification, err := VerifySnapshot(storeDir, replayID)
		if err != nil {
			return err
		}
		if !verification.DigestRecorded || !verification.Match {
			return fmt.Errorf("refusing to restore v2 snapshot that does not match its recorded digest")
		}
		currentDigest, _, digestErr := computeTreeDigests(smallDir)
		if digestErr == nil && currentDigest == verification.ComputedDigest {
			return nil
		}
		if !force {
			return fmt.Errorf("workspace .small tree differs from v2 snapshot, pass --force to replace it")
		}
		return replaceV2StateTree(baseDir, filepath.Join(snapshot.Dir, "state", small.SmallDir), verification.ComputedDigest)
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

func isV2Snapshot(snapshotDir string) bool {
	_, err := os.Stat(filepath.Join(snapshotDir, "state", small.SmallDir, "profile.json"))
	return err == nil
}

func replaceV2StateTree(baseDir, source, expectedDigest string) error {
	cacheDir := filepath.Join(baseDir, ".small-cache")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(cacheDir, "checkout-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	prepared := filepath.Join(staging, "new.small")
	if err := copyTree(source, prepared); err != nil {
		return fmt.Errorf("prepare v2 snapshot checkout: %w", err)
	}
	digest, _, err := computeTreeDigests(prepared)
	if err != nil {
		return err
	}
	if digest != expectedDigest {
		return fmt.Errorf("prepared v2 snapshot digest mismatch")
	}
	target := filepath.Join(baseDir, small.SmallDir)
	backup := filepath.Join(staging, "old.small")
	if err := os.Rename(target, backup); err != nil {
		return fmt.Errorf("stage existing .small tree: %w", err)
	}
	if err := os.Rename(prepared, target); err != nil {
		_ = os.Rename(backup, target)
		return fmt.Errorf("publish v2 snapshot checkout: %w", err)
	}
	if err := syncDirectory(baseDir); err != nil {
		return fmt.Errorf("sync v2 snapshot checkout: %w", err)
	}
	return nil
}

func syncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func payloadStrings(value any) []string {
	values := []string{}
	switch typed := value.(type) {
	case []string:
		return append(values, typed...)
	case []any:
		for _, item := range typed {
			if text, ok := item.(string); ok {
				values = append(values, text)
			}
		}
	}
	return values
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

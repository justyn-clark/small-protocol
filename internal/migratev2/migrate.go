package migratev2

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/justyn-clark/small-protocol/internal/sessionv2"
	"github.com/justyn-clark/small-protocol/internal/small"
	"gopkg.in/yaml.v3"
)

const PlanVersion = "small-migration-plan/v1"

type InputFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}
type TaskMapping struct {
	LegacyID string `json:"legacy_id"`
	TaskID   string `json:"task_id"`
}
type Plan struct {
	PlanVersion         string        `json:"plan_version"`
	To                  string        `json:"to"`
	Namespace           string        `json:"namespace"`
	Mode                string        `json:"mode"`
	ExpectedInputDigest string        `json:"expected_input_digest"`
	ImportID            string        `json:"import_id"`
	ProjectID           string        `json:"project_id"`
	LineageID           string        `json:"lineage_id"`
	PolicyRevision      string        `json:"policy_revision"`
	Files               []InputFile   `json:"files"`
	Tasks               []TaskMapping `json:"tasks"`
}
type Result struct {
	ImportID   string `json:"import_id"`
	ProjectID  string `json:"project_id"`
	LineageID  string `json:"lineage_id"`
	Frontier   string `json:"frontier"`
	EventCount int    `json:"event_count"`
	Idempotent bool   `json:"idempotent"`
	BackupPath string `json:"backup_path,omitempty"`
}

type legacyPlan struct {
	Tasks []struct {
		ID           string   `yaml:"id"`
		Title        string   `yaml:"title"`
		Acceptance   []string `yaml:"acceptance"`
		Dependencies []string `yaml:"dependencies"`
		Status       string   `yaml:"status"`
	} `yaml:"tasks"`
}
type legacyProgress struct {
	Entries []map[string]any `yaml:"entries"`
}
type legacyHandoff struct {
	Summary  string `yaml:"summary"`
	ReplayID struct {
		Value string `yaml:"value"`
	} `yaml:"replayId"`
}
type manifest struct {
	Profile          string        `json:"profile"`
	ImportID         string        `json:"import_id"`
	Namespace        string        `json:"namespace"`
	InputDigest      string        `json:"input_digest"`
	OriginalReplayID string        `json:"original_replay_id,omitempty"`
	Files            []InputFile   `json:"files"`
	Tasks            []TaskMapping `json:"tasks"`
}
type journal struct {
	ImportID  string `json:"import_id"`
	Stage     string `json:"stage"`
	Candidate string `json:"candidate"`
	Backup    string `json:"backup"`
}

func Preview(baseDir, namespace, mode string) (Plan, error) {
	if strings.TrimSpace(namespace) == "" {
		return Plan{}, fmt.Errorf("migration namespace is required")
	}
	if mode == "" {
		mode = "solo"
	}
	if mode != "solo" && mode != "collaborative" {
		return Plan{}, fmt.Errorf("mode must be solo or collaborative")
	}
	files, digest, contents, err := readLegacyTree(baseDir)
	if err != nil {
		return Plan{}, err
	}
	var lp legacyPlan
	if data := contents["plan.small.yml"]; data != nil {
		if err := yaml.Unmarshal(data, &lp); err != nil {
			return Plan{}, fmt.Errorf("plan.small.yml: %w", err)
		}
	}
	ns := strings.TrimSpace(namespace)
	projectID := stableID("project", ns, digest)
	importID := stableID("import", ns, digest)
	lineageSource := digest
	var handoff legacyHandoff
	if data := contents["handoff.small.yml"]; data != nil && yaml.Unmarshal(data, &handoff) == nil && strings.TrimSpace(handoff.ReplayID.Value) != "" {
		lineageSource = strings.TrimSpace(handoff.ReplayID.Value)
	}
	policyRevision := stableID("policy", ns, hashParts(string(contents["intent.small.yml"]), string(contents["constraints.small.yml"])))
	plan := Plan{PlanVersion: PlanVersion, To: sessionv2.ProfileVersion, Namespace: ns, Mode: mode, ExpectedInputDigest: digest, ImportID: importID, ProjectID: projectID, LineageID: stableID("lineage", ns, lineageSource), PolicyRevision: policyRevision, Files: files}
	for _, task := range lp.Tasks {
		plan.Tasks = append(plan.Tasks, TaskMapping{LegacyID: task.ID, TaskID: stableID("task", ns, digest, task.ID)})
	}
	return plan, nil
}

func Apply(baseDir string, plan Plan) (Result, error) {
	if err := validatePlan(plan); err != nil {
		return Result{}, err
	}
	var result Result
	err := small.WithStateLock(baseDir, func() error {
		if err := recoverLocked(baseDir); err != nil {
			return err
		}
		if existing, ok := currentImport(baseDir); ok {
			if existing != plan.ImportID {
				return fmt.Errorf("workspace already uses v2 import %s, not %s", existing, plan.ImportID)
			}
			store, err := sessionv2.Load(baseDir)
			if err != nil {
				return err
			}
			state, err := sessionv2.Strict(store)
			if err != nil {
				return err
			}
			result = Result{ImportID: plan.ImportID, ProjectID: store.Profile.ProjectID, LineageID: store.Profile.LineageID, Frontier: state.Frontier, EventCount: state.EventCount, Idempotent: true}
			return nil
		}
		_, digest, contents, err := readLegacyTree(baseDir)
		if err != nil {
			return err
		}
		if digest != plan.ExpectedInputDigest {
			return fmt.Errorf("%w: migration expected %s, actual %s", sessionv2.ErrStaleFrontier, plan.ExpectedInputDigest, digest)
		}
		root := filepath.Join(baseDir, ".small-cache", "migrations", plan.ImportID)
		candidate := filepath.Join(root, "candidate")
		backup := filepath.Join(root, "legacy.small")
		if err := os.RemoveAll(candidate); err != nil {
			return err
		}
		if err := os.MkdirAll(candidate, 0o755); err != nil {
			return err
		}
		events, err := buildCandidate(candidate, plan, contents)
		if err != nil {
			return err
		}
		store, err := sessionv2.Load(candidate)
		if err != nil {
			return fmt.Errorf("candidate validation: %w", err)
		}
		state, err := sessionv2.Strict(store)
		if err != nil {
			return fmt.Errorf("candidate strict validation: %w", err)
		}
		_, digestAfter, _, err := readLegacyTree(baseDir)
		if err != nil {
			return err
		}
		if digestAfter != plan.ExpectedInputDigest {
			return fmt.Errorf("%w: migration input changed during candidate build", sessionv2.ErrStaleFrontier)
		}
		j := journal{ImportID: plan.ImportID, Stage: "prepared", Candidate: candidate, Backup: backup}
		if err := writeJournal(root, j); err != nil {
			return err
		}
		if fault("prepared") {
			return fmt.Errorf("simulated migration interruption after prepared")
		}
		if _, err := os.Stat(backup); os.IsNotExist(err) {
			if err := os.Rename(filepath.Join(baseDir, ".small"), backup); err != nil {
				return err
			}
		}
		j.Stage = "backed_up"
		if err := writeJournal(root, j); err != nil {
			return err
		}
		if fault("backup") {
			return fmt.Errorf("simulated migration interruption after backup")
		}
		if _, err := os.Stat(filepath.Join(baseDir, ".small")); os.IsNotExist(err) {
			if err := os.Rename(filepath.Join(candidate, ".small"), filepath.Join(baseDir, ".small")); err != nil {
				return err
			}
		}
		j.Stage = "published"
		if err := writeJournal(root, j); err != nil {
			return err
		}
		if fault("publish") {
			return fmt.Errorf("simulated migration interruption after publish")
		}
		result = Result{ImportID: plan.ImportID, ProjectID: plan.ProjectID, LineageID: plan.LineageID, Frontier: state.Frontier, EventCount: events, BackupPath: backup}
		return nil
	})
	return result, err
}

func Recover(baseDir string) error {
	return small.WithStateLock(baseDir, func() error { return recoverLocked(baseDir) })
}

func buildCandidate(candidate string, plan Plan, contents map[string][]byte) (int, error) {
	intentDigest, constraintsDigest := "absent", "absent"
	if data, ok := contents["intent.small.yml"]; ok {
		intentDigest = sessionv2.PolicyMaterialDigest(data)
	}
	if data, ok := contents["constraints.small.yml"]; ok {
		constraintsDigest = sessionv2.PolicyMaterialDigest(data)
	}
	profile := sessionv2.Profile{SmallVersion: sessionv2.ProfileVersion, ProjectID: plan.ProjectID, LineageID: plan.LineageID, Mode: plan.Mode, PolicyRevision: plan.PolicyRevision, IntentDigest: intentDigest, ConstraintsDigest: constraintsDigest, ImportID: plan.ImportID}
	if err := writeCanonical(filepath.Join(candidate, ".small", "profile.json"), profile); err != nil {
		return 0, err
	}
	for source, target := range map[string]string{"intent.small.yml": "intent.small.yml", "constraints.small.yml": "constraints.small.yml"} {
		if data, ok := contents[source]; ok {
			if err := writeBytes(filepath.Join(candidate, ".small", "policy", target), data); err != nil {
				return 0, err
			}
		}
	}
	for path, data := range contents {
		if err := writeBytes(filepath.Join(candidate, ".small", "imports", "v1", plan.ImportID, "originals", filepath.FromSlash(path)), data); err != nil {
			return 0, err
		}
	}
	var handoff legacyHandoff
	_ = yaml.Unmarshal(contents["handoff.small.yml"], &handoff)
	if err := writeCanonical(filepath.Join(candidate, ".small", "imports", "v1", plan.ImportID, "manifest.json"), manifest{Profile: sessionv2.ProfileVersion, ImportID: plan.ImportID, Namespace: plan.Namespace, InputDigest: plan.ExpectedInputDigest, OriginalReplayID: strings.TrimSpace(handoff.ReplayID.Value), Files: plan.Files, Tasks: plan.Tasks}); err != nil {
		return 0, err
	}
	sessionID := stableID("session", plan.Namespace, plan.ImportID, "v1-import")
	session := sessionv2.Session{SmallVersion: sessionv2.ProfileVersion, ProjectID: plan.ProjectID, LineageID: plan.LineageID, SessionID: sessionID, Label: "v1 migration", ObservedFrontier: sessionv2.Frontier(nil), ToolVersion: "small-migrate", CreatedAt: "1970-01-01T00:00:00Z"}
	session.Digest, _ = sessionv2.DigestRecord(session)
	if err := writeCanonical(filepath.Join(candidate, ".small", "sessions", sessionID+".json"), session); err != nil {
		return 0, err
	}
	sequence := int64(0)
	previous := ""
	count := 0
	appendEvent := func(kind, occurrence string, taskID string, revision int64, source string, payload map[string]any) error {
		sequence++
		id := stableID("event", plan.Namespace, plan.ImportID, kind, occurrence, taskID)
		event := sessionv2.Event{SmallVersion: sessionv2.ProfileVersion, ProjectID: plan.ProjectID, LineageID: plan.LineageID, SessionID: sessionID, EventID: id, Kind: kind, Sequence: sequence, PreviousEvent: previous, OccurredAt: occurrence, TaskID: taskID, TaskRevision: revision, PolicyRevision: plan.PolicyRevision, SourceDigest: source, Payload: payload}
		event.Digest, _ = sessionv2.DigestRecord(event)
		if err := writeCanonical(filepath.Join(candidate, ".small", "events", sessionID, id+".json"), event); err != nil {
			return err
		}
		previous = id
		count++
		return nil
	}
	if err := appendEvent("session_started", "1970-01-01T00:00:00Z", "", 0, plan.ExpectedInputDigest, map[string]any{"imported": true}); err != nil {
		return 0, err
	}
	if err := appendEvent("v1_imported", "1970-01-01T00:00:01Z", "", 0, plan.ExpectedInputDigest, map[string]any{"import_id": plan.ImportID, "namespace": plan.Namespace, "input_digest": plan.ExpectedInputDigest}); err != nil {
		return 0, err
	}
	var lp legacyPlan
	_ = yaml.Unmarshal(contents["plan.small.yml"], &lp)
	mappings := map[string]string{}
	for _, mapping := range plan.Tasks {
		mappings[mapping.LegacyID] = mapping.TaskID
	}
	for i, task := range lp.Tasks {
		payload := map[string]any{"alias": task.ID, "title": task.Title, "acceptance": task.Acceptance, "dependencies": task.Dependencies, "legacy_status": task.Status}
		if err := appendEvent("task_created", fmt.Sprintf("legacy-plan-%06d", i), mappings[task.ID], 1, plan.ExpectedInputDigest, payload); err != nil {
			return 0, err
		}
	}
	var progress legacyProgress
	_ = yaml.Unmarshal(contents["progress.small.yml"], &progress)
	lastStatus := map[string]string{}
	for i, entry := range progress.Entries {
		legacyID := text(entry["task_id"])
		taskID := mappings[legacyID]
		if taskID == "" {
			taskID = stableID("task", plan.Namespace, plan.ExpectedInputDigest, "orphan", legacyID, fmt.Sprint(i))
			mappings[legacyID] = taskID
			if err := appendEvent("task_created", fmt.Sprintf("legacy-progress-orphan-%06d", i), taskID, 1, plan.ExpectedInputDigest, map[string]any{"alias": legacyID, "title": "Imported legacy progress task"}); err != nil {
				return 0, err
			}
		}
		status := text(entry["status"])
		occurrence := text(entry["timestamp"])
		if occurrence == "" {
			occurrence = fmt.Sprintf("legacy-progress-%06d", i)
		}
		if status != "" {
			if err := appendEvent("task_transitioned", occurrence+fmt.Sprintf("#%06d", i), taskID, 1, plan.ExpectedInputDigest, map[string]any{"status": status, "legacy_entry": entry}); err != nil {
				return 0, err
			}
			lastStatus[legacyID] = status
		}
	}
	for i, task := range lp.Tasks {
		status := strings.TrimSpace(task.Status)
		if status != "" && status != "pending" && lastStatus[task.ID] != status {
			if err := appendEvent("task_transitioned", fmt.Sprintf("legacy-plan-status-%06d", i), mappings[task.ID], 1, plan.ExpectedInputDigest, map[string]any{"status": status, "source": "plan.small.yml"}); err != nil {
				return 0, err
			}
		}
	}
	if strings.TrimSpace(handoff.Summary) != "" {
		if err := appendEvent("handoff_recorded", "legacy-handoff", "", 0, plan.ExpectedInputDigest, map[string]any{"summary": handoff.Summary, "authoritative": true, "legacy_replay_id": strings.TrimSpace(handoff.ReplayID.Value)}); err != nil {
			return 0, err
		}
	}
	if err := appendEvent("session_closed", "legacy-import-complete", "", 0, plan.ExpectedInputDigest, map[string]any{"reason": "migration import session complete"}); err != nil {
		return 0, err
	}
	return count, nil
}

func readLegacyTree(baseDir string) ([]InputFile, string, map[string][]byte, error) {
	root := filepath.Join(baseDir, ".small")
	if _, err := os.Stat(filepath.Join(root, "profile.json")); err == nil {
		return nil, "", nil, sessionv2.ErrNotV2
	}
	files := []InputFile{}
	contents := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("legacy state contains symlink %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("legacy state contains non-regular file %s", path)
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
		files = append(files, InputFile{Path: rel, SHA256: hex.EncodeToString(sum[:]), Size: int64(len(data))})
		contents[rel] = data
		return nil
	})
	if err != nil {
		return nil, "", nil, err
	}
	if len(files) == 0 {
		return nil, "", nil, fmt.Errorf("no legacy .small files found")
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	parts := []string{}
	for _, file := range files {
		parts = append(parts, file.Path+"\x00"+file.SHA256+"\x00"+fmt.Sprint(file.Size))
	}
	return files, hashParts(parts...), contents, nil
}

func validatePlan(plan Plan) error {
	if plan.PlanVersion != PlanVersion || plan.To != sessionv2.ProfileVersion {
		return fmt.Errorf("unsupported migration plan")
	}
	if plan.Namespace == "" || plan.ExpectedInputDigest == "" || plan.ImportID == "" || plan.ProjectID == "" || plan.LineageID == "" || plan.PolicyRevision == "" {
		return fmt.Errorf("migration plan identity fields are required")
	}
	if plan.Mode != "solo" && plan.Mode != "collaborative" {
		return fmt.Errorf("invalid migration mode")
	}
	return nil
}
func currentImport(baseDir string) (string, bool) {
	data, err := os.ReadFile(filepath.Join(baseDir, ".small", "profile.json"))
	if err != nil {
		return "", false
	}
	var profile sessionv2.Profile
	if sessionv2.DecodeStrict(data, &profile) != nil || profile.SmallVersion != sessionv2.ProfileVersion {
		return "", false
	}
	return profile.ImportID, true
}
func recoverLocked(baseDir string) error {
	root := filepath.Join(baseDir, ".small-cache", "migrations")
	entries, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		data, err := os.ReadFile(filepath.Join(dir, "journal.json"))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return err
		}
		var j journal
		if err := json.Unmarshal(data, &j); err != nil {
			return err
		}
		if j.Stage == "published" {
			continue
		}
		target := filepath.Join(baseDir, ".small")
		if _, err := os.Stat(target); os.IsNotExist(err) {
			if _, err := os.Stat(filepath.Join(j.Candidate, ".small")); err == nil {
				if err := os.Rename(filepath.Join(j.Candidate, ".small"), target); err != nil {
					return err
				}
			} else if _, err := os.Stat(j.Backup); err == nil {
				if err := os.Rename(j.Backup, target); err != nil {
					return err
				}
			}
		}
		if _, ok := currentImport(baseDir); ok {
			j.Stage = "published"
			if err := writeJournal(dir, j); err != nil {
				return err
			}
		}
	}
	return nil
}
func writeJournal(root string, j journal) error {
	return writeCanonical(filepath.Join(root, "journal.json"), j)
}
func writeCanonical(path string, value any) error {
	data, err := sessionv2.CanonicalJSON(value)
	if err != nil {
		return err
	}
	return writeBytes(path, append(data, '\n'))
}
func writeBytes(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
func stableID(prefix string, parts ...string) string { return prefix + "_" + hashParts(parts...)[:32] }
func hashParts(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}
func text(value any) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}
func fault(stage string) bool {
	return strings.TrimSpace(os.Getenv("SMALL_TEST_FAIL_MIGRATION_AFTER")) == stage
}

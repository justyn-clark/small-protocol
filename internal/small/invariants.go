package small

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

type InvariantViolation struct {
	File    string
	Message string
}

// DanglingTask represents a task that has progress entries but is not in a terminal state
type DanglingTask struct {
	ID     string
	Title  string
	Status string
}

func CheckInvariants(artifacts map[string]*Artifact, strict bool) []InvariantViolation {
	var violations []InvariantViolation

	for artifactType, artifact := range artifacts {
		root := artifact.Data
		if root == nil {
			violations = append(violations, InvariantViolation{
				File: artifact.Path, Message: "artifact must be a YAML mapping (object) at top level",
			})
			continue
		}

		// --- Global: version ---
		v, vok := root["small_version"].(string)
		if !vok || v != ProtocolVersion {
			violations = append(violations, InvariantViolation{
				File:    artifact.Path,
				Message: fmt.Sprintf(`small_version must be exactly "%s", got: %v`, ProtocolVersion, root["small_version"]),
			})
		}

		// --- Global: owner ---
		owner, ook := root["owner"].(string)
		if !ook || (owner != "human" && owner != "agent") {
			violations = append(violations, InvariantViolation{
				File:    artifact.Path,
				Message: fmt.Sprintf(`owner must be "human" or "agent", got: %v`, root["owner"]),
			})
		}

		// --- Global: allowed top-level keys (prevents silent spec drift) ---
		allowed := allowedTopLevelKeys(artifactType)
		if allowed == nil {
			violations = append(violations, InvariantViolation{
				File:    artifact.Path,
				Message: fmt.Sprintf("unknown artifact type: %s", artifactType),
			})
		} else {
			for k := range root {
				if !allowed[k] {
					violations = append(violations, InvariantViolation{
						File:    artifact.Path,
						Message: fmt.Sprintf("unknown top-level key %q for %s artifact", k, artifactType),
					})
				}
			}
		}

		// --- Per-type rules ---
		switch artifactType {
		case "intent":
			violations = append(violations, validateIntent(artifact.Path, root, owner)...)
		case "constraints":
			violations = append(violations, validateConstraints(artifact.Path, root, owner)...)
		case "plan":
			violations = append(violations, validatePlan(artifact.Path, root, owner)...)
		case "progress":
			violations = append(violations, validateProgress(artifact.Path, root, owner)...)
		case "handoff":
			violations = append(violations, validateHandoff(artifact.Path, root, owner)...)
		}

		// --- Strict mode: secrets + link hygiene ---
		if strict {
			violations = append(violations, checkSecrets(artifact)...)
			violations = append(violations, checkInsecureLinks(artifact)...)
		}
	}

	if strict {
		violations = append(violations, validateStrictInvariants(artifacts)...)
	}

	return violations

}

type planTask struct {
	ID     string
	Title  string
	Status string
}

type strictInvariantConfig struct {
	RequireReconcileMarker bool
}

func validateStrictInvariants(artifacts map[string]*Artifact) []InvariantViolation {
	return validateStrictInvariantsWithConfig(artifacts, strictInvariantConfig{RequireReconcileMarker: false})
}

func validateStrictInvariantsWithConfig(artifacts map[string]*Artifact, config strictInvariantConfig) []InvariantViolation {
	var violations []InvariantViolation

	planArtifact, hasPlan := artifacts["plan"]
	progressArtifact, hasProgress := artifacts["progress"]
	if hasPlan && hasProgress {
		violations = append(violations, validateStrictPlanTaskEvidence(planArtifact, progressArtifact)...)
		violations = append(violations, validateStrictProgressTaskIDs(planArtifact, progressArtifact)...)
		if config.RequireReconcileMarker {
			violations = append(violations, validatePlanReconciliation(planArtifact, progressArtifact)...)
		}
	}

	if hasPlan {
		if handoffArtifact, hasHandoff := artifacts["handoff"]; hasHandoff {
			violations = append(violations, validateStrictHandoffTasks(planArtifact, handoffArtifact)...)
		}
	}

	return violations
}

func allowedTopLevelKeys(artifactType string) map[string]bool {
	base := map[string]bool{
		"small_version": true,
		"owner":         true,
	}

	switch artifactType {
	case "intent":
		base["intent"] = true
		base["scope"] = true
		base["success_criteria"] = true
		return base
	case "constraints":
		base["constraints"] = true
		return base
	case "plan":
		base["tasks"] = true
		return base
	case "progress":
		base["entries"] = true
		return base
	case "handoff":
		base["summary"] = true
		base["resume"] = true
		base["links"] = true
		base["replayId"] = true
		base["run"] = true
		return base
	default:
		return nil
	}
}

func validateStrictPlanTaskEvidence(planArtifact, progressArtifact *Artifact) []InvariantViolation {
	if planArtifact == nil || progressArtifact == nil || planArtifact.Data == nil || progressArtifact.Data == nil {
		return nil
	}

	tasks := extractPlanTasks(planArtifact)
	if len(tasks) == 0 {
		return nil
	}

	entries := extractProgressEntries(progressArtifact)
	if entries == nil {
		return nil
	}

	satisfied := map[string]struct{}{}
	for _, entry := range entries {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		taskID := strings.TrimSpace(stringVal(entryMap["task_id"]))
		if taskID == "" {
			continue
		}
		if ProgressEntryHasStrictEvidence(entryMap) {
			satisfied[taskID] = struct{}{}
		}
	}

	var missing []string
	for _, task := range tasks {
		status := strings.ToLower(strings.TrimSpace(task.Status))
		if status != "completed" && status != "blocked" {
			continue
		}
		if task.ID == "" {
			continue
		}
		if _, ok := satisfied[task.ID]; !ok {
			reason := "no progress entry"
			if hasProgressEntriesForTask(entries, task.ID) {
				reason = "empty evidence/notes"
			}
			missing = append(missing, fmt.Sprintf("%s (%s) [%s]", task.ID, task.Title, reason))
		}
	}

	if len(missing) == 0 {
		return nil
	}

	sort.Strings(missing)
	return []InvariantViolation{{
		File:    progressArtifact.Path,
		Message: fmt.Sprintf("strict invariant S1 failed: %s", strings.Join(missing, "; ")),
	}}
}

func validateStrictProgressTaskIDs(planArtifact, progressArtifact *Artifact) []InvariantViolation {
	if planArtifact == nil || progressArtifact == nil || planArtifact.Data == nil || progressArtifact.Data == nil {
		return nil
	}

	tasks := extractPlanTasks(planArtifact)
	if len(tasks) == 0 {
		return nil
	}

	knownTasks := map[string]struct{}{}
	for _, task := range tasks {
		if task.ID != "" {
			knownTasks[task.ID] = struct{}{}
		}
	}

	entries := extractProgressEntries(progressArtifact)
	if entries == nil {
		return nil
	}

	var offenders []string
	for _, entry := range entries {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		taskID := strings.TrimSpace(stringVal(entryMap["task_id"]))
		if taskID == "" {
			continue
		}
		if strings.HasPrefix(taskID, "meta/") {
			continue
		}
		if _, ok := knownTasks[taskID]; ok {
			continue
		}
		closest := closestTaskIDs(taskID, tasks, 3)
		if len(closest) > 0 {
			offenders = append(offenders, fmt.Sprintf("%s (closest: %s)", taskID, strings.Join(closest, ", ")))
		} else {
			offenders = append(offenders, taskID)
		}
	}

	if len(offenders) == 0 {
		return nil
	}

	sort.Strings(offenders)
	return []InvariantViolation{{
		File:    progressArtifact.Path,
		Message: fmt.Sprintf("strict invariant S2 failed: unknown progress task ids: %s", strings.Join(offenders, "; ")),
	}}
}

func validateStrictHandoffTasks(planArtifact, handoffArtifact *Artifact) []InvariantViolation {
	if planArtifact == nil || handoffArtifact == nil || planArtifact.Data == nil || handoffArtifact.Data == nil {
		return nil
	}

	tasks := extractPlanTasks(planArtifact)
	if len(tasks) == 0 {
		return nil
	}

	knownTasks := map[string]planTask{}
	for _, task := range tasks {
		if task.ID != "" {
			knownTasks[task.ID] = task
		}
	}

	handoffRoot := handoffArtifact.Data
	resume, _ := handoffRoot["resume"].(map[string]interface{})
	if resume == nil {
		return nil
	}

	currentTaskID := strings.TrimSpace(stringVal(resume["current_task_id"]))
	if currentTaskID == "" {
		return nil
	}

	var violations []InvariantViolation
	if _, ok := knownTasks[currentTaskID]; !ok {
		violations = append(violations, InvariantViolation{
			File:    handoffArtifact.Path,
			Message: fmt.Sprintf("strict invariant S3 failed: resume.current_task_id references %q which is missing from plan", currentTaskID),
		})
	}

	return violations
}

func validatePlanReconciliation(planArtifact, progressArtifact *Artifact) []InvariantViolation {
	if planArtifact == nil || progressArtifact == nil {
		return nil
	}
	return nil
}

func extractPlanTasks(planArtifact *Artifact) []planTask {
	tasks, ok := planArtifact.Data["tasks"].([]interface{})
	if !ok {
		return nil
	}

	var result []planTask
	for _, raw := range tasks {
		taskMap, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		result = append(result, planTask{
			ID:     strings.TrimSpace(stringVal(taskMap["id"])),
			Title:  strings.TrimSpace(stringVal(taskMap["title"])),
			Status: strings.TrimSpace(stringVal(taskMap["status"])),
		})
	}

	return result
}

func extractProgressEntries(progressArtifact *Artifact) []interface{} {
	entries, ok := progressArtifact.Data["entries"].([]interface{})
	if !ok {
		return nil
	}
	return entries
}

func hasProgressEntriesForTask(entries []interface{}, taskID string) bool {
	for _, entry := range entries {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if strings.TrimSpace(stringVal(entryMap["task_id"])) == taskID {
			return true
		}
	}
	return false
}

func ProgressEntryHasStrictEvidence(entry map[string]interface{}) bool {
	if entry == nil {
		return false
	}
	if strings.TrimSpace(stringVal(entry["evidence"])) != "" {
		return true
	}
	if strings.TrimSpace(stringVal(entry["notes"])) != "" {
		return true
	}
	return false
}

func stringVal(value interface{}) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}

func closestTaskIDs(taskID string, tasks []planTask, limit int) []string {
	if limit <= 0 {
		return nil
	}

	var matches []string
	for _, task := range tasks {
		if task.ID == "" {
			continue
		}
		if strings.Contains(task.ID, taskID) || strings.Contains(taskID, task.ID) {
			matches = append(matches, task.ID)
		}
	}
	if len(matches) > 0 {
		sort.Strings(matches)
		if len(matches) > limit {
			return matches[:limit]
		}
		return matches
	}

	for _, task := range tasks {
		if task.ID == "" {
			continue
		}
		if strings.EqualFold(task.ID, taskID) {
			matches = append(matches, task.ID)
		}
	}
	if len(matches) > 0 {
		sort.Strings(matches)
		if len(matches) > limit {
			return matches[:limit]
		}
		return matches
	}

	return nil
}

func validateIntent(path string, root map[string]interface{}, owner string) []InvariantViolation {
	var v []InvariantViolation
	if owner != "human" {
		v = append(v, InvariantViolation{File: path, Message: `intent must have owner: "human"`})
	}

	intent, ok := root["intent"].(string)
	if !ok || strings.TrimSpace(intent) == "" {
		v = append(v, InvariantViolation{File: path, Message: "intent.intent must be a non-empty string"})
	}

	scope, ok := root["scope"].(map[string]interface{})
	if !ok || scope == nil {
		v = append(v, InvariantViolation{File: path, Message: "intent.scope must be an object with include/exclude arrays"})
		return v
	}
	if _, ok := scope["include"].([]interface{}); !ok {
		v = append(v, InvariantViolation{File: path, Message: "intent.scope.include must be an array"})
	}
	if _, ok := scope["exclude"].([]interface{}); !ok {
		v = append(v, InvariantViolation{File: path, Message: "intent.scope.exclude must be an array"})
	}
	if _, ok := root["success_criteria"].([]interface{}); !ok {
		v = append(v, InvariantViolation{File: path, Message: "intent.success_criteria must be an array"})
	}
	return v
}

func validateConstraints(path string, root map[string]interface{}, owner string) []InvariantViolation {
	var v []InvariantViolation
	if owner != "human" {
		v = append(v, InvariantViolation{File: path, Message: `constraints must have owner: "human"`})
	}

	items, ok := root["constraints"].([]interface{})
	if !ok || len(items) == 0 {
		v = append(v, InvariantViolation{File: path, Message: "constraints.constraints must be a non-empty array"})
		return v
	}

	for i, it := range items {
		m, ok := it.(map[string]interface{})
		if !ok {
			v = append(v, InvariantViolation{File: path, Message: fmt.Sprintf("constraints[%d] must be an object", i)})
			continue
		}
		if s, _ := m["id"].(string); strings.TrimSpace(s) == "" {
			v = append(v, InvariantViolation{File: path, Message: fmt.Sprintf("constraints[%d].id must be a non-empty string", i)})
		}
		if s, _ := m["rule"].(string); strings.TrimSpace(s) == "" {
			v = append(v, InvariantViolation{File: path, Message: fmt.Sprintf("constraints[%d].rule must be a non-empty string", i)})
		}
		severity, _ := m["severity"].(string)
		if severity != "error" && severity != "warn" {
			v = append(v, InvariantViolation{File: path, Message: fmt.Sprintf(`constraints[%d].severity must be "error" or "warn"`, i)})
		}
	}
	return v
}

func validatePlan(path string, root map[string]interface{}, owner string) []InvariantViolation {
	var v []InvariantViolation
	if owner != "agent" {
		v = append(v, InvariantViolation{File: path, Message: `plan must have owner: "agent"`})
	}

	tasks, ok := root["tasks"].([]interface{})
	if !ok || len(tasks) == 0 {
		v = append(v, InvariantViolation{File: path, Message: "plan.tasks must be a non-empty array"})
		return v
	}

	for i, t := range tasks {
		m, ok := t.(map[string]interface{})
		if !ok {
			v = append(v, InvariantViolation{File: path, Message: fmt.Sprintf("tasks[%d] must be an object", i)})
			continue
		}
		if s, _ := m["id"].(string); strings.TrimSpace(s) == "" {
			v = append(v, InvariantViolation{File: path, Message: fmt.Sprintf("tasks[%d].id must be a non-empty string", i)})
		}
		if s, _ := m["title"].(string); strings.TrimSpace(s) == "" {
			v = append(v, InvariantViolation{File: path, Message: fmt.Sprintf("tasks[%d].title must be a non-empty string", i)})
		}
	}
	return v
}

func validateProgress(path string, root map[string]interface{}, owner string) []InvariantViolation {
	var v []InvariantViolation
	if owner != "agent" {
		v = append(v, InvariantViolation{File: path, Message: `progress must have owner: "agent"`})
	}

	entries, ok := root["entries"].([]interface{})
	if !ok {
		v = append(v, InvariantViolation{File: path, Message: "progress.entries must be an array"})
		return v
	}

	var prevTime time.Time
	evidenceFields := []string{"evidence", "verification", "command", "test", "link", "commit"}

	for i, entry := range entries {
		em, ok := entry.(map[string]interface{})
		if !ok {
			v = append(v, InvariantViolation{File: path, Message: fmt.Sprintf("entries[%d] must be an object", i)})
			continue
		}

		if s, _ := em["task_id"].(string); strings.TrimSpace(s) == "" {
			v = append(v, InvariantViolation{File: path, Message: fmt.Sprintf("progress entry %d must include task_id (non-empty string)", i)})
		}

		tsValue, _ := em["timestamp"].(string)
		parsedTs, tsErr := ParseProgressTimestamp(tsValue)
		if tsErr != nil {
			v = append(v, InvariantViolation{
				File: path,
				Message: fmt.Sprintf(
					"progress entry %d timestamp %q invalid: %v (expected RFC3339Nano with fractional seconds, strictly increasing)",
					i, tsValue, tsErr,
				),
			})
			continue
		}
		if !prevTime.IsZero() && !parsedTs.After(prevTime) {
			v = append(v, InvariantViolation{
				File: path,
				Message: fmt.Sprintf(
					"progress entry %d timestamp %q must be after previous entry %q (expected strictly increasing RFC3339Nano)",
					i, parsedTs.Format(time.RFC3339Nano), prevTime.Format(time.RFC3339Nano),
				),
			})
			continue
		}
		prevTime = parsedTs

		hasEvidence := false
		for _, f := range evidenceFields {
			if val, exists := em[f]; exists && val != nil {
				switch vv := val.(type) {
				case string:
					if strings.TrimSpace(vv) != "" {
						hasEvidence = true
					}
				default:
					hasEvidence = true
				}
			}
		}
		if !hasEvidence {
			v = append(v, InvariantViolation{
				File: path,
				Message: fmt.Sprintf(
					"progress entry %d (task_id: %v) must have at least one evidence field (evidence, verification, command, test, link, commit)",
					i, em["task_id"],
				),
			})
		}
	}

	return v
}

func ParseProgressTimestamp(value string) (time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return time.Time{}, fmt.Errorf("timestamp is required (RFC3339Nano with fractional seconds)")
	}
	if !hasFractionalSeconds(trimmed) {
		return time.Time{}, fmt.Errorf("timestamp must include fractional seconds")
	}
	parsed, err := time.Parse(time.RFC3339Nano, trimmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("timestamp must be RFC3339Nano: %w", err)
	}
	return parsed, nil
}

func hasFractionalSeconds(value string) bool {
	tIndex := strings.Index(value, "T")
	timePart := value
	if tIndex >= 0 && tIndex+1 < len(value) {
		timePart = value[tIndex+1:]
	}
	end := len(timePart)
	if idx := strings.IndexAny(timePart, "Z+-"); idx >= 0 {
		end = idx
	}
	if end < 0 || end > len(timePart) {
		end = len(timePart)
	}
	return strings.Contains(timePart[:end], ".")
}

func validateCompletedTaskProgress(planArtifact, progressArtifact *Artifact) []InvariantViolation {
	var violations []InvariantViolation
	if planArtifact == nil || progressArtifact == nil || planArtifact.Data == nil || progressArtifact.Data == nil {
		return violations
	}

	tasks, ok := planArtifact.Data["tasks"].([]interface{})
	if !ok {
		return violations
	}

	completed := map[string]struct{}{}
	for _, t := range tasks {
		taskMap, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		status, _ := taskMap["status"].(string)
		if strings.ToLower(strings.TrimSpace(status)) != "completed" {
			continue
		}
		id, _ := taskMap["id"].(string)
		if strings.TrimSpace(id) == "" {
			continue
		}
		completed[id] = struct{}{}
	}

	if len(completed) == 0 {
		return violations
	}

	entries, ok := progressArtifact.Data["entries"].([]interface{})
	if !ok {
		return violations
	}

	satisfied := map[string]struct{}{}
	for _, entry := range entries {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		taskID, _ := entryMap["task_id"].(string)
		if strings.TrimSpace(taskID) == "" {
			continue
		}
		if ProgressEntryHasValidEvidence(entryMap) {
			satisfied[taskID] = struct{}{}
		}
	}

	var missing []string
	for id := range completed {
		if _, ok := satisfied[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) == 0 {
		return violations
	}

	sort.Strings(missing)
	violations = append(violations, InvariantViolation{
		File:    progressArtifact.Path,
		Message: fmt.Sprintf("progress entries missing or invalid for completed plan tasks: %s", strings.Join(missing, ", ")),
	})
	return violations
}

func ProgressEntryHasValidEvidence(entry map[string]interface{}) bool {
	if entry == nil {
		return false
	}
	ts, _ := entry["timestamp"].(string)
	if strings.TrimSpace(ts) == "" {
		return false
	}
	if _, err := ParseProgressTimestamp(ts); err != nil {
		return false
	}
	if hasNonEmptyStringField(entry, "notes") || hasNonEmptyStringField(entry, "evidence") {
		return true
	}
	return false
}

func hasNonEmptyStringField(entry map[string]interface{}, field string) bool {
	val, ok := entry[field]
	if !ok {
		return false
	}
	s, ok := val.(string)
	if !ok {
		return false
	}
	return strings.TrimSpace(s) != ""
}

func validateHandoff(path string, root map[string]interface{}, owner string) []InvariantViolation {
	var v []InvariantViolation
	if owner != "agent" {
		v = append(v, InvariantViolation{File: path, Message: `handoff must have owner: "agent"`})
	}

	summary, _ := root["summary"].(string)
	if strings.TrimSpace(summary) == "" {
		v = append(v, InvariantViolation{File: path, Message: "handoff.summary must be a non-empty string"})
	}

	resume, ok := root["resume"].(map[string]interface{})
	if !ok || resume == nil {
		v = append(v, InvariantViolation{File: path, Message: "handoff.resume must be an object"})
		return v
	}
	if _, ok := resume["current_task_id"].(string); !ok {
		v = append(v, InvariantViolation{File: path, Message: "handoff.resume.current_task_id must be a string"})
	}
	if _, ok := resume["next_steps"].([]interface{}); !ok {
		v = append(v, InvariantViolation{File: path, Message: "handoff.resume.next_steps must be an array"})
	}

	if _, ok := root["links"].([]interface{}); !ok {
		v = append(v, InvariantViolation{File: path, Message: "handoff.links must be an array"})
	}

	return v
}

func checkInsecureLinks(artifact *Artifact) []InvariantViolation {
	var v []InvariantViolation

	root := artifact.Data
	if root == nil {
		return v
	}

	var visit func(value interface{})
	visit = func(value interface{}) {
		switch vv := value.(type) {
		case map[string]interface{}:
			for _, x := range vv {
				visit(x)
			}
		case []interface{}:
			for _, x := range vv {
				visit(x)
			}
		case string:
			if strings.Contains(vv, "http://") {
				v = append(v, InvariantViolation{
					File:    artifact.Path,
					Message: "insecure link detected (http://). Use https://",
				})
			}
		}
	}

	visit(root)
	return v
}

func checkSecrets(artifact *Artifact) []InvariantViolation {
	var violations []InvariantViolation

	root := artifact.Data
	if root == nil {
		return violations
	}

	secretKeys := []string{"api_key", "apikey", "password", "secret", "token", "access_token", "private_key"}
	tokenPattern := regexp.MustCompile(`^[A-Za-z0-9+/]{32,}={0,2}$`)

	// Paths to exclude from secrets check (known safe fields)
	excludedPaths := map[string]bool{
		"replayId.value": true, // SHA256 hash for session replay, not a secret
	}

	checkValue := func(key string, value interface{}, path string) bool {
		// Skip excluded paths
		if excludedPaths[path] {
			return false
		}

		keyLower := strings.ToLower(key)
		for _, secretKey := range secretKeys {
			if strings.Contains(keyLower, secretKey) {
				return true
			}
		}
		if str, ok := value.(string); ok {
			if tokenPattern.MatchString(str) && len(str) >= 32 {
				return true
			}
		}
		return false
	}

	var checkMap func(map[string]interface{}, string)
	checkMap = func(m map[string]interface{}, prefix string) {
		for k, val := range m {
			path := prefix + "." + k
			if prefix == "" {
				path = k
			}

			if checkValue(k, val, path) {
				violations = append(violations, InvariantViolation{
					File:    artifact.Path,
					Message: fmt.Sprintf("potential secret detected at %s", path),
				})
			}

			if nestedMap, ok := val.(map[string]interface{}); ok {
				checkMap(nestedMap, path)
			} else if nestedSlice, ok := val.([]interface{}); ok {
				for i, item := range nestedSlice {
					if itemMap, ok := item.(map[string]interface{}); ok {
						checkMap(itemMap, fmt.Sprintf("%s[%d]", path, i))
					}
				}
			}
		}
	}

	checkMap(root, "")

	return violations
}

// CheckDanglingTasks returns tasks that have progress entries but are not in a terminal state
// (completed or blocked). These are tasks where work was started but not properly closed.
func CheckDanglingTasks(planArtifact, progressArtifact *Artifact) []DanglingTask {
	if planArtifact == nil || progressArtifact == nil || planArtifact.Data == nil || progressArtifact.Data == nil {
		return nil
	}

	tasks := extractPlanTasks(planArtifact)
	if len(tasks) == 0 {
		return nil
	}

	entries := extractProgressEntries(progressArtifact)
	if entries == nil || len(entries) == 0 {
		return nil
	}

	// Build a set of task IDs that have progress entries
	tasksWithProgress := make(map[string]struct{})
	for _, entry := range entries {
		entryMap, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		taskID := strings.TrimSpace(stringVal(entryMap["task_id"]))
		if taskID == "" || strings.HasPrefix(taskID, "meta/") {
			continue
		}
		tasksWithProgress[taskID] = struct{}{}
	}

	// Find tasks that have progress but are not in a terminal state
	var dangling []DanglingTask
	for _, task := range tasks {
		if task.ID == "" {
			continue
		}
		// Check if this task has progress entries
		if _, hasProgress := tasksWithProgress[task.ID]; !hasProgress {
			continue
		}
		// Check if status is not terminal (completed or blocked)
		status := strings.ToLower(task.Status)
		if status == "completed" || status == "blocked" {
			continue
		}
		// This task has progress but is not closed
		dangling = append(dangling, DanglingTask{
			ID:     task.ID,
			Title:  task.Title,
			Status: task.Status,
		})
	}

	return dangling
}

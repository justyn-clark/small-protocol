package small

import (
	"fmt"
	"regexp"
	"strings"
)

type InvariantViolation struct {
	File    string
	Message string
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
		return base
	default:
		return nil
	}
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

	checkValue := func(key string, value interface{}) bool {
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

			if checkValue(k, val) {
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

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
		// Check version
		if version, ok := artifact.Data["small_version"].(string); !ok || version != "0.1" {
			violations = append(violations, InvariantViolation{
				File:    artifact.Path,
				Message: fmt.Sprintf("small_version must be exactly \"0.1\", got: %v", artifact.Data["small_version"]),
			})
		}

		// Check ownership
		if artifactType == "intent" || artifactType == "constraints" {
			if owner, ok := artifact.Data["owner"].(string); !ok || owner != "human" {
				violations = append(violations, InvariantViolation{
					File:    artifact.Path,
					Message: fmt.Sprintf("%s must have owner: \"human\", got: %v", artifactType, artifact.Data["owner"]),
				})
			}
		}

		if artifactType == "plan" || artifactType == "progress" {
			if owner, ok := artifact.Data["owner"].(string); !ok || owner != "agent" {
				violations = append(violations, InvariantViolation{
					File:    artifact.Path,
					Message: fmt.Sprintf("%s must have owner: \"agent\", got: %v", artifactType, artifact.Data["owner"]),
				})
			}
		}

		// Check progress evidence
		if artifactType == "progress" {
			entries, ok := artifact.Data["entries"].([]interface{})
			if ok {
				for i, entry := range entries {
					entryMap, ok := entry.(map[string]interface{})
					if !ok {
						continue
					}

					hasEvidence := false
					evidenceFields := []string{"evidence", "verification", "command", "test", "link", "commit"}
					for _, field := range evidenceFields {
						if val, exists := entryMap[field]; exists && val != nil {
							hasEvidence = true
							break
						}
					}

					if !hasEvidence {
						violations = append(violations, InvariantViolation{
							File:    artifact.Path,
							Message: fmt.Sprintf("progress entry %d (task_id: %v) must have at least one evidence field (evidence, verification, command, test, link, commit)", i, entryMap["task_id"]),
						})
					}
				}
			}
		}

		// Check for secrets (heuristic-based, only in strict mode)
		if strict {
			secretViolations := checkSecrets(artifact)
			violations = append(violations, secretViolations...)
		}
	}

	return violations
}

func checkSecrets(artifact *Artifact) []InvariantViolation {
	var violations []InvariantViolation

	secretKeys := []string{"api_key", "apiKey", "apikey", "password", "secret", "token", "access_token", "private_key", "privateKey"}
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
		for k, v := range m {
			path := prefix + "." + k
			if prefix == "" {
				path = k
			}

			if checkValue(k, v) {
				violations = append(violations, InvariantViolation{
					File:    artifact.Path,
					Message: fmt.Sprintf("potential secret detected at %s", path),
				})
			}

			if nestedMap, ok := v.(map[string]interface{}); ok {
				checkMap(nestedMap, path)
			} else if nestedSlice, ok := v.([]interface{}); ok {
				for i, item := range nestedSlice {
					if itemMap, ok := item.(map[string]interface{}); ok {
						checkMap(itemMap, fmt.Sprintf("%s[%d]", path, i))
					}
				}
			}
		}
	}

	checkMap(artifact.Data, "")

	return violations
}

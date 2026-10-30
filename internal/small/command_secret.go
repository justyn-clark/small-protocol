package small

import (
	"regexp"
	"strings"
)

// Keep value heuristics independent of assignment classification: recognizing a
// source reference must never hide credentials elsewhere in the original text.
var commandCredentialPattern = regexp.MustCompile(`(?i)(?:\bbearer\s+[a-z0-9._~+/-]{8,}|\b(?:gh[pousr]_[a-z0-9]{20,}|github_pat_[a-z0-9_]{20,}|sk-[a-z0-9_-]{16,}|AKIA[A-Z0-9]{16})\b|-----BEGIN (?:RSA |EC |OPENSSH )?PRIVATE KEY-----)`)
var commandAssignmentPattern = regexp.MustCompile(`(?i)\b(?:api[_-]?key|access[_-]?token|password|passwd|secret|token)\s*=\s*["']?[^\s"'$;<>][^\s"';]*`)

// Escaped dots cover the same source text inside a regex search literal. No
// general shell/JavaScript unescaping is performed.
const environmentReference = `process\s*(?:\.|\\\.)\s*env\s*(?:\.|\\\.)\s*[A-Za-z_][A-Za-z0-9_]*`

// This finite source grammar recognizes references, not runtime values. Only a
// complete declaration with an explicit semicolon is classified. Unknown calls,
// interpolation, escapes, operators and fallbacks retain the original warning.
var commandReferenceValue = regexp.MustCompile(`^\s*(?:` + environmentReference + `|JSON\s*\.\s*parse\s*\(\s*fs\s*\.\s*readFileSync\s*\(\s*` + environmentReference + `\s*\+\s*(?:"/[A-Za-z0-9._/-]+"|'/[A-Za-z0-9._/-]+')\s*,\s*(?:"utf-?8"|'utf-?8')\s*\)\s*\)(?:\s*\.\s*[A-Za-z_][A-Za-z0-9_]*)+)\s*;`)

func capturedCommandHasSecret(command string) bool {
	if commandCredentialPattern.MatchString(command) {
		return true
	}
	for _, match := range commandAssignmentPattern.FindAllStringIndex(command, -1) {
		if !commandDeclarationBefore(command, match[0]) {
			return true
		}
		equals := match[0] + strings.IndexByte(command[match[0]:match[1]], '=')
		if !commandReferenceValue.MatchString(command[equals+1:]) {
			return true
		}
	}
	return false
}

// Inspect only the adjacent declaration. Re-scanning the entire preceding
// command for each assignment would make large verified captures quadratic.
func commandDeclarationBefore(command string, start int) bool {
	end := start
	for end > 0 && strings.ContainsRune(" \t\r\n\f", rune(command[end-1])) {
		end--
	}
	if end == start {
		return false
	}
	for _, keyword := range []string{"const", "let", "var"} {
		begin := end - len(keyword)
		if begin < 0 || command[begin:end] != keyword {
			continue
		}
		if begin == 0 {
			return true
		}
		previous := command[begin-1]
		return !(previous >= 'a' && previous <= 'z' || previous >= 'A' && previous <= 'Z' || previous >= '0' && previous <= '9' || previous == '_' || previous == '$')
	}
	return false
}

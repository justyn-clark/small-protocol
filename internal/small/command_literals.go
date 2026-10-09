package small

import "regexp"

var capturedHTTPLiteralPattern = regexp.MustCompile("(?i)`http://[^`\r\n]*`")

const commandPortIdentifier = `[A-Za-z_$][A-Za-z0-9_$]*`

// This is a source-literal grammar, not evaluation of arbitrary expressions.
// A port may be an identifier, a member ending in .port, or address().port.
var capturedLoopbackPortPattern = regexp.MustCompile(`^(?i:http)://(localhost|127\.0\.0\.1|0\.0\.0\.0|\[::1\]):\$\{(` + commandPortIdentifier + `(?:\.` + commandPortIdentifier + `)*(?:\.address\(\))?\.port|` + commandPortIdentifier + `)\}([/?#].*)?$`)

// commandHTTPLiteralsForLint is used only for URL checks on display fields
// bound to exact command proof. Original commands remain intact for secret
// checks, storage, hashes and every authored artifact field.
func commandHTTPLiteralsForLint(command string) string {
	return capturedHTTPLiteralPattern.ReplaceAllStringFunc(command, func(literal string) string {
		raw := literal[1 : len(literal)-1]
		if isAllowedLocalhostHTTP(raw) {
			return "'" + raw + "'"
		}
		parts := capturedLoopbackPortPattern.FindStringSubmatch(raw)
		if parts == nil {
			return literal
		}
		normalized := "http://" + parts[1] + ":0" + parts[3]
		if !isAllowedLocalhostHTTP(normalized) {
			return literal
		}
		return "'" + normalized + "'"
	})
}

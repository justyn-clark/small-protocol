package small

import (
	"strings"
	"testing"
)

func TestCapturedCommandEnvironmentAssignments(t *testing.T) {
	config := `JSON.parse(fs.readFileSync(process.env.HOME + "/.config/example/config.json", "utf8")).user.token`
	credential := "sk-" + strings.Repeat("synthetic", 4)
	tests := []struct {
		name, command string
		secret        bool
	}{
		{"direct environment", `const token = process.env.EXAMPLE_TOKEN;`, false},
		{"regex environment search", `perl -0pi -e 's|const token = process\.env\.EXAMPLE_TOKEN;|replacement|g' file`, false},
		{"regex environment fallback", `const token = process\.env\.EXAMPLE_TOKEN \|\| "fixture-value";`, true},
		{"escaped namespace", `const token = process.\u0065nv.EXAMPLE_TOKEN;`, true},
		{"let environment", `let password = process.env.EXAMPLE_PASSWORD;`, false},
		{"config file", `const token = ` + config + `;`, false},
		{"single quoted config", `const token = JSON.parse(fs.readFileSync(process.env.HOME + '/config.json', 'utf-8')).user.token;`, false},
		{"shell quoted source", `node -e 'const token = process.env.EXAMPLE_TOKEN;'`, false},
		{"literal fallback", `const token = process.env.EXAMPLE_TOKEN || "fixture-value";`, true},
		{"nullish fallback", `const token = process.env.EXAMPLE_TOKEN ?? "fixture-value";`, true},
		{"config fallback", `const token = ` + config + ` || "fixture-value";`, true},
		{"literal", `const token = "fixture-value";`, true},
		{"later assignment", `const token = process.env.EXAMPLE_TOKEN; password=fixture-value`, true},
		{"earlier assignment", `password=fixture-value; const token = process.env.EXAMPLE_TOKEN;`, true},
		{"later credential", `const token = process.env.EXAMPLE_TOKEN; echo ` + credential, true},
		{"credential in config path", `const token = JSON.parse(fs.readFileSync(process.env.HOME + "/` + credential + `", "utf8")).user.token;`, true},
		{"arbitrary call", `const token = getToken(process.env.EXAMPLE_TOKEN);`, true},
		{"literal config path", `const token = JSON.parse(fs.readFileSync("/config.json", "utf8")).token;`, true},
		{"extra config argument", `const token = JSON.parse(fs.readFileSync(process.env.HOME + "/config.json", "utf8", "fixture-value")).token;`, true},
		{"template path", "const token = JSON.parse(fs.readFileSync(process.env.HOME + `/config.json`, 'utf8')).token;", true},
		{"escaped path", `const token = JSON.parse(fs.readFileSync(process.env.HOME + "/config\u002ejson", "utf8")).token;`, true},
		{"escaped shell quotes", `node -e "const token = \"fixture-value\";"`, true},
		{"escaped JavaScript literal", `const token = "fixture\"value";`, true},
		{"unfinished environment", `const token = process.env.EXAMPLE_TOKEN...`, true},
		{"unfinished statement", `const token = process.env.EXAMPLE_TOKEN`, true},
		{"shell expression", `token=process.env.EXAMPLE_TOKEN;`, true},
		{"property assignment", `object.token = process.env.EXAMPLE_TOKEN;`, true},
		{"different namespace", `const token = process.envish.EXAMPLE_TOKEN;`, true},
		{"keyword suffix", `notconst token = process.env.EXAMPLE_TOKEN;`, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := len(commandSecretViolations("capture", map[string]any{"command_summary": test.command})) > 0
			if got != test.secret {
				t.Fatalf("potential secret = %t, want %t", got, test.secret)
			}
		})
	}
}

func TestCapturedCommandManyEnvironmentReferences(t *testing.T) {
	command := strings.Repeat("const token = process.env.EXAMPLE_TOKEN;\n", 3000)
	if len(commandSecretViolations("capture", map[string]any{"command_summary": command})) != 0 {
		t.Fatal("repeated references rejected")
	}
	if !capturedCommandHasSecret(command + " password=fixture-value") {
		t.Fatal("late literal hidden after repeated references")
	}
}

func TestEnvironmentAssignmentRetainsFullCommandPolicy(t *testing.T) {
	safe := `const token = process.env.EXAMPLE_TOKEN;`
	for _, suffix := range []string{"", " password=fixture-value", " http://external.example/path"} {
		command := safe + " # " + strings.Repeat("x", 240) + suffix
		base, record := proofFixture(t, command, true)
		violations := CheckCommandRecord(base, "capture", record)
		if suffix == "" && len(violations) != 0 {
			t.Fatalf("environment reference rejected: %v", violations)
		}
		if suffix != "" && len(violations) == 0 {
			t.Fatal("violation after display cap was hidden")
		}
		if missing := CheckCommandRecord(t.TempDir(), "capture", record); !proofMessagesContain(missing, "proof unavailable") {
			t.Fatal("environment reference bypassed required command proof")
		}
	}
}

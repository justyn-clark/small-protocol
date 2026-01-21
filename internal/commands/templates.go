package commands

import "github.com/justyn-clark/small-protocol/internal/small"

var (
	intentTemplate = `small_version: "` + small.ProtocolVersion + `"
owner: "human"
intent: "Define project intent"
scope:
  include: []
  exclude: []
success_criteria: []
`

	constraintsTemplate = `small_version: "` + small.ProtocolVersion + `"
owner: "human"
constraints:
  - id: "no-secrets"
    rule: "No secrets in .small artifacts"
    severity: "error"
  - id: "no-prod-writes"
    rule: "No writes to production by default"
    severity: "error"
`

	planTemplate = `small_version: "` + small.ProtocolVersion + `"
owner: "agent"
tasks:
  - id: "task-1"
    title: "Initial task"
    steps: []
    acceptance: []
`

	progressTemplate = `small_version: "` + small.ProtocolVersion + `"
owner: "agent"
entries: []
`
)

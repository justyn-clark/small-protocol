package commands

var (
	intentTemplate = `small_version: "0.1"
owner: "human"
project_name: "Project Name"
goals:
  - "Adopt SMALL protocol for agent-legible project continuity"
  - "Define clear project intent and scope"
  - "Establish verifiable progress tracking"
context:
  purpose: "Enable agents to understand, execute, and resume work across sessions"
  scope: "Project continuity and state management"
`

	constraintsTemplate = `small_version: "0.1"
owner: "human"
technical_constraints:
  - "All artifacts must conform to SMALL v0.1 schemas"
  - "Progress entries must include verifiable evidence"
security_constraints:
  - "No secrets in any artifact"
  - "No sensitive tokens or credentials"
other:
  protocol_constraints:
    - "Handoff is the only resume entrypoint"
    - "Plan is disposable; progress is append-only"
`

	planTemplate = `small_version: "0.1"
owner: "agent"
generated_at: ""
tasks:
  - id: "task-1"
    description: "Define project intent and constraints"
    status: "pending"
    dependencies: []
  - id: "task-2"
    description: "Validate SMALL artifacts against schemas"
    status: "pending"
    dependencies: ["task-1"]
  - id: "task-3"
    description: "Generate handoff for agent resume"
    status: "pending"
    dependencies: ["task-2"]
`

	progressTemplate = `small_version: "0.1"
owner: "agent"
entries: []
`

	handoffTemplate = `small_version: "0.1"
generated_at: ""
current_plan:
  generated_at: ""
  tasks: []
recent_progress: []
next_steps: []
`
)

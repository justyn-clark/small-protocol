package commands

var (
	intentTemplate = `small_version: "0.1"
owner: "human"
project_name: "Example Web Application"
goals:
  - "Build a responsive web application with user authentication"
  - "Implement REST API with proper error handling"
  - "Add comprehensive test coverage (minimum 80%)"
context:
  target_audience: "Developers and end users"
  timeline: "6 weeks"
  tech_stack: ["TypeScript", "React", "Node.js"]
`

	constraintsTemplate = `small_version: "0.1"
owner: "human"
technical_constraints:
  - "Must use TypeScript for all code"
  - "No external dependencies without security audit"
  - "Must support Node.js 18+ and modern browsers"
business_constraints:
  - "Must comply with GDPR requirements"
  - "Budget limit: $5000 for third-party services"
security_constraints:
  - "No secrets in code or version control"
  - "All API endpoints must require authentication"
  - "User data must be encrypted at rest"
`

	planTemplate = `small_version: "0.1"
owner: "agent"
generated_at: "2024-01-15T10:30:00Z"
tasks:
  - id: "task-1"
    description: "Set up project structure and development environment"
    status: "completed"
    dependencies: []
  - id: "task-2"
    description: "Implement user authentication system"
    status: "in_progress"
    dependencies: ["task-1"]
  - id: "task-3"
    description: "Build REST API endpoints"
    status: "pending"
    dependencies: ["task-2"]
  - id: "task-4"
    description: "Write unit and integration tests"
    status: "pending"
    dependencies: ["task-3"]
`

	progressTemplate = `small_version: "0.1"
owner: "agent"
entries:
  - timestamp: "2024-01-15T09:00:00Z"
    task_id: "task-1"
    status: "completed"
    evidence: "Project structure created with TypeScript configuration"
    command: "npm init -y && npm install typescript @types/node"
    commit: "a1b2c3d4e5f6"
  - timestamp: "2024-01-15T10:00:00Z"
    task_id: "task-2"
    status: "in_progress"
    evidence: "Authentication routes scaffolded"
    verification: "Routes respond with 401 for unauthenticated requests"
    link: "https://github.com/example/project/pull/1"
  - timestamp: "2024-01-15T10:30:00Z"
    task_id: "task-2"
    status: "in_progress"
    evidence: "JWT token generation implemented"
    test: "Authentication test suite passes (12/12 tests)"
    commit: "b2c3d4e5f6a7"
`

	handoffTemplate = `small_version: "0.1"
generated_at: "2024-01-15T10:30:00Z"
current_plan:
  generated_at: "2024-01-15T10:30:00Z"
  tasks:
    - id: "task-1"
      description: "Set up project structure and development environment"
      status: "completed"
      dependencies: []
    - id: "task-2"
      description: "Implement user authentication system"
      status: "in_progress"
      dependencies: ["task-1"]
    - id: "task-3"
      description: "Build REST API endpoints"
      status: "pending"
      dependencies: ["task-2"]
    - id: "task-4"
      description: "Write unit and integration tests"
      status: "pending"
      dependencies: ["task-3"]
recent_progress:
  - timestamp: "2024-01-15T09:00:00Z"
    task_id: "task-1"
    status: "completed"
    evidence: "Project structure created with TypeScript configuration"
    command: "npm init -y && npm install typescript @types/node"
    commit: "a1b2c3d4e5f6"
  - timestamp: "2024-01-15T10:00:00Z"
    task_id: "task-2"
    status: "in_progress"
    evidence: "Authentication routes scaffolded"
    verification: "Routes respond with 401 for unauthenticated requests"
    link: "https://github.com/example/project/pull/1"
  - timestamp: "2024-01-15T10:30:00Z"
    task_id: "task-2"
    status: "in_progress"
    evidence: "JWT token generation implemented"
    test: "Authentication test suite passes (12/12 tests)"
    commit: "b2c3d4e5f6a7"
next_steps:
  - "Complete JWT token validation middleware"
  - "Add password hashing with bcrypt"
  - "Implement user registration endpoint"
`
)

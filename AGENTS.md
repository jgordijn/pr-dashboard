<!-- OPENSPEC:START -->
# OpenSpec Instructions

These instructions are for AI assistants working in this project.

Always open `@/openspec/AGENTS.md` when the request:
- Mentions planning or proposals (words like proposal, spec, change, plan)
- Introduces new capabilities, breaking changes, architecture shifts, or big performance/security work
- Sounds ambiguous and you need the authoritative spec before coding

Use `@/openspec/AGENTS.md` to learn:
- How to create and apply change proposals
- Spec format and conventions
- Project structure and guidelines

Keep this managed block so 'openspec update' can refresh the instructions.

<!-- OPENSPEC:END -->

## Development Commands

### Running Tests
Go is not in the system PATH. Use Docker to run tests:
```bash
docker run --rm -v "$(pwd)":/app -w /app golang:1.23 go test ./...
```

### Building
Use Docker to build (requires `-buildvcs=false` since Docker can't access git metadata):
```bash
docker run --rm -v "$(pwd)":/app -w /app golang:1.23 go build -buildvcs=false -o /tmp/pr-dashboard ./cmd/pr-dashboard
```

### Static Analysis
```bash
docker run --rm -v "$(pwd)":/app -w /app golang:1.23 go vet ./...
```

### Git Operations
- No remote is configured - push not available
- GPG signing is required by default; use `-c commit.gpgsign=false` to commit:
  ```bash
  git -c commit.gpgsign=false commit -m "message"
  ```
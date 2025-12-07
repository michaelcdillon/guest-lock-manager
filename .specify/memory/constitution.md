<!--
SYNC IMPACT REPORT
==================
Version Change: N/A → 1.0.0 (Initial ratification)

Added Principles:
  - I. Code Quality First (DRY, readable, maintainable)
  - II. YAGNI & Simplicity (lightweight, avoid over-engineering)
  - III. Continuous Refactoring (maintain quality through iteration)
  - IV. Performance Excellence (performant Go backend)
  - V. Documentation Standards (comprehensive project docs)

Added Sections:
  - Technical Stack & Constraints (Go RESTful API requirements)
  - Development Workflow (review process, quality gates)

Templates Status:
  ✅ plan-template.md - Compatible (Constitution Check section aligns)
  ✅ spec-template.md - Compatible (requirements/success criteria align)
  ✅ tasks-template.md - Compatible (phase structure supports principles)

Follow-up TODOs: None

Commit: docs: ratify constitution v1.0.0 (initial project governance)
-->

# Guest Lock Manager Constitution

## Core Principles

### I. Code Quality First

All code MUST be written to staff-level engineering standards:

- **DRY (Don't Repeat Yourself)**: Extract shared logic into reusable functions, 
  packages, or modules. Duplication indicates a missing abstraction.
- **Readable Code**: Code MUST be self-documenting through clear naming conventions.
  Function and variable names describe intent without requiring comments to explain
  *what* the code does.
- **Meaningful Comments**: Comments explain *why* decisions were made, not *what*
  the code does. Document non-obvious business logic, edge cases, and architectural
  trade-offs.
- **Maintainability**: Code MUST be structured for future maintainers. Prefer explicit
  over clever. Complex logic requires accompanying documentation.

**Rationale**: Staff-level code outlives its authors. Investment in quality now
prevents technical debt accumulation and enables rapid, confident changes later.

### II. YAGNI & Simplicity

You Aren't Gonna Need It. Build only what is required:

- **No Speculative Features**: Do not implement functionality "just in case."
  Every feature MUST have a concrete, current requirement.
- **Lightweight Code**: Prefer simple solutions. Fewer lines of code means fewer
  bugs, faster compilation, and easier understanding.
- **Minimal Dependencies**: Each external dependency MUST be justified. Prefer
  standard library solutions when reasonable.
- **Avoid Premature Abstraction**: Create abstractions only when patterns emerge
  from concrete implementations, not in anticipation of future needs.

**Rationale**: Over-engineering is the silent killer of projects. Unused code
still requires maintenance, testing, and mental overhead.

### III. Continuous Refactoring

Refactor often to maintain code health:

- **Boy Scout Rule**: Leave code cleaner than you found it. Every PR should improve
  surrounding code quality when touching a file.
- **Refactor in Small Batches**: Prefer frequent, small refactors over large,
  risky rewrites. Each change MUST be independently verifiable.
- **Technical Debt Visibility**: When taking on debt, document it with `// TODO:`
  comments including context and estimated remediation approach.
- **Safe Refactoring**: Refactors MUST NOT change behavior. Ensure adequate test
  coverage before restructuring.

**Rationale**: Codebases degrade without active maintenance. Regular refactoring
keeps velocity high and prevents "legacy code" syndrome.

### IV. Performance Excellence

The Go backend MUST be lightweight and performant:

- **Efficient Algorithms**: Choose appropriate data structures and algorithms.
  O(n²) operations on unbounded inputs are not acceptable.
- **Resource Awareness**: Minimize memory allocations in hot paths. Reuse buffers.
  Avoid unnecessary goroutine creation.
- **Benchmark Critical Paths**: Performance-sensitive code MUST have benchmarks.
  Regressions MUST be caught before merge.
- **Latency Targets**: API endpoints SHOULD respond in <100ms p95 for standard
  operations. Deviations require documented justification.

**Rationale**: A RESTful API serves many clients. Performance issues multiply
across all consumers and degrade user experience.

### V. Documentation Standards

Comprehensive documentation enables team effectiveness:

- **README Requirements**: Project root MUST contain a README with: purpose,
  quick start instructions, architecture overview, and development setup.
- **API Documentation**: All REST endpoints MUST be documented with request/response
  schemas, error codes, and example usage.
- **Code-Level Docs**: Public functions and packages MUST have GoDoc comments
  explaining purpose and usage.
- **Decision Records**: Significant architectural decisions SHOULD be captured
  in ADRs (Architecture Decision Records) under `docs/adr/`.

**Rationale**: Documentation is force multiplication. Time invested in docs
pays dividends across all team members and future contributors.

## Technical Stack & Constraints

**Primary Language**: Go (latest stable release)
**API Style**: RESTful HTTP/JSON
**Project Type**: Backend API service

### Go-Specific Requirements

- Follow [Effective Go](https://go.dev/doc/effective_go) conventions
- Use `gofmt` and `go vet` on all code
- Prefer explicit error handling over panic
- Use context propagation for cancellation and timeouts
- Structure packages by domain, not by technical layer

### API Design Principles

- Resource-oriented URLs (`/guests/{id}/locks`, not `/getGuestLocks`)
- Appropriate HTTP methods (GET, POST, PUT, DELETE, PATCH)
- Consistent error response format with meaningful messages
- Idempotent operations where semantically appropriate
- Pagination for list endpoints

### Testing Requirements

- Table-driven tests for Go functions
- Integration tests for API endpoints
- Test coverage as a quality signal, not a target to game

## Development Workflow

### Code Review Standards

- All changes require PR review before merge
- Reviewers verify constitution compliance
- Performance-impacting changes require benchmark comparison
- Documentation changes accompany feature changes

### Quality Gates

- `go build ./...` MUST succeed
- `go test ./...` MUST pass
- `go vet ./...` MUST report no issues
- Linter configured in CI MUST pass

### Commit Hygiene

- Atomic commits (one logical change per commit)
- Descriptive commit messages following conventional commits format
- Reference issue/ticket numbers where applicable

## Governance

This constitution is the authoritative source for project standards:

- **Supremacy**: Constitution principles supersede personal preferences or
  external conventions when in conflict.
- **Amendment Process**: Changes require documented justification, team review,
  and version increment per semantic versioning rules.
- **Compliance Verification**: Code reviews MUST verify adherence. Violations
  require either code changes or constitution amendment.
- **Living Document**: Update this constitution as the project evolves. Outdated
  governance is worse than no governance.

**Version**: 1.0.0 | **Ratified**: 2025-12-07 | **Last Amended**: 2025-12-07

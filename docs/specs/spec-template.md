# {Project Name} ({abbreviation}) spec

<!-- Instructions: Replace all {placeholder} values. Delete this comment block when done. -->

| Field                | Value        |
| -------------------- | ------------ |
| **Date**             | {YYYY-MM-DD} |
| **Author**           | {name}       |
| **Status**           | Draft        |
| **Version**          | 0.1          |
| **Last reviewed**    | {YYYY-MM-DD} |
| **Last reviewed by** | {name}       |

## Section index

<!--
  Write a one-sentence description of what each section covers.
  This helps readers jump to the right section and understand scope at a glance.

  Example from nd:
  - **Problem statement:** Defines the asset management problem that nd solves for coding agent users.
  - **Goals:** Lists the measurable outcomes nd aims to achieve.
-->

- **Problem statement:** {One sentence describing what this section covers.}
- **Goals:** {One sentence.}
- **Non-goals:** {One sentence.}
- **Assumptions:** {One sentence.}
- **Functional requirements:** {One sentence.}
- **Non-functional requirements:** {One sentence.}
- **User stories:** {One sentence.}
- **Technical design:** {One sentence listing the subsections included.}
- **Boundaries:** {One sentence.}
- **Success criteria:** {One sentence.}
- **Open questions:** {One sentence.}
- **Changelog:** Tracks document revisions.

## Problem statement

<!--
  Describe the problem this project solves in 1-3 paragraphs.
  - Who experiences the problem? (target user)
  - What is the current workaround and why is it inadequate?
  - What pain does this cause, and at what scale?

  Example from nd:
  "Developers who use coding agents like Claude Code accumulate large libraries of assets...
   Managing these assets is manual and fragile. Developers create symlinks by hand,
   maintain ad-hoc shell scripts, and lose track of what's deployed where."
-->

{Describe the problem here.}

## Goals

<!--
  List 3-8 measurable outcomes this project aims to achieve.
  Each goal should start with "A {user role} can..." to keep goals user-centric and verifiable.
  Number them for easy reference.

  Optional: Add a "Progressive complexity" section if your tool has layered learning curves.

  Example from nd:
  1. A developer can deploy any combination of assets to any supported coding agent's
     configuration directory through a single command or interactive selection.
  2. A developer can detect and repair broken, drifted, or stale asset deployments
     without manual investigation.
-->

1. {Goal 1}
2. {Goal 2}
3. {Goal 3}

## Non-goals

<!--
  List what this project will NOT do. For each non-goal:
  - Bold the name
  - Explain WHY it's excluded and what scope creep it prevents
  This protects against feature creep and sets clear expectations.

  Example from nd:
  - **Editing asset content.** nd deploys and organizes assets but never modifies their
    content. Content authoring belongs in the user's editor or AI agent. Adding editing
    would blur nd's role and create conflict with version-controlled source files.
-->

- **{Non-goal name}.** {Explanation of why excluded and what scope creep it prevents.}
- **{Non-goal name}.** {Explanation.}

## Assumptions

<!--
  State conditions believed true but not verified that the spec depends on.
  Use a table with Status column: Confirmed, Unconfirmed, or Design decision.
  Number assumptions for cross-referencing in other sections.

  Example from nd:
  | A3 | Symlinks are reliable on macOS for this use case. | Confirmed |
  | A8 | All nd data is stored under ~/.config/nd/. | Design decision |
-->

| #  | Assumption | Status      |
| -- | ---------- | ----------- |
| A1 | {Assumption text.} | Unconfirmed |
| A2 | {Assumption text.} | Confirmed   |

## Functional requirements

<!--
  Specify the project's behaviors using MoSCoW prioritization:
  - Must Have: Required for the minimum viable product.
  - Should Have: Important but the product works without them.
  - Could Have: Nice-to-have, deliver if time allows.
  - Won't Have (this time): Explicitly deferred with rationale and reconsider triggers.

  Tag each requirement with a stable ID ([FR-XXX]). IDs are never reused.
  Order requirements by implementation dependency within each tier.
  State the prioritization constraint (quality, deadline, or budget).

  Keep Must-Have ratio at or below 60% of total FRs.

  Example from nd:
  - **[FR-009]** The user can deploy a single asset to a coding agent's configuration
    directory by creating a symlink.
  - **[FR-037]** Support for coding agents other than Claude Code. Deferred because:
    v1 focuses on Claude Code. Reconsider when: v1 is stable and user demand is validated.
-->

This section specifies {project name}'s behaviors tagged by MoSCoW priority. Requirements are ordered by implementation dependency within each tier. The prioritization constraint is {quality | deadline | budget}.

### Must have

- **[FR-001]** {Requirement description.}
- **[FR-002]** {Requirement description.}

### Should have

- **[FR-0XX]** {Requirement description.}

### Could have

- **[FR-0XX]** {Requirement description.}

### Won't have (this time)

<!--
  For each deferred item, include:
  - "Deferred because:" with the rationale
  - "Reconsider when:" with the trigger condition
-->

- **[FR-0XX]** {Feature name.} Deferred because: {rationale}. Reconsider when: {trigger condition}.

**MoSCoW distribution:** Must: {N}, Should: {N}, Could: {N}, Won't: {N}. Must-Have ratio: {N}/{total} = {%}%. Within the 60% ceiling.

## Non-functional requirements

<!--
  Define measurable quality attributes for how the system operates (not what it does).
  Tag each with [NFR-XXX]. Cover relevant categories:
  - Performance (startup time, operation latency, throughput)
  - Error reporting and graceful degradation
  - Maintainability (code standards, documentation, project layout)
  - Distribution and installation
  - Test coverage targets
  - Security (input validation, safe parsing, path traversal prevention)
  - Concurrency and data integrity (atomic writes, file locking)
  - Logging and debugging support
  - Exit codes and machine-readable output

  Example from nd:
  - **[NFR-001]** Startup performance: The TUI must render the initial menu within
    500ms of invocation on a machine with a local asset source of 500+ assets.
  - **[NFR-010]** Atomic writes: All state file writes must use atomic file operations:
    write to a temporary file, call fsync, then rename to the target path.
-->

This section defines quality attributes for {project name}. These are measurable constraints on how the system operates, not what it does.

- **[NFR-001]** {Category}: {Measurable constraint.}
- **[NFR-002]** {Category}: {Measurable constraint.}

## User stories

<!--
  Describe key workflows from the user's perspective.
  Tag each with [US-XXX]. Each story includes:
  - A one-line "As a {role}, I want to {action} so that {benefit}" statement
  - Acceptance criteria (bulleted, specific, testable)
  - Related requirements (FR-XXX cross-references)
  - Priority note (if the story maps to Could/Won't Have FRs)

  Example from nd:
  **US-001: Deploy skills to a new project.**
  As a developer starting a new project, I want to deploy my preferred set of skills
  to the project's .claude/ directory so that Claude Code has my custom capabilities.

  - Acceptance criteria: I can run a single command to deploy multiple skills as symlinks.
    The tool confirms which assets were deployed and reports any errors.
  - Related requirements: FR-009, FR-010, FR-011.
-->

**US-001: {Story title.}**
As a {role}, I want to {action} so that {benefit}.

- Acceptance criteria: {Specific, testable conditions.}
- Related requirements: {FR-XXX, FR-XXX.}

**US-002: {Story title.}**
As a {role}, I want to {action} so that {benefit}.

- Acceptance criteria: {Specific, testable conditions.}
- Related requirements: {FR-XXX, FR-XXX.}

## Technical design

<!--
  Capture high-level architecture decisions and component structure.
  Describe what communicates with what, not low-level implementation details.
  Include only the subsections relevant to your project. Common subsections:

  Required:
  - Component overview
  - Data flow
  - Key technology choices

  As needed:
  - Configuration hierarchy (if multi-layered config)
  - Project structure (directory layout)
  - CLI command reference (if CLI tool)
  - Data/state schema (if persistent state)
  - API reference (if service/library)
  - Domain-specific mappings (deployment mapping, data model, etc.)
  - Error behavior (how errors are handled across scenarios)
-->

This section captures high-level architecture decisions and component structure. It describes what communicates with what, not implementation details.

### Component overview

<!--
  List 3-7 major components with a short paragraph each.
  Describe responsibility, inputs, outputs, and key behaviors.

  Example from nd:
  1. **Source manager.** Handles registration, discovery, and syncing of asset sources.
     Scans local directories and cloned repos for assets using convention-based directory
     layout. Reads optional nd-source.yaml manifests. Maintains an index of all known
     assets across all registered sources.
-->

{Project name} has {N} major components:

1. **{Component name}.** {Responsibility, inputs, outputs, key behaviors.}
2. **{Component name}.** {Responsibility, inputs, outputs, key behaviors.}

### Data flow

<!--
  Show how data moves through the system. Use an ASCII diagram.

  Example from nd:
  Asset sources (local dirs, cloned repos)
          │
          ▼
    Source manager (discover + index)
          │
          ▼
    Deploy engine (symlink create/remove/sync)
          │
          ▼
    Agent config directories (~/.claude/, .claude/)
-->

```text
{ASCII diagram showing data flow between components}
```

{Brief narrative explaining the flow.}

### Key technology choices

<!--
  Table of technology decisions with rationale.

  Example from nd:
  | Go 1.23+  | Single binary distribution, fast startup, strong CLI ecosystem. |
  | Cobra     | Standard Go CLI framework. Subcommands, flags, help generation. |
-->

| Choice     | Rationale |
| ---------- | --------- |
| {Tech}     | {Why this choice over alternatives.} |
| {Tech}     | {Why this choice.} |

### Configuration hierarchy

<!--
  Include this section if your project has multi-layered configuration.
  List resolution order from lowest to highest priority.

  Example from nd:
  1. Built-in defaults (opinionated, ships with nd).
  2. Global config (~/.config/nd/config.yaml).
  3. Project config (.nd/config.yaml in current working directory).
  4. CLI flags (highest priority, overrides everything).
-->

Configuration resolves in this order (later overrides earlier):

1. {Lowest priority source.}
2. {Next source.}
3. {Highest priority source.}

### Project structure

<!--
  Include this section to show the directory layout of the project.
  Use a tree diagram.
-->

```text
{project}/
├── {file/dir}
└── {file/dir}
```

### CLI command reference

<!--
  Include this section if your project is a CLI tool.
  Derive the command tree from functional requirements.
  Include global flags and exit codes.

  Example from nd:
  | `nd deploy <asset>` | Deploy one or more assets | FR-009, FR-010 |
-->

**Command tree:**

| Command | Description | FR |
| ------- | ----------- | -- |
| `{cmd}` | {Description} | {FR-XXX} |

**Global flags:**

| Flag | Description | FR |
| ---- | ----------- | -- |
| `{--flag}` | {Description} | {FR-XXX} |

**Exit codes:**

| Code | Meaning |
| ---- | ------- |
| 0    | Success |
| 1    | General error |

### API reference

<!--
  Include this section if your project exposes an API (REST, GraphQL, library, etc.).
  Document endpoints, methods, request/response schemas, and error codes.
-->

### Data/state schema

<!--
  Include this section if your project persists state to disk or database.
  Define schema fields, types, and annotated examples.
  Document write safety and concurrency strategy.

  Example from nd:
  | `source_id` | string | Identifier of the registered source |
  | `scope`     | string | `global` or `project` |

  Write safety: All writes use atomic file operations (write-to-temp-then-rename).
  Concurrency: Advisory file locking on state files.
-->

**Schema fields:**

| Field | Type | Description |
| ----- | ---- | ----------- |
| `{field}` | {type} | {Description} |

**Annotated example:**

```yaml
# {description of the example}
{field}: {value}
```

**Write safety:** {Describe atomic write strategy.}

**Concurrency:** {Describe locking strategy.}

### Error behavior

<!--
  Define how the project handles error scenarios. Include:
  - Config validation failures
  - Partial failures in bulk operations
  - Permission errors
  - Network/external service failures
  - Data corruption recovery

  The guiding principle from nd: "Never crash silently, always provide actionable
  guidance, and prefer continuing over aborting when partial progress is useful."

  Example from nd:
  **Partial failure in bulk operations:**
  - nd uses fail-open behavior: when one asset fails, nd continues with remaining.
  - After completion, outputs a summary: "Deployed 47/50 assets. 3 failed:"
  - In CLI mode, exits with code 2 (partial failure).
-->

**{Error scenario}:**

- {How the system behaves.}
- {What message the user sees.}
- {Recovery path.}

## Boundaries

<!--
  Define behavior tiers for AI agents implementing this spec.
  Three tiers:
  - Always: Invariants that must never be violated.
  - Ask-first: Actions that require user confirmation.
  - Never: Hard prohibitions.

  Example from nd:
  Always: Always validate that a source directory exists before attempting discovery.
  Ask-first: Ask before removing assets not managed by this tool.
  Never: Never modify the content of source files.
-->

This section defines behavior tiers for AI agents implementing this spec.

### Always

- Always {invariant behavior.}
- Always {invariant behavior.}

### Ask-first

- Ask before {action requiring confirmation.}
- Ask before {action requiring confirmation.}

### Never

- Never {hard prohibition.}
- Never {hard prohibition.}

## Success criteria

<!--
  Define how to determine whether the project succeeded.
  Each criterion should be:
  - Specific and measurable
  - Tied to a verification method
  - Mapped to requirement tiers (core vs. extended)

  Example from nd:
  1. A user with 500+ assets can deploy, remove, and sync without errors.
     Verified by: end-to-end test with a 500-asset source directory.
  2. A user can complete first-time setup and deploy their first asset within
     5 minutes. Verified by: timed walkthrough with a new user.
-->

**Core success criteria** (verifiable with Must-Have requirements):

1. {Criterion.} Verified by: {verification method.}
2. {Criterion.} Verified by: {verification method.}

**Extended success criteria** (require Should-Have or Could-Have requirements):

3. {Criterion.} Verified by: {verification method.}

## Open questions

<!--
  Track unresolved decisions. Use a table with stable IDs (never reuse Q numbers).
  Categorize by impact:
  - Blocking: Must be resolved before implementation can proceed.
  - Non-blocking: Can be resolved during or after implementation.
  - Assumption-dependent: Resolution depends on validating an assumption.

  Move resolved questions to a separate "Resolved questions" table with the resolution.

  Example from nd:
  | Q1 | What is the full schema for nd-source.yaml? | Non-blocking | Can be iterated after v1. |
-->

Question IDs are stable and not reused across revisions.

### Open questions

| #  | Question | Category | Impact |
| -- | -------- | -------- | ------ |
| Q1 | {Question text.} | {Blocking / Non-blocking / Assumption-dependent} | {Impact description.} |

### Resolved questions

| #  | Question | Resolution |
| -- | -------- | ---------- |
| Q{N} | ~~{Original question text.}~~ | {How it was resolved.} |

## Changelog

<!--
  Track document revisions. Each row captures what changed and why.
  Use the spec version number, not a separate document version.

  Example from nd:
  | 0.2 | 2026-03-14 | Larah | Audit revision: added commands as asset type; added asset
    deployment mapping table; clarified symlink direction in FR-009. |
-->

| Version | Date       | Author | Changes |
| ------- | ---------- | ------ | ------- |
| 0.1     | {YYYY-MM-DD} | {name} | Initial draft. |

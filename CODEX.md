# Codex Instructions for GoTravel

Codex, read this before touching the repository. This project is intentionally small. Do not inflate it into a platform, framework, daemon, or architectural wildlife reserve.

## Required Reading Order

Before making changes, read:

1. `AUTHORITATIVE_SPECIFICATION.md`
2. `COMMANDS.md`
3. `AGENTS.md`
4. `README.md`
5. `CHANGES.md`
6. `TRACKER_SIGNALS.md` if tracker signal parsing is involved

## Prime Directive

Preserve simplicity.

The goal is a clear CLI tool that imports tracker CSV data into SQLite and exports/reports it. Prefer explicit, boring code over clever abstractions.

## Do Not Do These Without Explicit Approval

- Add a web server.
- Add a UI framework.
- Add background workers.
- Add Docker-only workflows.
- Replace SQLite.
- Add Postgres, Redis, Elasticsearch, Kafka, or similar infrastructure.
- Add an ORM unless specifically requested.
- Introduce broad auto-detection of CSV formats.
- Change command syntax silently.
- Change stored schema silently.
- Rewrite the project because you dislike the current shape.

## Safe Work Types

Good Codex tasks:

- Add focused tests.
- Fix import parsing bugs.
- Fix export bugs.
- Add one importer at a time.
- Add one exporter at a time.
- Improve error messages.
- Update docs to match behaviour.
- Refactor within a package without changing behaviour.

Risky Codex tasks:

- Multi-package rewrites.
- New architecture.
- Dependency swaps.
- Routing engine integration.
- Schema migrations.

Risky tasks require a short implementation plan before code changes.

## Testing Expectations

For any behaviour change:

```bash
go test ./...
```

If tests cannot be run, state why clearly.

When changing import/export behaviour, add or update fixtures under:

```text
tests/fixtures/
```

## Documentation Expectations

Update the following when relevant:

```text
COMMANDS.md       CLI changes
README.md         user-facing usage changes
CHANGES.md        version/change log
AGENTS.md         agent workflow changes
TRACKER_SIGNALS.md tracker signal interpretation changes
```

## Commit Style

Use small commits with direct messages:

```text
Add Gator corrupt row fixture
Fix export overwrite guard
Document import force behaviour
```

Avoid vague commit messages like:

```text
Update stuff
Refactor
Various fixes
```

## Output Discipline

When reporting back, include:

- What changed.
- What files changed.
- Tests run.
- Any risks or follow-up work.

Do not claim tests passed if they were not run.

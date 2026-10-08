# Repository Guidelines

## Fork Scope

- This is a customized downstream fork of Aerion, a desktop email client (Go, Wails v2, Svelte 5, TypeScript).
- Prioritize the owner's requirements; upstream contribution restrictions do not govern local work.
- Keep customizations focused to ease future upstream merges.

## Layout Notes

- `frontend/wailsjs/` holds generated Wails bindings; regenerate with `make generate` after binding changes.
- `extensions/` (contacts, calendar) contains both backend and frontend code.
- Go tests live beside implementation files.

## Commands

Toolchain: Go 1.25, Node.js 24 (matches CI), Wails v2 CLI. Linux needs GTK 3 and WebKitGTK 4.1 development libraries.

- `cd frontend && npm ci`: install locked frontend dependencies.
- `cd frontend && npm run build`: build frontend assets. Run before Go checks that include the root package if `frontend/dist/` is missing; rebuild when validating changed assets.
- `make dev`: launch the desktop app with hot reload.
- `make build`: production build in `build/bin/`.
- `make generate`: regenerate Wails TypeScript bindings.
- `make test`: run `go test ./...`.
- `make lint`: golangci-lint v2 and frontend ESLint.
- `make fmt`: format Go code.
- `cd frontend && npm run check`: Svelte/TypeScript checks.

## Code Style

- Go: log with zerolog, comment exported functions, and prefer guard clauses and switches over long `else if` chains.
- Frontend: use Svelte 5 runes (`$state`, `$derived`), keep components under 500 lines, and preserve keyboard accessibility. ESLint enforces formatting.
- Localization: this fork targets English. Update English locale files only, and leave non-English locale files out of searches and reviews unless the task concerns localization. Preserve the existing i18n structure. See `docs/LANGUAGE.md` when changing localized strings.

## Testing

- Start with checks relevant to the change.
- Run `make test` when a change affects multiple Go packages or shared behavior, or when focused checks leave unresolved risks.
- Use table-driven Go tests for critical paths and edge cases.
- There is no frontend test script. For frontend changes, run `make lint` and `npm run check`, then verify the UI manually.

## Documentation

- When a change affects user-visible behavior, commands, configuration, or developer workflows, update the relevant documentation in the same task.
- Correct existing guidance instead of adding duplicate instructions.
- Internal changes that do not alter documented behavior need no documentation update.

## Milestones

Before marking a milestone done:

1. Review the diff for correctness and regressions.
2. Have a read-only subagent check the milestone's acceptance criteria. If no subagent or review tool is available, do the equivalent review directly and report the missing independent check.
3. Fix gaps that affect correctness or stated requirements; treat style-only findings as optional.
4. Simplify the changed code without changing behavior.
5. Commit locally, staging only files changed for the task and preserving unrelated user changes.

Never push.

## Commits and PRs

- Use descriptive prefixes for downstream commits: `feat:`, `fix:`, `docs:`.
- In PRs, describe behavior changes, link relevant issues, and report validation.
- See `CONTRIBUTING.md` for setup and workflow details. Consult upstream's own guidelines when submitting upstream.

## Planning Files

- `PLAN.md`: guides new features. Create it when needed with scope, milestones, acceptance criteria, and validation steps.
- `TASKS.md`: current status of the tasks in progress.
- `ISSUES.md`: issues found that may need fixing in this or a future session.
- Keep task-specific progress out of `AGENTS.md`.
- **Required:** remove completed plan, task, and issue items once they no longer need to be referenced, so these three files stay short and current.

## Compaction

When compacting, preserve the current milestone and sub-step, files modified so far, failing checks with their exact commands, and discoveries not yet written to the plan.

## Security

- Copy `.env.example` to `.env` for OAuth testing; `.env.local` overrides it.
- Keep credentials out of commits and logs.
- Follow `SECURITY.md` for vulnerability reporting.

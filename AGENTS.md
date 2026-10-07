# Repository Guidelines

## Fork Scope

This is a customized downstream fork of Aerion. Prioritize the owner's requirements; upstream contribution restrictions do not govern local work. Keep customizations focused to ease future upstream merges.

## Project Structure & Module Organization

Aerion is a desktop email client built with Go, Wails v2, Svelte 5, and TypeScript. Root Go files initialize the application; `app/` contains application methods, and `internal/` contains mail protocols, synchronization, storage, and security packages. `frontend/src/` holds UI code, themes, and assets; `frontend/wailsjs/` contains generated bindings. `extensions/` holds contacts and calendar modules with backend and frontend code. Go tests live beside implementation files. Packaging lives in `build/`, branding in `brand/`, and documentation in `docs/`.

## Build, Test, and Development Commands

Use Go 1.25, Node.js 24 (matching CI), and the Wails v2 CLI. Linux development requires GTK 3 and WebKitGTK 4.1 development libraries.

- `go mod download`: download Go dependencies.
- `cd frontend && npm ci`: install locked frontend dependencies when needed.
- `cd frontend && npm run build`: build frontend assets. Run before Go checks that include the root package if `frontend/dist/` is missing; rebuild when validating changed assets.
- `make dev`: launch the desktop app with hot reload.
- `make build`: build the production application in `build/bin/`.
- `make generate`: regenerate Wails TypeScript bindings after binding changes.
- `make test`: run `go test ./...`.
- `make lint`: run golangci-lint v2 and frontend ESLint.
- `cd frontend && npm run check`: run Svelte/TypeScript checks.

## Coding Style & Naming Conventions

Format Go with `gofmt` or `make fmt`; use tabs, explicit error handling, exported-function comments, and zerolog logging. Prefer guard clauses and switches over long `else if` chains.

Frontend ESLint requires two-space indentation, single quotes, and no semicolons. Use PascalCase component names, descriptive camelCase variables, and Svelte 5 runes such as `$state` and `$derived`. Keep components focused and under 500 lines; preserve keyboard accessibility.

This fork targets English. For custom UI changes, update English locale files only. Exclude non-English locale files from routine searches and reviews unless the task concerns localization. Preserve the existing i18n structure to ease upstream merges.

## Documentation

When a change affects user-visible behavior, commands, configuration, or developer workflows, check the relevant documentation and update it in the same task so it remains accurate. Documentation updates are not required for internal changes that do not alter documented behavior or workflows. Prefer correcting existing guidance over adding duplicate instructions.

## Testing Guidelines

Use Go's `testing` package, `*_test.go` files, and `TestXxx` functions. Cover critical paths and edge cases with table-driven tests where appropriate. Start with checks relevant to the change. Run `make test` for changes affecting multiple Go packages or shared behavior, or when focused checks leave unresolved risks. No numeric coverage threshold or frontend test script is configured; use frontend lint and type checks plus manual UI verification for frontend changes.

- Before marking a milestone done, review the diff for correctness and regressions, have a read-only subagent check its acceptance criteria, then simplify changed code without changing behavior. Use available review and simplify tools or skills; if unavailable, perform the equivalent review directly and report any missing independent check. Fix gaps that affect correctness or stated requirements; treat style-only findings as optional.
- Commit locally at milestone boundaries, staging only files changed for the task and preserving unrelated user changes. Never push.
- When compacting, preserve: the current milestone and sub-step, files modified so far, failing checks with their exact commands, and discoveries not yet written to the plan.

## Project Planning, Task Management, and Issue Tracking

- Use `PLAN.md` to guide new features; create it when needed with scope, milestones, acceptance criteria, and validation steps. Keep task-specific progress out of `AGENTS.md`.
- Use TASKS.md to track the current status of the tasks you are working on.
- Add to ISSUES.md whenever you find any issues that may need to be fixed in this session or a future session.

**Required:** ensure you keep PLAN.md, TASKS.md, and ISSUES.md concise and compact over time, and clean of outdated information. Remove completed plan, task, and issue items once they have been completed and do not need to be referenced later.

## Commit & Pull Request Guidelines

Inherited history uses release titles and short maintenance summaries. Use descriptive prefixes such as `feat:`, `fix:`, and `docs:` for downstream commits. In PRs, describe behavior changes, link relevant issues when available, and report validation.

Use `CONTRIBUTING.md` for setup and contribution workflow details, and `docs/LANGUAGE.md` when changing localized strings or translations. Consult upstream’s own guidelines when submitting upstream.

## Security & Configuration

Copy `.env.example` to `.env` for OAuth testing; `.env.local` overrides it. Keep credentials out of commits and logs. Follow `SECURITY.md` for vulnerability reporting.

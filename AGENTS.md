# Repository Guidelines

## Fork Scope

- This is a customized downstream fork of Aerion, a desktop email client (Go, Wails v2, Svelte 5, TypeScript).
- Prioritize the owner's requirements; upstream contribution restrictions do not govern local work.
- Keep customizations focused to ease future upstream merges.

## Layout Notes

- `frontend/wailsjs/` holds generated Wails bindings; regenerate with `make generate` after binding changes, then revert unrelated churn in `frontend/wailsjs/runtime/*`. Stop any running Aerion first (`dev.sh stop`): the single-instance lock makes the generator exit without writing anything.
- `extensions/` (contacts, calendar) contains both backend and frontend code. Check extension, core API, and kit changes against `docs/EXT_RULES.md` before committing.
- `docs/EXTENSIONS.md` (about 170 KB), `docs/CASAT2.md`, and `docs/LANGUAGE.md` are long: grep them or read by section, never whole.

## Commands

Toolchain: Go 1.26, Node.js 24 (matches CI), Wails v2 CLI. Linux needs GTK 3 and WebKitGTK 4.1 development libraries.

- `cd frontend && npm ci`: install locked frontend dependencies.
- `cd frontend && npm run build`: build frontend assets. Run before Go checks that include the root package if `frontend/dist/` is missing; rebuild when validating changed assets.
- `make dev`: launch the desktop app with hot reload. It blocks; for agent-driven UI checks use `.claude/skills/run-aerion/dev.sh` instead.
- `make build`: production build in `build/bin/`.
- `make generate`: regenerate Wails TypeScript bindings.
- `make test`, `make fmt`: Go tests and formatting.
- `make lint`: golangci-lint v2 and frontend ESLint.
- `cd frontend && npm run check`: Svelte/TypeScript checks.
- OAuth testing: copy `.env.example` to `.env`; `.env.local` overrides it.

## Code Style

- Go: log with zerolog, and prefer guard clauses and switches over long `else if` chains.
- Frontend: use Svelte 5 runes (`$state`, `$derived`), keep components under 500 lines, and preserve keyboard accessibility.
- Localization: this fork targets English. Update English locale files only, and leave non-English locale files out of searches and reviews unless the task concerns localization. Preserve the existing i18n structure. See `docs/LANGUAGE.md` when changing localized strings.

## Testing

- Start with checks relevant to the change.
- Run `make test` when a change affects multiple Go packages or shared behavior, or when focused checks leave unresolved risks.
- Use table-driven Go tests for critical paths and edge cases.
- There is no frontend test script. For frontend changes, run `npm run lint` and `npm run check` in `frontend/`, then verify the UI with the `run-aerion` skill. If no display is available (for example, in a cloud session), say so in the PLAN.md Progress entry and in your report.

## Documentation

- When a change affects user-visible behavior, commands, configuration, or developer workflows, update the relevant documentation in the same task.
- Correct existing guidance instead of adding duplicate instructions.

## Milestones

These steps apply to `PLAN.md` milestones. For a small independent fix, run the relevant checks, review the diff, and commit; skip steps 2 and 4.

Before marking a milestone done:

1. Review the diff for correctness and regressions.
2. Have a read-only subagent check the milestone's acceptance criteria. If no subagent or review tool is available, do the equivalent review directly and report the missing independent check.
3. Fix gaps that affect correctness or stated requirements; treat style-only findings as optional.
4. Simplify the changed code without changing behavior.
5. Commit and push, staging only files changed for the task and preserving unrelated user changes.

## Commits and PRs

- Use descriptive prefixes for downstream commits: `feat:`, `fix:`, `docs:`.
- Work on a branch per feature or audit pass, not on `main`, and land it as described in Landing a Branch.
- One commit per milestone or independent fix. Fold corrections to a commit that has not been pushed into it (`git commit --amend`) instead of adding a new commit.
- Commit `PLAN.md` and `ISSUES.md` updates with the change they describe, not separately.
- See `CONTRIBUTING.md` for setup and workflow details. Consult upstream's own guidelines when submitting upstream.

## Landing a Branch

When all of a feature's milestones are done or an audit pass is complete, land the branch without being asked. Checks must pass and nothing may be uncommitted.

1. If `main` has commits the branch lacks, rebase the branch onto `main` and rerun the relevant checks. On conflicts, run `git rebase --abort`, stop, and report the conflicting files.
2. Run `git switch main` and `git merge --squash <branch>`, then commit once, listing each change the branch made in the commit body.
3. Run `git branch -D <branch>` only after `git diff main <branch>` prints nothing.

## Planning Files

- `PLAN.md`: guides new features. Create it when needed with Scope; Milestones, each with acceptance criteria and validation steps; and these living `##` sections: Progress (one checkbox per milestone), Surprises & Discoveries, Decision Log, and Outcomes & Retrospective. Track progress in Progress, not a separate file.
- `ISSUES.md`: issues found that may need fixing in this or a future session.
- Keep task-specific progress out of `AGENTS.md`.
- **Required:** remove completed plan and issue items once they no longer need to be referenced, so these files stay short and current.

## Compaction

When compacting, preserve the current milestone and sub-step, files modified so far, failing checks with their exact commands, and discoveries not yet written to the plan.

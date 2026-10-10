# Contributing to This Fork

This repository is a personal, customized downstream fork of [Aerion](https://github.com/hkdb/aerion). The owner's requirements guide development. Upstream contribution restrictions apply only to submissions to upstream.

## Workflow

- Keep changes focused and explain their purpose so future upstream merges remain manageable.
- Use this fork's [issues](https://github.com/clabs-public-forks/aerion/issues) for bugs, ideas, and questions. An issue or upstream approval is not required before local work.
- Base changes on the intended downstream branch. For PRs, use the fork's default branch unless the task specifies another target.
- Describe the behavior change, relevant issues, checks performed, and known limitations. Include screenshots for visible UI changes when useful.
- Use descriptive commit messages with prefixes such as `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, and `chore:`.

## Development Setup

Use Go 1.26, Node.js 24 (matching CI), and the Wails v2 CLI. On Linux, install GTK 3 and WebKitGTK 4.1 development libraries.

```bash
git clone https://github.com/clabs-public-forks/aerion.git
cd aerion
go mod download
cd frontend
npm ci
npm run build
cd ..
make dev
```

For OAuth testing, copy `.env.example` to `.env` and configure your own credentials. `.env.local` overrides `.env`. Do not commit credentials or include private mail data in logs or issues.

`make dev` uses the same data directory (`~/.local/share/aerion` on Linux) and OS keyring as a native build. Refresh tokens belong to the OAuth client that issued them, so after the client ID in `.env`/`.env.local` changes, or for accounts added by a build with other credentials, sync fails with `token refresh failed: unauthorized_client`. Re-authorize the account once in the dev app (edit it in Settings → Accounts, then **Re-authorize**) to fix it.

`make dev` always renders the Nord (Dark) theme, whatever theme is saved in Settings. Theme choices are still saved and apply in production builds.

## Code and Validation

Prefer small changes, explicit Go error handling, and guard clauses. Format Go with `gofmt`, document exported functions, and use zerolog for logging. Frontend code uses two-space indentation, single quotes, no semicolons, PascalCase component names, and Svelte 5 runes. Keep components focused and under 500 lines. Preserve keyboard accessibility, resource efficiency, and user privacy.

- `make fmt`: format Go code.
- `make lint`: run golangci-lint v2 and frontend ESLint.
- `cd frontend && npm run check`: check Svelte and TypeScript.
- `make test`: run Go tests; build the frontend first because Go embeds `frontend/dist/`.
- `make build`: produce the desktop application.
- `make generate`: regenerate bindings after changing the Wails API.

Start with checks relevant to the change. Add Go tests for meaningful behavior and edge cases, using table-driven tests where appropriate. No frontend test script or numeric coverage threshold is configured. Manually check affected UI flows and report checks that could not run.

## Translations and Security

Use [docs/LANGUAGE.md](docs/LANGUAGE.md) for translation implementation and validation. Report vulnerabilities according to [SECURITY.md](SECURITY.md).

## Upstream Contributions and License

For changes intended for upstream, consult upstream's current contribution rules separately. Preserve upstream attribution and the repository's Apache License 2.0 terms.

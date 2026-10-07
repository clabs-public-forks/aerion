# Issues

## Open

### Flatpak Docker build environment is out of date

`build/flatpak/Dockerfile` (used by `build-flatpak-docker.sh`) no longer matches the project toolchain:

- Go 1.23.0 → should be Go 1.25 (`go.mod`, CI).
- Node.js 20.11.0 → should be Node.js 24 (CI).
- `wails@latest` → pin to the `go.mod` version (`v2.16.0`) for reproducible builds.
- GNOME Platform/Sdk 47 → manifest (`io.github.hkdb.Aerion-dev.yml`) and other Flatpak scripts use 50.
- Downloads are hardcoded to `linux-amd64`/`linux-x64`, so the image won't build on arm64.

Fix: update the versions and keep them in sync with CI, then run `build/flatpak/build-flatpak-docker.sh` to verify the build.

### `build/wails-output.py` wrapper gaps

The OAuth-redacting wrapper used by `make build`/`dev`/`dev-race`/`build-linux`/`build-windows-installer` (commit 7b08649):

- No signal handling: Ctrl+C in `make dev` raises `KeyboardInterrupt` in Python (traceback, exit without `process.wait()`), closing the pipe while Wails is still shutting down, so Wails' cleanup output is lost and it can hit SIGPIPE. Ignore SIGINT in the wrapper, keep draining output, and forward SIGTERM to the child.
- `bufsize=0` makes `readline()` read one byte per syscall; use the default buffering (lines are still flushed per line).
- Secrets are replaced in env order; sort by length (longest first) so one credential that contains another is not partially leaked.
- New Python 3 build dependency is undocumented (`docs/BUILD.md`, `CONTRIBUTING.md`) and `python3` may not resolve on Windows (`release.yml` runs `make build-windows-installer` on `windows-latest`); verify or default `PYTHON` per platform.
- `stderr=subprocess.STDOUT` merges Wails/Go errors into stdout, so `make build 2>err.log` or CI steps that capture stderr miss them. Filter stderr separately and write it back to stderr.
- Line-based redaction holds partial output (progress text ending in `\r` or without a newline) until the next newline, and the LDFlags row loses its original indentation. Preserve leading whitespace and flush partial lines.
- Design concern: filtering output treats the symptom. Credentials are passed as `-ldflags` (`Makefile:78`), and Wails/Go may print them quoted or escaped (e.g. `-v 2`, error output), which an exact match misses. Consider generating a gitignored Go source file with the values before the build so they never appear on a command line, which would also remove the Python wrapper.

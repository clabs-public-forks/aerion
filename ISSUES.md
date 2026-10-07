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

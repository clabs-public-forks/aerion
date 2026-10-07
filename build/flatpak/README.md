# Flatpak Packaging for Aerion

This directory contains files for building and distributing Aerion as a Flatpak.

## Files

- `io.github.hkdb.Aerion-dev.yml` - Dev manifest (packages pre-built host binary, no compilation)
- `io.github.hkdb.Aerion.metainfo.xml` - AppStream metadata
- `build-flatpak.sh` - Dev build script (uses `-dev.yml`)
- `build-local.sh` - From-source local build script (uses flathub manifest)
- `test-build.sh` - CI build test script (Docker container)
- `build-flatpak-docker.sh`, `Dockerfile` - Docker-based build
- `flathub/` - Flathub submission files (from-source manifests + vendored deps)

## Prerequisites

Install flatpak-builder:

```bash
# Fedora
sudo dnf install flatpak-builder

# Ubuntu/Debian
sudo apt install flatpak-builder

# Arch
sudo pacman -S flatpak-builder
```

Add Flathub repository (if not already added):

```bash
flatpak remote-add --if-not-exists --user flathub https://flathub.org/repo/flathub.flatpakrepo
```

Install required runtimes and SDKs:

```bash
flatpak install --user flathub org.gnome.Platform//50 org.gnome.Sdk//50
flatpak install --user flathub org.freedesktop.Sdk.Extension.golang//25.08
flatpak install --user flathub org.freedesktop.Sdk.Extension.node24//25.08
```

The local-checkout build also needs the host build prerequisites listed in [the project build guide](../../docs/BUILD.md), including WebKitGTK 4.1 development files.

## Building Locally

### Build and install this checkout

This builds the Linux binary on the host, packages it with the local dev manifest, installs it into your user Flatpak scope, and writes a bundle to `build/bin/Aerion-dev.flatpak`.

```bash
make flatpak-install
```

Run the installed app with:

```bash
flatpak run io.github.hkdb.Aerion
```

`make flatpak-dev` runs the same local-checkout build path.

### Build the Flathub source manifest

`make flatpak` uses the Flathub manifest and builds its configured source inside the Flatpak sandbox. The script also installs the result into your user scope and creates a bundle under `build/bin/`.

```bash
make flatpak
```

For a direct script invocation, use:

```bash
./build/flatpak/build-local.sh
```

## Validation

Before submitting to Flathub, validate the metainfo file:

```bash
# Install appstream-util
sudo dnf install libappstream-glib  # Fedora
sudo apt install appstream-util      # Ubuntu/Debian

# Validate (from project root)
appstream-util validate build/flatpak/io.github.hkdb.Aerion.metainfo.xml
```

Validate the desktop file:

```bash
desktop-file-validate build/linux/aerion.desktop
```

## Submitting to Flathub

See [`flathub/README.md`](flathub/README.md) for complete Flathub submission instructions.

## Additional Resources

- [Flatpak Documentation](https://docs.flatpak.org/)
- [Flathub Submission Guide](https://github.com/flathub/flathub/wiki/App-Submission)
- [AppStream Guidelines](https://www.freedesktop.org/software/appstream/docs/)
- [Flatpak Builder Manifest](https://docs.flatpak.org/en/latest/flatpak-builder-command-reference.html)

## Advantages Over AppImage

- WebKit provided by GNOME runtime (no bundling needed)
- Works on ALL Linux distros consistently
- Sandboxing is properly implemented
- Automatic updates via Flatpak
- Centralized distribution through Flathub
- Better integration with desktop environments
- Shared runtime = smaller download size

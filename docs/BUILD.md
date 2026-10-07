# BUILD

Set .env file

### 🔨 Building from Source
---

**Flatpak (Recommended for Linux):**

```bash
# Install flatpak-builder
sudo dnf install flatpak-builder  # Fedora
sudo apt install flatpak-builder  # Ubuntu/Debian
sudo pacman -S flatpak-builder    # Arch

# Set Microsoft and Google oAuth creds
cp .env.example .env
# Fill in your own creds

# Build from this checkout and install for your user
make flatpak-install

# Run
flatpak run io.github.clabs_public_forks.Aerion
```

`make flatpak-install` installs into your user Flatpak scope and also writes a bundle under `build/bin/`. See [build/flatpak/README.md](../build/flatpak/README.md) for other build paths and Flathub submission details.

**Native Binary:**

```bash
# Install dependencies (Ubuntu/Debian)
sudo apt install build-essential libgtk-3-dev libwebkit2gtk-4.1-dev

# Set Microsoft and Google oAuth creds
cp .env.example .env
# Fill in your own creds

# Build
make build

# Run
./build/bin/aerion
```

The `make` build and dev targets write the OAuth credentials from `.env` / `.env.local` into the gitignored `internal/oauth2/credentials_gen.go` and compile it with the `aerion_creds` build tag, so the values never appear in build output. `make clean` removes the file.


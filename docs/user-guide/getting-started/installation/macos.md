---
sidebar_position: 2
pagination_next: getting-started/verify
---

# macOS

1. Download the latest `zip` file:
    - [amd64](https://github.com/hkdb/aerion/releases/latest/download/Aerion-darwin-amd64.zip)
    - [arm64](https://github.com/hkdb/aerion/releases/latest/download/Aerion-darwin-arm64.zip)
2. Extract the archive and open .dmg file
3. Move `Aerion.app` to your Applications folder
4. On first launch, you may see a security warning since the app is not signed with an Apple Developer certificate

## Bypassing Gatekeeper

If macOS blocks the app:

1. Right-click (or Control-click) on `Aerion.app`
2. Select "Open" from the context menu
3. Click "Open" in the dialog that appears

Alternatively, run this command:
```bash
xattr -cr /Applications/Aerion.app
```

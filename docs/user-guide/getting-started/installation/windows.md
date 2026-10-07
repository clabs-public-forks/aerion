---
sidebar_position: 3
---

# Windows 

## Installer

1. Download the latest installer:
    - [amd64](https://github.com/hkdb/aerion/releases/latest/download/Aerion-windows-setup-amd64.exe)
    - [arm64](https://github.com/hkdb/aerion/releases/latest/download/Aerion-windows-setup-arm64.exe)
2. Run the installer
3. Click on "More Info" in the SmartScreen dialog and then click "Run Anyway"
4. Aerion should now be launchable from your start menu


## Portable

1. Download the latest `.exe`:
    - [amd64](https://github.com/hkdb/aerion/releases/latest/download/Aerion-windows-amd64.exe)
    - [arm64](https://github.com/hkdb/aerion/releases/latest/download/Aerion-windows-arm64.exe)
2. Move executable to a safe place of choice, create a shortcut to put on your taskbar or start menu and then run the executable
3. Click on "More Info" in the SmartScreen dialog and then click "Run Anyway"
4. Aerion will launch

## WebView2 Runtime

Aerion requires the Microsoft Edge WebView2 Runtime. On Windows 11 and recent Windows 10 updates, this is pre-installed. If needed, download it from [Microsoft](https://developer.microsoft.com/en-us/microsoft-edge/webview2/).

## Community Maintained Packages

### Chocolatey

https://community.chocolatey.org/packages/aerion

This is an unofficial package maintained by [@Infiniti151](https://github.com/Infiniti151). For any issues, please submit a Github issue directly at the [Infiniti151/aerion-chocolatey](https://github.com/Infiniti151/aerion-chocolatey) project.

Standard:

This pulls down the native setup utility, handles programmatic configuration, and hooks seamlessly into Windows Apps & Features.

```bash
choco install aerion
```

Portable:

This downloads the raw, uncompressed execution binary directly into your localized Chocolatey application sandbox. It keeps your Windows registry completely untouched and instantly configures a global CLI command-line shortcut shim so aerion can be called from any terminal session immediately.

```bash
choco install aerion.portable
```

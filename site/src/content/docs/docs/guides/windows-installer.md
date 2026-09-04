---
title: Install and update Freehand
description: Install, verify, update, or remove the Freehand Windows alpha.
---

Download Freehand only from the official
[GitHub Releases](https://github.com/tnware/freehand-stt/releases) page. Each
release provides:

- `freehand-windows-amd64-installer.exe` for a normal installation;
- `freehand-windows-amd64.exe` for portable use and in-app updates; and
- `SHA256SUMS` to verify both downloads.

## Install

Use the installer for the normal first-install experience. It installs Freehand
for the current Windows user beneath `%LOCALAPPDATA%\Programs`, creates one
Start Menu shortcut, and does not require administrator access.

The installer checks the Windows version, processor architecture, and WebView2
Runtime. If WebView2 is missing, it runs Microsoft's bootstrapper. It does not
create a Desktop shortcut or automatically launch Freehand after installation.

:::caution[Unsigned alpha]
The current artifacts are not Authenticode-signed, so Windows may identify the
publisher as unknown. Verify that the download came from the official release
and compare its SHA-256 hash with `SHA256SUMS` before proceeding.
:::

To verify a download in PowerShell:

```powershell
Get-FileHash -Algorithm SHA256 .\freehand-windows-amd64-installer.exe
```

## Upgrade

Installing a newer package in place is the normal alpha upgrade path. Exit
Freehand from its tray menu first; the installer will not silently terminate a
running instance.

Automatic update checks are enabled by default and can be disabled under
**Settings → General**. When a newer release is available, Freehand opens the
Wails update window and asks before restarting into the verified bare
executable. The updater does not run the NSIS installer.

Because this is alpha software, deliberate downgrade testing is not blocked.

## Portable executable

The bare executable can run without the installer. It does not provide Start
Menu registration or an uninstaller. Keep it in a stable, user-writable
location if you want the in-app updater to replace it.

## Uninstall

Exit Freehand from the tray, then remove it through **Installed apps** or its
Start Menu uninstaller. Uninstall removes the installed executable, bundled
third-party notice, uninstaller, registration, and Start Menu shortcut.

Settings, interface preferences, and Windows Credential Manager entries are retained.
Freehand does not currently offer a destructive remove-all-data option.

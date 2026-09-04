---
title: Brand asset pipeline
description: Canonical Freehand artwork and deterministic platform asset generation.
---

Freehand keeps product artwork as vector source and commits deterministic
platform inputs so normal builds do not need a design application or an SVG
rasterizer installed. The pipeline distinguishes three visual jobs instead of
forcing one large tile into every surface:

- the detailed product mark for in-app identity;
- the rounded application tile for large application/package surfaces;
- the optically simplified, mark-only treatment for the Windows system tray.

## Source-to-consumer matrix

| Classification | Path | Consumer |
| --- | --- | --- |
| Authoritative source | `branding/freehand-mark.svg` | In-app `BrandMark`; Apple Icon Composer vector layer |
| Authoritative source | `branding/freehand-app-tile.svg` | Rounded surface composed with the detailed mark for platform icons |
| Authoritative source | `branding/freehand-system-mark.svg` | Windows tray light/dark families only |
| Derived presentation | `branding/freehand-readme-light.png`, `branding/freehand-readme-dark.png` | Theme-aware repository hero; never read by a build or runtime path |
| Reference only | Other `branding/*.png` files | Design review; never read by a build or runtime path |
| Generator | `build/scripts/brandassets` | Deterministic SVG-subset rasterization, PNG encoding, and ICO assembly |
| Generated platform input | `build/appicon.png` | Wails macOS/Linux/mobile generation and packaging |
| Generated platform input | `build/windows/icon.ico` | Windows executable resources, taskbar/application surfaces, NSIS installer and uninstaller |
| Generated runtime input | `build/windows/tray-light.ico` | Wails `SystemTray.SetIcon` in Windows light mode |
| Generated runtime input | `build/windows/tray-dark.ico` | Wails `SystemTray.SetDarkModeIcon` in Windows dark mode |
| Generated vector | `frontend/src/lib/assets/freehand-mark.svg` | Svelte `BrandMark` in the main header and About window through Vite's asset pipeline |
| Verified publication copy | `site/src/assets/freehand-mark.svg` | Starlight documentation header; byte-compared with the authoritative source in documentation CI |
| Verified publication copy | `site/public/favicon.svg` | Astro site favicon; byte-compared with the authoritative source in documentation CI |
| Generated vector | `build/appicon.icon/Assets/freehand_mark_vector.svg` | Apple Icon Composer source layer |
| Generated platform output | `build/darwin/icons.icns` | macOS bundle icon, produced by Wails from `build/appicon.png` |

The Windows ICO files contain 16, 20, 24, 32, 48, 64, 128, and 256 pixel PNG
frames. Wails selects the closest tray frame to Windows' current small-icon
metric, including DPI-scaled 20px and 24px shells. Every application frame uses
the same detailed mark, tile composition, and relative scale so Windows cannot
change the visual identity when it selects a different resolution. The tray is
mark-only and has separate light- and dark-theme color treatments. The in-app
mark uses a tighter vector viewBox so its artwork, rather than an invisible
application-icon canvas, determines its rendered size. The application tile
retains deliberate outer safe space for Windows surfaces that mask or frame it.

The native main, Settings, and About captions deliberately set Wails'
`DisableIcon`; they are not missing icon consumers. Their taskbar identity comes
from the executable resource, while the About content uses the generated vector
mark. The current release path is NSIS. Wails v3.0.0-beta.16's experimental
`tool msix` path synthesizes transparent placeholder visual assets and exposes no
project-asset input, so it is not treated as a shippable branded package until
that packaging path is replaced or upstream gains an explicit asset contract.

## Updating artwork

From the repository root:

```powershell
wails3 task common:update:brand-assets
wails3 task common:check:brand-assets
```

`common:generate:icons` first runs the same generator, then asks the pinned
Wails CLI to produce the macOS icon bundle. The generator implements only the
SVG primitives used by the three canonical sources (`rect`, rounded `rect`,
`circle`, grouping, fill, and stroke). Unsupported SVG content fails loudly so
a design-tool export cannot silently render incorrectly. The detailed glyph
exists in only `freehand-mark.svg`; the generator composes it with the tile,
preventing the in-app and application-icon presentations from drifting. CI
checks every generator-owned application, tray, Apple-layer, and frontend
output byte for byte. The macOS ICNS remains a pinned-Wails output of the generated
`build/appicon.png` rather than an output of this repository-owned renderer.

Do not edit generated PNG, ICO, or copied SVG artifacts. Make an intentional
change to a canonical SVG and regenerate them.

The README hero is curated presentation artwork rather than generator output.
Keep its light and dark compositions synchronized, use the exact canonical
mark and product language, and replace both variants together when the identity
changes.

## Native Windows acceptance

Cross-compilation and image decoding are not visual acceptance. On Windows 11,
inspect the tray and executable/package icons in light and dark modes at 100%,
125%, 150%, and 200% scale. Confirm the waveform and caret remain separate,
centered, and recognizable; the tray has no white application tile; theme
changes swap the tray treatment; and taskbar, Explorer, installer, uninstaller,
Installed apps, and Start Menu surfaces use an appropriate application frame.
Also confirm the in-app mark remains sharp in the main header and About window.

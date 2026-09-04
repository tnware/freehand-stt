# Freehand brand sources

Freehand is the user-facing product name. `freehand-stt` is the repository and
technical discovery name; it should not appear as the product name in ordinary
application chrome.

These SVG files are authoritative production artwork:

- [`freehand-mark.svg`](freehand-mark.svg) is the detailed waveform-to-caret
  product mark used inside the application and by Apple Icon Composer.
- [`freehand-app-tile.svg`](freehand-app-tile.svg) supplies the rounded surface
  onto which the generator composes the detailed mark for large application,
  package, and installer surfaces.
- [`freehand-system-mark.svg`](freehand-system-mark.svg) is a deliberately
  simplified and thickened, mark-only treatment for the Windows system tray.
  It is never substituted into the tiled application icon.

The Astro product site keeps byte-identical publication copies of
`freehand-mark.svg` under `site/src/assets/` and `site/public/`. Documentation
CI compares both copies with this authoritative source before building, so
drift fails visibly.

The paired `freehand-readme-light.png` and `freehand-readme-dark.png` files are
approved, derived repository-presentation artwork. The root README selects the
appropriate treatment for the viewer's color scheme. They are not logo sources
or runtime/build inputs; regenerate both intentionally whenever their canonical
mark, product language, typography, or composition changes.

Other PNG files in this directory are visual references only. New reference
boards belong here with an explicit, descriptive name; temporary exports do not
belong in the build tree.

## Product language

- Name: **Freehand**
- Description: **Speech to text, anywhere you type.**
- Repository: **freehand-stt**
- Executable: **freehand**

## Visual foundation

- Accent: cobalt `#2563EB`
- Application UI: Inter
- Technical values: IBM Plex Mono

The existing Fluent/Mica token system remains the application design system.

## Generated assets

Run `wails3 task common:update:brand-assets` from the repository root after
editing a production SVG. Run `wails3 task common:check:brand-assets` to verify
the committed outputs. CI performs the same deterministic drift check.

The generator and complete source-to-consumer matrix are documented in
[`site/src/content/docs/docs/development/brand-assets.md`](../site/src/content/docs/docs/development/brand-assets.md). Generated files must not be
edited directly.

The Freehand artwork was created for this project and carries no third-party
logo or icon attribution. Third-party software notices remain in
[`THIRD_PARTY_NOTICES.md`](../THIRD_PARTY_NOTICES.md).

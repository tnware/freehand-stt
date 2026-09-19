# Freehand site

The Freehand product site and documentation use Astro with Starlight. The
project was created with the official starter:

```powershell
npm create astro@latest -- --template starlight
```

Run commands from this directory:

```powershell
npm ci
npm run dev
npm run build
```

Use Node.js 22.19 or newer, matching the current Astro dependency floor.

The custom product landing page is `src/pages/index.astro`. Documentation lives
under `src/content/docs/docs/` and is served from `/docs/`.

The home and download pages share `src/components/ParticleBackground.astro`, a
decorative canvas effect behind the content. It uses the existing blue palette,
runs at up to 30 frames per second, caps particle density and canvas resolution,
and pauses when the tab is hidden. Reduced-motion preferences render a static
background; Windows contrast themes hide the effect. Keep this component out
of the documentation reading surface.

## Documentation authoring

Choose the page's job before choosing its formatting. Readers should be able to
identify the task, find the next action, and recognize success without reading
every paragraph. Use a short opening, descriptive headings, and one primary
home for each topic.

| Page type | Reading order |
| --- | --- |
| Start or overview | What the reader can do → meaningful choices → links to the task guides |
| Task guide | Prerequisites → steps → expected result → recovery → next action |
| Backend or model guide | What works and with which versions → setup → supported options → limits and related guides |
| Troubleshooting | Symptom → likely check → action → where to go if it persists |
| Contributor procedure | When to use it → commands or checks → expected evidence → safety constraints |

The sidebar follows installation, service setup, everyday tasks, and options/help.
Backend, model, technical, and contributor references are collapsed initially.
Keep detailed file transcription, speech playback, history, and workspace
instructions in their dedicated guides; `using-freehand.mdx` owns dictation and
links to the other workflows. `getting-started/index.mdx` should help a new user
complete one request, rather than repeat every workflow's reference material.

Use `.mdx` when a page benefits from Starlight's built-in components; plain `.md`
still suits short references using tables and callouts. MDX is included by Starlight;
there is no extra UI framework or dependency to install. Keep the same file
stem when converting a page so its published URL remains unchanged.

| Content | Supported presentation |
| --- | --- |
| Ordered setup procedure | `Steps` around a Markdown numbered list |
| Alternative workflows | `Tabs` / `TabItem`; keep required warnings outside tabs |
| Related guides or next actions | `LinkCard` inside `CardGrid` |
| Brief context, a helpful option, or a limitation | `:::note`, `:::tip`, or `:::caution` |
| Commands and settings | Fenced code with language, `title`, and selective line markers |
| Secondary explanation | Native `<details>` / `<summary>` |
| Defaults and compatibility | Markdown table |
| Keyboard chords | `<kbd>` elements |

### Components in MDX

Imports follow the frontmatter. Keep blank lines around Markdown nested in a
component. Steps and tabs should express a real sequence or choice, rather
than decorate ordinary paragraphs.

```mdx
---
title: Connect your speech service
description: Set up a speech endpoint and finish your first dictation.
---

import { Steps, Tabs, TabItem, CardGrid, LinkCard } from '@astrojs/starlight/components';

<Steps>

1. Enter the speech endpoint and exact model ID.
2. Choose **Save changes**, then **Test connection**.

</Steps>

<Tabs>
  <TabItem label="No cleanup">

Leave **Cleanup** off to deliver the speech model's transcript unchanged.

  </TabItem>
  <TabItem label="Custom cleanup">

Choose a separate chat endpoint and supply your own instructions.

  </TabItem>
</Tabs>

<CardGrid>
  <LinkCard title="Use Freehand" href="../using-freehand/" description="Explore capture and delivery." />
</CardGrid>
```

Use `syncKey` on tab groups only when they share the same labels and should
remember the same choice across pages. Keep headings and existing anchor IDs
stable. Use route-relative links in MDX components so local previews and the
GitHub Pages base path both work.

### Callouts and code blocks in Markdown or MDX

````md
:::tip[Compare both versions]
Enable session history before dictating to retain raw and cleaned text within
the history limits.
:::

```powershell title="Verify the installer"
Get-FileHash -Algorithm SHA256 .\freehand-windows-amd64-installer.exe
```

```text title="Settings → Transcription" {1}
Base URL: https://speech.example.com/v1
Model: your-speech-model
```

<details>
<summary>Why does the first request take longer?</summary>

The selected inference server may need to load its model before processing.

</details>
````

Keep callouts short and specific. Keep prerequisites, retention limits, and
actions needed to finish setup visible. Let Starlight provide responsive
layout, theme styling, and keyboard behavior; do not recreate its components
with custom CSS. Use `Badge` for compact status labels and `FileTree` for
actual directory structures when those help a page, not as decoration.

Avoid replacing a wall of paragraphs with a wall of callouts or cards. Use a
table when readers need to compare the same properties, not to contain an
entire essay in each cell. Keep explanatory prose that connects ideas. Place
shared limits outside tabs, use tabs only for alternatives, and reserve
`<details>` for information the reader can safely skip. Keep headings outside
hidden panels where possible so the page outline remains useful.

When splitting a page, retain its existing headings or explicit anchor IDs with
short links to the new primary home. Update incoming links to that home, and
check that the sidebar, page outline, and next-page links form a coherent path.

Working examples: `getting-started/index.mdx`, `guides/connect-a-server.mdx`,
`guides/post-processing.mdx`, `guides/windows-installer.mdx`,
`guides/using-freehand.mdx`, and `development/index.mdx` under
`src/content/docs/docs/`.

After authoring, run `npm run build` and check both local and GitHub Pages
base-path links and fragments, including preserved anchors. Preview a task
guide, a reference table, and any changed tabs at narrow and desktop widths in
both themes; check keyboard tab and disclosure controls. For syntax and available props, use the official
[Starlight component reference](https://starlight.astro.build/components/using-components/)
and [Markdown guide](https://starlight.astro.build/guides/authoring-content/).

## Custom-page metadata and responsive checks

The homepage and download page share `CustomPageMetadata.astro`. Canonical and
social URLs use `site-url.mjs`, the same `SITE_URL` configuration as Astro, and
remain production URLs during local preview. Keep the page title and description
in that component invocation. The dedicated share image in `public/social/freehand.jpg`
is an optimized 1200 × 630 JPEG. Keep its wordmark and tagline away from the edges
so modest platform cropping preserves them. When replacing it, update the image
URL, MIME type, dimensions, and alt text in the metadata component as needed.
This asset is independent of the repository README banners.

For changes to the custom header or mobile layouts, check both pages at 320 px,
390 px, the 1040 px navigation breakpoint, and desktop width. Verify viewport
scale stays at 1, the menu exposes every navigation destination, keyboard users
can open it and dismiss it with Escape, and icon-only links have accessible names.
Check docs tabs and supporting text in both themes when changing palette tokens.

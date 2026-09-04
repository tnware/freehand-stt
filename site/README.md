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

// Shared by Astro configuration and the custom pages' canonical/social metadata.
export const productionUrl = new URL(
  process.env.SITE_URL ?? 'https://tnware.github.io/freehand-stt/',
);
if (!productionUrl.pathname.endsWith('/')) productionUrl.pathname += '/';

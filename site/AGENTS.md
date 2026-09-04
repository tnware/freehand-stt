# Site instructions

- This is the Freehand product site and public documentation, built from the
  official Astro Starlight starter.
- Keep the custom landing page under `src/pages/` and documentation content
  under `src/content/docs/docs/`.
- Keep audiences explicit: product landing copy in the custom home page,
  task-oriented installation and usage in the user guide, advanced endpoint
  behavior in Reference, and repository internals in Contribute or Maintainer
  records. Do not mix build/release instructions into an end-user task page.
- Prefer Astro components and static HTML. Add a client framework only when a
  real interactive island requires it.
- Extend Starlight through supported configuration, custom CSS, or component
  overrides. Do not patch vendored packages.
- Preserve keyboard navigation, reduced-motion behavior, responsive layouts,
  readable contrast, and semantic landmarks.
- Run `npm run build` after editing site source or documentation.

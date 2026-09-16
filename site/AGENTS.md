# Site instructions

- This is the Freehand product site and public documentation, built from the
  official Astro Starlight starter.
- Keep the custom landing page under `src/pages/` and documentation content
  under `src/content/docs/docs/`.
- Keep audiences explicit: product landing copy in the custom home page,
  task-oriented installation and usage in the user guide, advanced endpoint
  behavior in Reference, and actionable contributor procedures in Contribute.
  Do not mix build/release instructions into an end-user task page.
- Write user documentation about the product as it works today. Lead with the
  task, prerequisites, steps, expected result, and recovery. Use current UI labels
  checked against source; do not copy issue acceptance criteria into the guide.
- Keep contributor pages focused on actionable procedures and safety rules.
  Record implementation decisions and validation results in issues or pull requests.
- Remove development chronology, rejected-design comparisons (such as "not a
  separate window"), migration internals, and commentary about layout fixes from
  user pages. Keep an upgrade note only when a user must act or understand a
  change to their data.
- Explain safety and compatibility through user consequences and actions. Keep
  important limits, data lifetimes, costs, permission restrictions, and versioned
  backend requirements. Keep implementation details out of user guides; consult
  source and tests instead of duplicating their structure in prose.
- Give each topic one primary home and link to it instead of accumulating repeated
  setup instructions or capability tables. Preserve public heading anchors when
  editing; check rendered internal links and fragments after structural changes.
- Prefer Astro components and static HTML. Add a client framework only when a
  real interactive island requires it.
- Extend Starlight through supported configuration, custom CSS, or component
  overrides. Do not patch vendored packages.
- Preserve keyboard navigation, reduced-motion behavior, responsive layouts,
  readable contrast, and semantic landmarks.
- Run `npm run build` after editing site source or documentation.

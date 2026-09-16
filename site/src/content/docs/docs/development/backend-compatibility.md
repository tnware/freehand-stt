---
title: Maintain backend compatibility
description: Keep the application catalog, public matrix, provider guides, and validation evidence aligned.
---

## Ownership and update path

`internal/compatibility` owns profile IDs, operation scope, availability, routes,
and implemented capabilities. The renderer receives that catalog through the
Settings DTO. The website consumes its generated JSON export; public availability
and the feature matrix must never be maintained as a second set of support flags.

When a contract changes:

1. Implement and validate the bounded request/response behavior in Go. Reject
   wrong-operation and unavailable profiles at the backend boundary.
2. Update the profile catalog and its capability rules. Keep advanced features
   unavailable until their model and server requirements are represented.
3. Run `go generate ./internal/compatibility` from the repository root. Commit
   `site/src/data/compatibility.generated.json` with the implementation.
4. Update editorial copy in `site/src/data/backends.ts` and the relevant backend
   guide under `site/src/content/docs/docs/backends/`. Every catalog profile must
   have a directory entry and a guide or a specific planned-contract anchor.
5. Run the affected Go fixtures and the site build. The Go catalog test rejects
   a stale export; site rendering rejects missing or extra directory entries.
6. Record validation evidence and limitations in the issue or pull request. Do not promote
   a source review or fixture result into a claim of native Windows or macOS interoperability.

Run `go test ./internal/compatibility` to check the app/site catalog boundary.
Local site-only builds consume the committed export and need no Go runtime.
The Pages workflow also runs the Go catalog check, including for site-only
changes, before publishing the site.

## Backend and model documentation

Keep server APIs under `docs/backends/` and dedicated model behavior under
`docs/models/`. The public `/backends/` directory compares server operations;
`/models/` explains the controls each dedicated profile adds in Freehand.
NeMo-Speech.cpp is a backend, Nemotron is a model, and Qwen3-ASR belongs under
model profiles alongside S1-mini.

`internal/modelprofile` remains the authority for dedicated profiles and their
backend intersections. Update the editorial summaries in `site/src/data/models.ts`
and the corresponding model guides when these contracts change. Do not present
Generic examples, such as Whisper or Kokoro, as additional dedicated profiles.
Link model guides to backend installation and backend guides to model controls.
Keep test reports and qualification history in issues or pull requests, not in
this guide or on product cards.

When moving a guide, update internal links and the explicit Starlight sidebar,
and preserve its old URL with a base-path-aware Astro redirect. Check the Models
and Backends navigation on desktop and mobile, cross-links, and redirects after
building the static site. No inference is needed for this review.

## Evidence to record in the issue or pull request

For a live setup, record the operation, Freehand revision, server release or
commit when known, model/voice identifier, response format or streaming dialect,
and observed outcome. Explicitly mark unknown versions. Do not publish private
URLs, credentials, transcripts, machine names, or personal file paths.

Separate these kinds of evidence:

- Client contract fixtures, including request fields, errors, truncation, and
  completion semantics.
- Tagged upstream source or documentation with the inspected version.
- Reported live behavior for a particular setup.
- Native interactive acceptance performed separately on Windows and macOS.

A model list is metadata and cannot be used as a capability proof.

## Adding a planned profile

Add a stable ID and only the relevant operation entries with availability off
and no implemented capabilities. Explain the concrete missing contract work.
Add a public directory entry and a specific guide anchor, then regenerate the
catalog. Keep the planned state consistent across Settings and the public site. Track scheduling and delivery in GitHub issues/PRs,
not in a separate public task checklist.

## Public-page review

Verify desktop and mobile navigation, active-page indication, keyboard focus,
small-screen matrix scrolling, base-path-aware links, provider guide anchors,
and canonical metadata. The directory tracks main-branch behavior; keep that
notice visible so it is not mistaken for a promise about an older release.

## Provider identity assets

Use the shared SVG collection and provenance manifest in `branding/providers/`.
Follow its README for source, license, and asset requirements. Provider identity
is independent of capability support; do not maintain another set of support
flags in presentation code. Check the affected app and site surfaces after an
asset change, including neutral fallbacks and the public icon credits.

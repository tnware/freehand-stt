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
6. Record validation evidence and limitations in the guide and PR. Do not promote
   a source review or fixture result into a claim of live Windows interoperability.

Run `go test ./internal/compatibility` to check the app/site catalog boundary.
Local site-only builds consume the committed export and need no Go runtime.
The Pages workflow also runs the Go catalog check, including for site-only
changes, before publishing the site.

## Evidence to record

For a live setup, record the operation, Freehand revision, server release or
commit when known, model/voice identifier, response format or streaming dialect,
and observed outcome. Explicitly mark unknown versions. Do not publish private
URLs, credentials, transcripts, machine names, or personal file paths.

Separate these kinds of evidence:

- Client contract fixtures, including request fields, errors, truncation, and
  completion semantics.
- Tagged upstream source or documentation with the inspected version.
- Reported live behavior for a particular setup.
- Native interactive acceptance performed on Windows.

Existing Speaches and llama.cpp reports have limited version information. Keep
that limitation visible until a more specific report replaces it. A model list
is metadata and cannot be used as a capability proof.

## Adding a planned profile

Add a stable ID and only the relevant operation entries with availability off
and no implemented capabilities. Explain the concrete missing contract work.
Add a public directory entry and a specific guide anchor, then regenerate the
catalog. Keep the planned state consistent across Settings, the public site,
and technical documentation. Track scheduling and delivery in GitHub issues/PRs,
not in a separate public task checklist.

## Public-page review

Verify desktop and mobile navigation, active-page indication, keyboard focus,
small-screen matrix scrolling, base-path-aware links, provider guide anchors,
and canonical metadata. The directory tracks main-branch behavior; keep that
notice visible so it is not mistaken for a promise about an older release.

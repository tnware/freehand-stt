---
title: "ADR 0014: Clean settings baseline"
description: Replace alpha persistence with a distinct, immutable SQLite lineage and no compatibility import.
---

Status: Accepted.

## Decision

Start a new settings lineage on Windows and macOS. `freehand.db` has a distinct
SQLite `application_id`; renaming an alpha database cannot make it compatible.
First launch creates safe defaults and an empty connection catalog. Users repeat
setup and enter API keys again. Do not read, import, convert, or delete alpha
`settings.db`, `settings.json`, their recovery files, or legacy native credentials.
Old installations and their data are not an upgrade source.

The sole embedded Goose/sqlc schema source is `internal/storage/schema/`, starting
with `00001_initial.sql`. Remove the alpha `internal/storage/migrations/` chain
rather than replaying it into a new database. The baseline directly describes
current task, connection, and model ownership; obsolete compatibility columns,
import markers, and historical task fields in remembered-model rows are not part
of the new contract. Goose's version table remains the sole schema-version
authority. Do not add another version counter or migration runner.

Published migrations in this new lineage are immutable. The generation guard
compares `schema/` against the target revision: an alpha-only base has no published
new-lineage migrations, while later changes, deletion, or renaming of a published
baseline fail. This is an explicit lineage transition, not a general exception to
immutability. Subsequent migrations are ordered, transactional, and forward-only;
Goose and sqlc consume the same directory and generated queries remain committed.

## Preserved boundaries

This supersedes ADR 0006's lineage/import policy and the alpha migration and
historical-field compatibility provisions in ADRs 0007–0010 and 0012. Their
product decisions and historical rationale remain intact. Keep modernc SQLite,
Goose, sqlc, typed storage adapters, coherent settings transactions, fail-closed
identity/history checks, backups, recovery, and immutable request snapshots.

Keys remain native-only in Windows Credential Manager or macOS Keychain; SQLite
stores opaque references, never secrets. The reset does not enumerate or clean up
legacy credentials. Window placement remains separate and transcript history
remains optional and memory-only. No inference runs as part of migration or CI.

## Validation

Use temporary Git repositories to prove the alpha-to-baseline transition and
published new-lineage immutability. Reject empty/invalid migration sets and
nontransactional migrations; check reproducible sqlc generation. Real temporary
SQLite fixtures must prove fresh initialization, alpha-file isolation, foreign
identity rejection (including renamed alpha databases), typed round trips,
recovery, and future forward upgrades. Native Windows and macOS acceptance remain
separate from fixtures and cross-compilation.

See the [storage guide](../../development/storage/) for the current contract and
[first-launch notice](../../getting-started/#first-launch) for user action.

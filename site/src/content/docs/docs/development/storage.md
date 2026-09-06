---
title: SQLite storage
description: Maintain the schema, generated queries, credential references, and recovery contract.
---

Freehand persists non-secret settings in SQLite. Follow
[ADR 0006](../../decisions/0006-sqlite-storage-contract/) and the repository's
`AGENTS.md` for every storage change. Saved connections use stable server records with explicit capability memberships. Reusable model
preferences and persistent history are separate features. Disposable window
placement remains in `window-state.json`, independent of settings recovery.

## Ownership and tools

| Concern | Owner |
| --- | --- |
| Defaults, compatibility rules, validation | `internal/config` |
| Save coordination, native rollback, captured request profiles | `internal/settings` |
| Database, backups, typed adapters, credential-reference lifecycle | `internal/storage` |
| Schema and version history | Embedded goose migrations in `internal/storage/migrations` |
| Application SQL | `internal/storage/queries`; sqlc output in `internal/storage/dbgen` |
| Actual API keys | Windows Credential Manager |

The initial dependency pins are `modernc.org/sqlite v1.58.0`, its required
`modernc.org/libc v1.75.6`, `goose/v3 v3.28.0`, and `sqlc v1.31.1`.
Runtime versions live in `go.mod`; the generation command pins sqlc in
`build/scripts/storage/main.go`. Respect the driver's libc version relationship
when upgrading. SQLite adds no SQLite-specific CGO dependency; the native audio
and Wails build requirements still apply.

The first migration creates four `STRICT` singleton settings tables, related
headers, opaque credential references, a credential cleanup queue, and an
initialization marker. Foreign keys and explicit deletion rules protect related
rows. Go validates complete domain settings before writing and after reading.
Generated rows and database handles never cross into Wails or domain services.

## Change the schema or a query

1. Add the next five-digit goose SQL migration, such as `00004_example.sql`.
   Never edit or delete a migration already on the target branch. Runtime startup
   runs forward only; migrations must remain transactional.
2. Add explicit, parameterized application queries under `queries/`. `sqlc.yaml`
   reads the same migration directory used by the executable.
3. Generate from the repository root:

   ```sh
   go run ./build/scripts/storage
   ```

4. Map generated rows in storage adapters, update domain validation as needed,
   and add file-backed upgrade and recovery tests. Commit SQL and generated Go
   together; never manually edit `dbgen`.
5. Run the contract check against the target branch and the affected tests:

   ```sh
   go run ./build/scripts/storage -check -base main
   go test ./internal/storage ./internal/settings ./build/scripts/storage
   ```

The equivalent Wails tasks are `storage:generate` and `storage:check`.
CI checks generated drift, new untracked generated files, immutable target-branch
migrations, and database import/query boundaries. Handwritten infrastructure SQL
is confined to `store.go` and `recovery.go`: connection settings, identity/version
inspection, integrity checks, and backup work. Application queries require sqlc.
A different driver, ORM, migration runner, or boundary requires a superseding ADR.

## Connection and durability policy

On Windows, the database is `%LOCALAPPDATA%\Freehand\settings.db`, with a
protected directory ACL for the current user and SYSTEM. SQLite sidecars and
backups inherit that ACL. A separate OS file lock serializes app ownership and
initialization. The connection pool has one open/idle connection. Every new
connection enables foreign keys, disables trusted schema, and configures a
1.5-second SQLite busy timeout and `synchronous=EXTRA`. The store verifies
rollback `journal_mode=DELETE`, foreign keys, and synchronization at startup.

Operations use a cancellable store-owned context and a ten-second budget.
Shutdown cancels database work before waiting for the store lock and closing its
handle. Native credential calls are synchronous OS operations. Writes are short
and serialized; no database transaction includes network or inference work.
Goose's own table is the sole migration authority. Unknown/newer or inconsistent
history, foreign database identity, integrity failures, and unsupported settings
block normal operation rather than triggering defaults.

## Import, upgrade, and recovery

With no database present, the bounded `config.LegacyReader` reads
`%APPDATA%\Freehand\settings.json`. A valid known document is imported into a
private temporary database; schema, typed settings, references to existing native
keys, and initialization state must complete before the file is published as
`settings.db`. Invalid or unknown fields block import. The source JSON is never
rewritten or removed. Interrupted initialization leaves no authoritative partial
database, so the next launch can retry. Unpublished temporary files are not
loaded as settings and can be removed while Freehand is closed.

Before upgrading an existing schema, SQLite's backup API creates a consistent,
synced copy under `backups/`; the newest three successful backups are retained.
A backup failure blocks the upgrade. Each migration and its goose version update
commit together. On an uncertain settings commit, saves and new inference
profiles pause until Retry validates and reloads the committed state.

Explicit Reset constructs a complete replacement, closes SQLite, archives the
old database and any journal/WAL sidecars in `settings-recovery-*`, and publishes
the replacement. Recovery archives preserve evidence and are not automatically
pruned. `Store.RestoreBackup` similarly validates a private copy, upgrades that
copy if needed, and replaces the current database only after it is usable. This
method is an internal native recovery boundary, not a renderer binding or an
arbitrary-path API. User-facing manual restoration instructions are in
[troubleshooting](../../guides/troubleshooting/#saved-settings-need-attention).

Backups contain configuration, not API keys. Old backups can refer to credentials
that have since been replaced and deleted; restoring settings may require entering
keys again. Explicit reset preserves known committed references when available,
but it cannot reconstruct references from an unreadable database. It never deletes
keys as part of reset. Do not log SQL, values, native account identifiers, or file
paths when handling these failures.

## Credential consistency

For a key replacement, persist cleanup intent, write a new randomly named native
account, and stage its reference. A single SQLite transaction commits all settings
and the selected account references. Only then may cleanup delete obsolete native
accounts. Failed saves discard staged references; orphan cleanup retries after a
successful load or save. Cleanup is bounded to 128 accounts per pass, and new key
replacements stop if the pending queue reaches 128 until cleanup can succeed.
Existing jobs already hold their private credential string and settings snapshot.

This is crash reconciliation across two stores, not an atomic transaction with
Windows Credential Manager. If the database commit outcome is uncertain, keep
both credential versions until the committed references can be reloaded. Startup
and shortcut state likewise reconcile from the database after restart.

## Acceptance

Run real temporary SQLite fixtures on Windows, including process exit during
initialization, an uncommitted settings write, and a goose upgrade; reopen them
and check committed state and backups. Cover lock contention, read-only/full disk,
constraints, foreign/newer history, backup restore, and credential/native rollback.
The Windows service integration uses an isolated fake vault and startup adapter;
it does not change personal credentials or startup registration. Interactive
app acceptance and real Credential Manager behavior remain distinct from fixtures
and compilation; follow the [native checklist](../../safety/native-test-checklist/).

## Saved connections

Migration 00003 imports singleton connections. Forward migration 00004 removes
connection-level model/preset columns and unused empty bootstrap entries;
current model and preset values remain in runtime tables. Earlier migrations
remain unchanged. Fresh initialization starts with no saved connections;
legacy import seeds only configured endpoints. Forward migration 00005 adds
`saved_connection_uses`, removes exclusive purpose ownership, and preserves
existing IDs, original uses, names, and opaque key references without merging
servers. Historical same-name entries are preserved; the serialized settings
owner rejects newly conflicting names across the library.

`saved_connections`, `selected_connections`, and `saved_connection_headers`
retain stable IDs, typed endpoint fields, opaque credential references,
bounded names and explicit foreign keys. Selected (connection, purpose) pairs
reference `saved_connection_uses`, so an undeclared use cannot be selected. Each capability has
zero or one active selection. `internal/savedconnection` owns domain validation
and selection application; `internal/settings` remains the sole save coordinator.

Explicit create/update actions take bounded connection details and a transient
credential draft. Creating or duplicating does not select. Runtime saves restore
the committed connection fields before validation and save only feature options.
Selection clears the role's model; optional features disable until configured.
Settings, selected IDs, entry mutations, and active credential references commit
in one transaction, then publish. Expected selected IDs reject stale editors.
An entry must be deselected from every active feature before deletion or removal
of an active use; empty catalogs are valid. Shared edits project endpoint details
into every selected feature while preserving model/options. Credential replacement
stages one connection-owned account and updates every selected reference in the
same transaction. Legacy credential adapters also synchronize all active uses.
Opaque references from older purpose-scoped accounts remain valid for shared use.

Inactive entries retain their keys. Duplication can share a reference, while
replacement creates a new reference for the edited entry. Cleanup excludes every
reference still used by any saved entry. SQL rollback leaves the old catalog and
keys intact. `TestSavedConnection` resolves the specified entry's details and key
under the store lock, never borrowing an active connection's credential. It reads
metadata only. Restored databases must pass catalog/runtime/reference checks.

Model-profile selection is feature-owned. Migration 00006 adds transcription and
speech IDs, defaulting to Generic; cleanup retains its existing `preset` ID.
Generated queries read/write all three through the same settings transaction.
The v5 upgrade fixture verifies cleanup choices, connections, credentials, and
restart persistence; unknown profile IDs enter recovery instead of silently
falling back to Generic. No connection or settings reset is required.

## Remembered model preferences

Migration `00007_remembered_models.sql` adds a STRICT table with an explicit
connection/purpose/model primary key, bounded typed option columns, a cascading
connection foreign key, and a partial unique index for each use's last selection.
It seeds current selections without changing active settings. Legacy import also
captures its active choices after creating connections. The settings transaction
persists remembered and active options together through sqlc; no configuration
JSON, credentials, audio, or generated transcript content is stored in these rows.
Go validates role ownership, model behavior, backend options, and per-use counts
on load and save. Recovery loads the same catalog and preserves existing backup,
forward-only migration, and commit-uncertainty behavior.

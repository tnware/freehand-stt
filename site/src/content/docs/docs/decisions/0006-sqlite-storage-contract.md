---
title: "ADR 0006: SQLite storage with goose and sqlc"
description: Define the required storage tooling, ownership boundaries, and validation contract.
---

- Status: Accepted; implemented
- Date: 2026-09-05

## Context

At the time of this decision, Freehand stored non-secret configuration in `settings.json` and kept
optional transcript history in memory. Saved connections and model preferences
will need structured persistence. The storage foundation must preserve coherent
settings/credential snapshots, native lifecycle ownership, and existing privacy
boundaries while making future schema changes reproducible.

## Decision

Use SQLite through `modernc.org/sqlite` and Go's `database/sql`, goose for schema
migrations, and sqlc for application queries. This is the required persistence
path for the SQLite implementation and subsequent features. Changes to these
tools or boundaries require a superseding architecture decision and corresponding
updates to repository instructions and validation.

The driver supplies embedded SQLite without introducing a SQLite-specific CGO
requirement. This does not remove Freehand's existing native build requirements.
Pin the selected driver, goose, and sqlc versions; test their combination on
Windows before shipping. Respect the driver's documented `modernc.org/libc`
version relationship when resolving dependencies.

### Ownership and layout

Keep database infrastructure together under `internal/storage`: connection
lifecycle, embedded `migrations/`, query source in `queries/`, sqlc output in a
private `dbgen/` package, and adapters exposing narrow operations to domain
owners. Keep configuration defaults and validation in `internal/config` and
coherent save/request ownership in `internal/settings`. The storage package must
not become a new Wails service or own dictation, credentials, or model behavior.

- The renderer cannot execute SQL or access database paths or handles.
- Application queries must come from sqlc-generated methods. Do not add an ORM,
  another query generator, or handwritten application query/scanning paths.
- Handwritten infrastructure SQL is limited to connection configuration,
  database identity/version inspection, integrity checks, and backup operations.
  DDL belongs in migrations; fixtures may use SQL to construct test conditions.
- Generated database rows remain internal to storage. Convert them to domain
  values; do not expose them as Wails DTOs or spread them across feature packages.
- Use explicit columns and bound parameters. Go validation and SQL constraints
  both protect stored values. Use `STRICT` tables and explicit relationship and
  deletion rules; enable foreign-key enforcement on every connection.

### Migration contract

Embed ordered SQL migrations in the executable and run the goose provider with
the application-owned database connection before settings-dependent services
start. Goose's migration table is the sole schema-version authority; do not
maintain a parallel application migration counter or homegrown runner.

Use consistently zero-padded sequence numbers, such as `00001_initial.sql`.
Point sqlc at the same migration directory as its schema input. Prefer SQL
migrations so runtime and generation share the schema. If a future SQLite table
rebuild cannot be parsed by sqlc, solve and test that generation problem without
introducing a second manually maintained schema.

Released migrations are immutable. Normal startup applies forward migrations
transactionally, including goose's version record. Nontransactional maintenance
must remain outside schema migrations. A newer or unrecognized migration history
blocks writes and produces a recoverable compatibility error; the app must not
silently downgrade a database or treat it as empty.

Before upgrading an existing database, make a consistent backup with SQLite's
backup facilities and enforce bounded retention. Test restoration. Never copy an
active database casually or replace a corrupt/unreadable database with defaults.

A bounded legacy importer may validate and import `settings.json` only when
initializing a new database. Import rows and completion state transactionally,
preserve the original file, and define retry behavior for interrupted imports.
It is a one-time compatibility adapter, not an alternative persistence path.

### Runtime and credential safety

Use a per-user local application-data directory with Windows access controls.
Start with one managed connection, short serialized writes, bounded lock waits,
and cancellation-aware operations. Apply and verify connection settings whenever
a connection opens. The initial durability policy is rollback journaling with
`synchronous=EXTRA`; a later switch to WAL requires evidence and explicit backup
and checkpoint handling. Do not hold database transactions across network calls.

The settings service publishes a new runtime snapshot only after persistence
succeeds. Active operations retain their captured settings and credentials.
SQLite transactions cannot atomically commit Windows Credential Manager or
native shortcut/startup changes. Preserve ordinary failure rollback and define
restart reconciliation from committed settings. Credential replacement must
preserve the credential referenced by the last committed configuration until
the replacement reference commits; clean up obsolete references only after
successful commit and when active requests no longer need them.

API keys remain in Windows Credential Manager. Only opaque credential references
may be stored in SQLite. Database errors and metrics must follow the existing
bounded logging contract, excluding query values, secrets, and user content.
Database storage does not authorize persistent transcripts, audio, audit trails,
or changes to history retention.

### Generation and acceptance contract

The implementation must include a pinned, reproducible local generation command
and matching CI checks in the same change. Commit generated query code and never
edit it by hand. CI regenerates it and rejects modified or newly generated
untracked output. It must also prevent changes to previously released migrations
and validate the agreed query/import boundaries.

Tests must execute actual migrations and generated queries against temporary
SQLite databases, including file-backed reopen and locking behavior. Cover fresh
creation, upgrades from retained schema fixtures, repeated startup, constraints,
failed commits, interrupted imports/upgrades, newer schemas, backup/restore, and
credential/native-state failure recovery. Record Windows runtime acceptance
separately from code generation and cross-compilation.

Implementation note: the database, importer, versioned credential references,
recovery operations, pinned generation command, and CI enforcement now follow
this contract. See [the storage maintenance guide](../../development/storage/)
for the concrete layout and validation commands. Saved connections subsequently use this foundation; reusable model preferences
remain separate feature work.

## References

- [modernc SQLite driver and platform support](https://pkg.go.dev/modernc.org/sqlite)
- [Goose provider and embedded migrations](https://pressly.github.io/goose/documentation/provider/)
- [sqlc SQLite integration](https://docs.sqlc.dev/en/latest/tutorials/getting-started-sqlite.html)
- [sqlc parsing of goose migrations](https://docs.sqlc.dev/en/latest/howto/ddl.html#goose)
- [SQLite strict tables](https://www.sqlite.org/stricttables.html)
- [SQLite backup API](https://www.sqlite.org/backup.html)

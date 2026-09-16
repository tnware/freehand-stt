---
title: SQLite storage
description: Maintain the schema, generated queries, credential references, and recovery contract.
---

Freehand persists non-secret settings in SQLite. Saved connections use stable server records
with explicit capability memberships. Remembered model preferences share the
settings transaction; transcript history remains optional and memory-only. Disposable window
placement remains in `window-state.json`, independent of settings recovery.

## Ownership and tools

| Concern | Owner |
| --- | --- |
| Defaults, compatibility rules, validation | `internal/config` |
| Save coordination, native rollback, captured request profiles | `internal/settings` |
| Database, backups, typed adapters, credential-reference lifecycle | `internal/storage` |
| Schema and version history | Embedded goose migrations in `internal/storage/schema/` |
| Application SQL | `internal/storage/queries`; sqlc output in `internal/storage/dbgen` |
| Actual API keys | Windows Credential Manager or macOS Keychain |

Runtime dependency versions live in `go.mod`; the generation command pins sqlc
in `build/scripts/storage/main.go`. Respect the driver's libc version relationship
when upgrading. SQLite adds no SQLite-specific CGO dependency; the native audio
and Wails build requirements still apply.

`00001_initial.sql` directly creates the current `STRICT` settings, connection,
vocabulary, remembered-model, and credential-reference schema.
Foreign keys and explicit deletion rules protect related rows. Go validates complete domain settings before writing and after reading.
Generated rows and database handles never cross into Wails or domain services.

## Change the schema or a query

1. Add the next five-digit goose SQL migration with a version greater than every
   migration on the target branch. Never edit, delete, or fill gaps below published
   versions. Runtime startup runs forward only; migrations must remain transactional.
   The contract check rejects Goose's `NO TRANSACTION` annotation regardless of case.
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
migrations in `schema/`, and database import/query boundaries. An alpha-only
Git base permits this explicit new lineage; it does not exempt a published
`schema/00001_initial.sql` from byte-for-byte immutability. Handwritten infrastructure SQL
is confined to `store.go` and `recovery.go`: connection settings, identity/version
inspection, integrity checks, and backup work. Application queries require sqlc.

## Recovery and verification

Use temporary databases and disposable credentials for failure and recovery tests;
never operate on personal settings. Exercise real SQLite migrations and reopen the
database after writes, interrupted upgrades, and recovery. Validate native credential
behavior separately on Windows and macOS.

Preserve the database identity and forward-only migration boundary. Do not read,
convert, or delete alpha `settings.db`, `settings.json`, or legacy credentials.
Unknown or newer schemas and uncertain commits must enter recovery instead of
silently loading defaults. Credentials remain in the native vault; the database
and its backups hold opaque references only.

For user-facing recovery steps, see
[saved-settings troubleshooting](../../guides/troubleshooting/#saved-settings-need-attention).
For native checks, use the [acceptance procedure](../../safety/native-test-checklist/).
Record results and limitations in the issue or pull request, not this guide.

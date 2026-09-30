# Backend Unit Test Report

**Date:** 2026-09-30  
**Scope:** Go backend handlers and current build/test quality  
**Result:** PASS with coverage gaps

## Summary

- Added 11 handler unit tests using `go-sqlmock`; tests do not connect to or modify the real MySQL database.
- `go test ./... -v -cover`: PASS, 11/11 tests passed.
- Handler statement coverage: **39.8%**.
- `go test -race ./...`: **Unable to verify**; the Windows test process exits with `0xc0000139` before producing a race result.
- `go build ./...`: PASS.
- `go vet ./...`: PASS.
- `cmd/server`, `config`, `db`, and `models` currently have no tests; reported coverage is 0% for packages with executable statements and `[no test files]` for models.

## Covered Behavior

| Area | Assertions |
| --- | --- |
| Health | Returns HTTP 200 and `status: ok` |
| Book creation | Rejects missing required fields; trims values; defaults status to `available`; checks insert and read-back |
| Member creation | Trims submitted fields; checks insert and read-back |
| Borrowing | Rejects unavailable books; success commits record and status update; downstream update failure rolls back |
| Returning | Updates the active record and book status inside a committed transaction |
| Book listing | Returns the expected serialized row |
| Statistics | Returns all five expected counts |
| Book deletion | Returns 404 when no row was deleted |

## Test Limitations

- SQL mock tests verify handler decisions, SQL arguments, HTTP responses, and transaction calls. They do **not** prove compatibility with a live MySQL server.
- The race detector could not complete in this Windows environment. `CGO_ENABLED=1` and GCC are present, but the race test process exits with `0xc0000139`; this is not a passing race result and does not indicate a reported data race.
- Database creation and schema setup in `internal/db.Open` are not covered. Add an isolated MySQL integration test to verify first-run database creation, table DDL, permissions, and restart behavior.
- The borrow tests verify the current sequential transaction path, not concurrent requests. Add a concurrency test after making the availability update atomic or locking the book row.
- Update/Get book, list members, list borrow records, and several SQL failure/row-scan paths are not covered.
- No frontend, API end-to-end, or deployment tests are included in this backend report.

## Recommended Quality Improvements

1. **P1 — Data integrity:** Make borrowing an atomic conditional update (`status = 'available'`) or use `SELECT ... FOR UPDATE`; check affected rows and prevent multiple active loans for one book at the database level. Add a parallel-request regression test.
2. **P1 — Database integration:** Test `db.Open` against an ephemeral MySQL instance. Verify a missing database is created, all tables are present, failures stop startup, and rerunning initialization is safe.
3. **P1 — API security:** Add authentication and authorization, restrict CORS origins, and use a least-privilege database account with secrets supplied outside source-controlled Compose configuration.
4. **P2 — Handler coverage:** Add tests for update/read endpoints, invalid IDs and payloads, SQL errors, scan failures, and `rows.Err()` handling. Keep status codes and error responses consistent.
5. **P2 — Operations:** Add database readiness checks/retry, request and database timeouts, graceful shutdown, and a health endpoint that distinguishes liveness from readiness.
6. **P2 — Automated quality gate:** Add CI steps for `gofmt` verification, `go vet ./...`, `go test -race ./...`, and a gradually raised coverage threshold.

## Reproduction

Run from `backend/`:

```powershell
go test ./... -v -cover
go test -race ./...
go vet ./...
go build ./...
```

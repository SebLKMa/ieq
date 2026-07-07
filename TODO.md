# TODO — Go 1.22.5 Upgrade and Code Review

Findings from a review of the codebase ahead of the Go 1.22.5 upgrade, ordered by
priority. Items in section 1 are hard blockers: the code will not build on Go 1.22
until they are done.

> **Status (2026-07-07): implemented.** All items are done except the optional
> `log/slog` migration (logging was standardized on `log` for now). Notes on the
> completed work:
>
> - Dependencies were upgraded in place; the optional `lib/pq` → `pgx` migration
>   was not taken.
> - `LightingFormula` gained a `NewLightingFormula(scale)` constructor; the
>   struct stays exported for compatibility and `SetScale` is marked deprecated.
> - The exposed uHoo credential was removed from the code, but it lives on in
>   git history — **the credential still needs to be rotated by the owner.**
> - The DB scan-error fix has no unit test (it needs a live postgres); the other
>   section 3 fixes are covered by tests, and the sensor HTTP paths are now
>   tested against `httptest` fakes.

## 1. Build blockers for Go 1.22

**Purpose:** Make the repository compile at all under the Go 1.22.5 toolchain.

**Description:** The repo predates the module era in practice: most packages have no
`go.mod` and relied on GOPATH-mode builds, while the few modules present are pinned
to `go 1.15` with 2020-era dependencies. Module mode has been the default since
Go 1.16, and Go 1.22 removed `go get` support in legacy GOPATH mode, so dependency
management for this layout is dead. Everything else in this document assumes these
items are done first — nothing can be built, vetted or tested repo-wide until then.

- [x] **Consolidate into a single Go module.** Go 1.22 removed GOPATH mode, and most
  packages (`configs`, `formulas`, `frontend`, `scoreserver`, `sensors`, `tasks`,
  `utils`) have no `go.mod` — they only ever built in GOPATH mode. Create one root
  `go.mod` (`module github.com/seblkma/ieq`, `go 1.22`) and delete the per-package
  modules in `interfaces/`, `models/`, `ratings/` and `db/postgres/` along with their
  `replace` directives. Keep `gqlgen-ieq/` as its own module only if it must stay
  independently versioned; otherwise fold it in too.
- [x] **Regenerate gqlgen code.** `gqlgen-ieq/server.go` and
  `graph/schema.resolvers.go` import `gqlgen-ieq/graph/generated`, but no `generated`
  package is committed — the module cannot compile until `go generate ./...` is run.
  Commit the generated code (or document the generate step in CI).
- [x] **Upgrade dependencies.** All pinned versions are from ~2020:
  - `github.com/99designs/gqlgen` v0.13.0 → current release (breaking API changes
    expected; regenerate after upgrading).
  - `github.com/lib/pq` v1.9.0 → current, or migrate to `jackc/pgx` (lib/pq is in
    maintenance mode).
  - `gopkg.in/yaml.v3` → latest patch (older versions have known CVEs).
  - Run `go mod tidy` after consolidation.
- [x] **Replace `github.com/christophwitzko/go-curl`** (`sensors/awair/awairdevice.go`).
  The library is unmaintained, has an unidiomatic API (`err` as first return value),
  and everything it does is a plain GET with headers — use `net/http` directly, as
  `sensors/uhoo` already does.

## 2. Deprecated / superseded stdlib APIs

**Purpose:** Move off standard-library APIs that are deprecated or have modern
replacements, so the code lints clean and reads as current Go.

**Description:** These changes are mechanical and low-risk: none alter behaviour,
but they remove deprecation warnings (`io/ioutil`), adopt spellings introduced by
newer Go releases (`any`, `log/slog`), and take advantage of Go 1.22's routing
improvements in `net/http` that eliminate hand-rolled request plumbing. Doing them
during the upgrade keeps the codebase from accumulating two styles side by side.

- [x] `sensors/uhoo/uhoodevice.go`: replace `io/ioutil.ReadAll` with `io.ReadAll`
  (`io/ioutil` is deprecated since Go 1.16).
- [x] Replace `interface{}` with `any` throughout (`utils/size.go`, both sensor
  packages).
- [x] `frontend/main.go`: use Go 1.22's enhanced `net/http.ServeMux` patterns
  (`mux.HandleFunc("GET /ieq/device", ...)`) instead of bare `http.HandleFunc`,
  which also removes the need for manual method handling.
- [ ] Consider `log/slog` (Go 1.21) to replace the current mix of `log.Printf`,
  `fmt.Println` and `fmt.Printf` used for operational logging in `db/`, `sensors/`,
  `tasks/` and `ratings/compute.go`.

## 3. Correctness bugs

**Purpose:** Fix defects that produce wrong scores, silently lose errors, or leak
internal details — independent of the Go version.

**Description:** These were found by reading the code, not by the compiler, and
they affect the system's core output: stored scores can be computed from partial
data (ignored `AddIndex` errors), database failures can return empty results with
a nil error (wrong variable returned after `Scan`), and scoring failures are
recorded as a legitimate 0. Each fix should land with a regression test (see
section 7), because none of them are caught by the existing test suite.

- [x] **`db/postgres/ieqdb.go` and `device.go`: Scan errors are swallowed.** Every
  `Read*` function does `if err2 != nil { return data, err }` — returning the (nil)
  `err` from `Query` instead of `err2` from `Scan`, so a scan failure returns zero
  data with a nil error. Return `err2` (and check `rows.Err()` after the loop).
- [x] **`ratings/standard.go` `AddIndex` validation is wrong and its errors are
  ignored.** It rejects an index when the *sum of scores* exceeds 100, but each
  metric score legitimately ranges up to 100 (Thermal = Temperature + Humidity can
  sum to 200). With good readings, the second index is silently rejected — and every
  `AddIndex` call in `tasks/scoreexecute.go` and `ratings/ieq.go` discards the error,
  so the rating is then computed from a partial index list. Fix the validation
  (weightings should sum to 100, not scores) and check the returned errors.
- [x] **`ratings/compute.go` `ComputeScore` hides failures.** When `Score` returns
  `ok == false` it prints "Unable to compute score :(" and returns 0 — callers then
  store 0 as a legitimate score. Return an error (or the `ok` flag) to the caller.
- [x] **`formulas/lighting.go` `Setup`:** the first range is inserted twice (once
  before the loop, again on the first loop iteration), creating a duplicate node in
  the tree.
- [x] **`tasks/scoreexecute.go` duration arithmetic is obscure:**
  `time.Duration(minutes * 60000)` later multiplied by `time.Millisecond` happens to
  work but reads as a bug. Use `time.Duration(minutes) * time.Minute` once.
- [x] **`frontend/main.go`:** error responses are written with `fmt.Fprintf` and
  status 200, leaking internal error strings (including DB errors) to clients. Use
  `http.Error` with proper status codes and log the details server-side.
- [x] **`frontend/main.go` `main`:** the `http.ListenAndServe` error is discarded —
  wrap in `log.Fatal(...)` so bind failures are visible.
- [x] **`sensors/uhoo/uhoodevice.go`:** HTTP status codes are never checked in
  `GetState`/`GetRawMetrics`; a 401/500 body gets passed to `json.Unmarshal` and
  produces a confusing downstream error.
- [x] **`utils/utils.go` `FileExists`:** if `os.Stat` fails with an error other than
  not-exist (e.g. permission denied), `info` is nil and `info.IsDir()` panics.

## 4. Security / secret hygiene

**Purpose:** Get credentials out of source control and harden the code paths that
handle them.

**Description:** The repository currently contains a real credential pair in
commented-out code, hardcoded database credentials, and vendor tokens expected in
committed YAML files. Because git history preserves all of these forever, removal
alone is not enough — exposed credentials must also be rotated. The remaining items
(parameterized `LIMIT`, `url.Values` encoding) are defence-in-depth: not exploitable
today, but they close off classes of bugs rather than instances.

- [x] **Remove real credentials from comments** in `sensors/uhoo/uhoodevice.go`
  (lines ~25 and ~55): commented-out request bodies contain an actual username and
  password hash. Delete the comments and rotate the uHoo credential — it is in git
  history.
- [x] **`db/postgres/connect.go`:** hardcoded host/user/password/dbname (already
  flagged by the in-code TODO and README). Read from environment variables or config.
- [x] **`db/postgres/ieqdb.go` `ReadMetrics`:** the `LIMIT` clause is built by string
  concatenation (`strconv.Itoa(count)`). It's an `int` so not injectable today, but
  make it a `$2` query parameter for consistency.
- [x] **`sensors/uhoo`:** credentials are form-encoded by string concatenation; use
  `url.Values.Encode()` so special characters in tokens don't corrupt the request.
- [x] Device tokens live in the scoreserver YAML files; keep placeholder-only files
  in git and load real tokens from the environment.

## 5. Architecture and design cleanups

**Purpose:** Make the code use its own abstractions, manage resources the way the
standard library intends, and remove duplication that will otherwise be paid for
on every future change.

**Description:** The project defines good seams — `interfaces.Device`,
`interfaces.Scorer`, `interfaces.Executable` — but the task layer bypasses them
with duplicated per-vendor code. The database layer treats `sql.DB` as a one-shot
connection instead of the long-lived pool it is designed to be, and no HTTP or DB
call carries a timeout or `context.Context`, so one hung vendor API stalls a
scoring loop indefinitely. These items restructure without changing observable
behaviour (except the awair vendor-name mismatch fix, which is also a bug).

- [x] **Use the `interfaces.Device` abstraction in `tasks/scoreexecute.go`.** The
  `awair` and `uhoo` switch cases are ~25 identical lines each; both sensor types
  already satisfy `interfaces.Device`. Construct the right `Device` from
  `Cfg.VENDOR.Name` once, then run one shared code path. (Note: the config uses
  `Name: awair-device` in `configawair.yaml` but the switch matches `"awair"` — the
  awair task currently matches nothing; verify and fix while refactoring.)
- [x] **`db/postgres`: stop opening a new connection pool per call.** Every function
  does `connect()` + `defer db.Close()`; `sql.DB` is a pool designed to be created
  once and shared. Create it at startup, `Ping` to validate, and pass it (or a small
  repository struct) to callers. Remove the `panic` in `connect`.
- [x] **Add `context.Context`** to the DB layer (`QueryContext`/`ExecContext`) and
  sensor HTTP calls (`http.NewRequestWithContext`), and use a shared `http.Client`
  with a timeout — `http.DefaultClient` has none, so a hung vendor API hangs the
  scoring loop forever.
- [x] **`db/postgres`: replace `SELECT *` with explicit column lists.** Positional
  `Scan` against `SELECT *` breaks silently if the table gains or reorders columns.
  Also the three `ReadLatest*` functions and `ReadMetrics` are near-duplicates —
  factor the row-scan loop.
- [x] **`formulas/mingood.go` `Setup` duplicates `standard.go` `Setup` verbatim** —
  delegate to the embedded `StandardFormula.Setup` instead.
- [x] **`tasks/scoreexecute.go`:** replace the 1-second busy-poll waiting for a
  5-minute boundary with a single computed `time.Sleep` (or `time.Ticker`), and use
  a `time.Ticker` for the main loop. Add graceful shutdown (signal handling +
  context cancellation) to both `scoreserver` and `frontend`.
- [x] **`frontend/main.go`:** embed templates with `//go:embed gotemplates/...`
  instead of 25 hardcoded `../gotemplates/...` relative paths that break unless the
  binary runs from `frontend/`. Cache the `time.LoadLocation("Asia/Singapore")`
  result at startup instead of per request (and consider making the zone
  configurable).
- [x] **`frontend/main.go` `getDeviceIDFromURL`:** drop the unused
  `http.ResponseWriter` parameter and the re-parse of `r.URL.String()` —
  `r.URL.Query().Get("device_id")` is sufficient.
- [x] **Sensor JSON decoding:** replace the `map[string]interface{}` + type-switch
  ladders in `awair.GetDeviceInfo` and both uhoo functions with typed structs and
  plain `json.Unmarshal`, as `awairmetrics.go` already does. Shorter, faster, and
  type-checked at compile time.

## 6. Go 1.22-era modernization (optional but recommended)

**Purpose:** Adopt language and library features added between Go 1.15 and 1.22
where they make the code shorter, faster, or harder to misuse.

**Description:** Six releases of language evolution separate this codebase from
the target toolchain: generics and the `slices` package, `min`/`max` builtins,
`go:embed`, and stricter community lint conventions all postdate the original
code. None of these items are required to ship the upgrade, but each one either
deletes code (slice reversal, sorted-slice search), removes an ordering trap
(`SetScale` before `Setup`), or eliminates shadowing of names that are now
builtins.

- [x] `utils/skiptree`: inserts arrive in ascending key order, so the BST degenerates
  into a linked list (O(n) search) — the README itself notes balance matters. For
  these small, build-once/read-many range tables, a sorted slice +
  `sort.Search`/`slices.BinarySearchFunc` is simpler and faster; alternatively make
  the tree self-balancing. Generics (Go 1.18+) could replace the float64-only API if
  reuse is intended.
- [x] `frontend/main.go`: replace the index-arithmetic reversal loop (whose loop
  variable `m` is otherwise unused) with `slices.Reverse` (Go 1.21).
- [x] `ratings/standard.go`, `ratings/ieq.go`: rename the local variable `len` — it
  shadows the builtin. Likewise `min`/`max` locals in `formulas` and `tasks` now
  shadow the Go 1.21 builtins; legal, but worth renaming during the upgrade.
- [x] Error strings: `errors.New("No record found")`, `"Awair data is empty"`, etc.
  violate ST1005 (capitalized); lowercase them and prefer `fmt.Errorf` with `%w` for
  wrapping. Consider a sentinel `ErrNoRecord` so callers can distinguish "no data
  yet" from real failures instead of string-matching.
- [x] `utils/size.go` `SizeOfPublicStruct` and `utils/utils.go` flag helpers appear
  unused — confirm and delete dead code (also the commented-out debug `fmt.Printf`
  blocks scattered through `formulas`, `ratings`, `sensors`, `frontend`).
- [x] `formulas/lighting.go`: implement the existing in-code TODO — unexport the
  struct and provide a constructor requiring the scale, removing the fragile
  "SetScale must be called before Setup" ordering contract.

## 7. Tooling and tests for the upgrade

**Purpose:** Put verification in place so the upgrade — and the bug fixes riding
along with it — can be proven correct and kept that way.

**Description:** Today the repo cannot run `go build ./...` or `go test ./...`
from the root, so there is no single command that says "the upgrade worked."
This section establishes that baseline, adds static analysis that would have
caught several section 2 and 6 items mechanically, and calls out the untested
packages (`db`, `sensors`, `tasks`) whose bugs in section 3 slipped through
precisely because nothing exercises them.

- [x] After the module consolidation, get `go build ./...`, `go vet ./...` and
  `go test ./...` passing at the repo root — today tests can only run per-directory.
- [x] Add `gofmt`/`govet` plus `staticcheck` or `golangci-lint` to catch the
  deprecation and shadowing issues above mechanically.
- [x] Existing tests cover only `formulas`, `ratings` and `skiptree`. The bug fixes
  in section 3 (AddIndex validation, DB scan errors, lighting duplicate insert)
  should each land with a regression test; `db` and `sensors` currently have no
  tests at all (consider `httptest` fakes for the vendor APIs).

## 8. Cited best practices

The recommendations above are grounded in the following published guidance:

1. **Go modules are the standard build mode; GOPATH mode is legacy.** Module mode
   is the default since Go 1.16, and Go 1.22 dropped `go get` support under
   `GO111MODULE=off`. — *Go 1.16 and Go 1.22 Release Notes*
   (https://go.dev/doc/go1.16, https://go.dev/doc/go1.22)
2. **`io/ioutil` is deprecated; use `io` and `os` equivalents.** — *Go 1.16 Release
   Notes and `io/ioutil` package documentation* (https://pkg.go.dev/io/ioutil)
3. **Use `any` as the alias for `interface{}`.** Introduced with generics. — *Go
   1.18 Release Notes* (https://go.dev/doc/go1.18)
4. **Prefer `log/slog` for structured, leveled logging.** — *"Structured Logging
   with slog", Go Blog* (https://go.dev/blog/slog)
5. **Use Go 1.22's method-and-wildcard `ServeMux` patterns** instead of manual
   method/path dispatch. — *"Routing Enhancements for Go 1.22", Go Blog*
   (https://go.dev/blog/routing-enhancements)
6. **`sql.DB` is a long-lived connection pool: open once, share, do not
   open/close per query.** — *"Accessing a relational database" and "Managing
   connections", go.dev tutorials* (https://go.dev/doc/database/manage-connections)
7. **Always use parameterized queries; never build SQL from strings.** — *"Avoiding
   SQL injection risk", go.dev* (https://go.dev/doc/database/sql-injection) and
   *OWASP SQL Injection Prevention Cheat Sheet*
8. **`lib/pq` is in maintenance mode; new work should prefer `pgx`.** — *lib/pq
   project README* (https://github.com/lib/pq)
9. **Never discard errors; handle every returned `error`.** — *Effective Go,
   "Errors"* (https://go.dev/doc/effective_go#errors) and *Go Code Review
   Comments, "Handle Errors"* (https://go.dev/wiki/CodeReviewComments)
10. **Error strings should be lowercase and unpunctuated.** — *Go Code Review
    Comments, "Error Strings"*; enforced by staticcheck check ST1005
    (https://staticcheck.dev/docs/checks/#ST1005)
11. **Wrap errors with `fmt.Errorf("...: %w", err)` and match with
    `errors.Is`/`errors.As`; expose sentinel errors for expected conditions.** —
    *"Working with Errors in Go 1.13", Go Blog* (https://go.dev/blog/go1.13-errors)
12. **Propagate `context.Context` through I/O paths** (`QueryContext`,
    `http.NewRequestWithContext`) for cancellation and deadlines. — *"Go
    Concurrency Patterns: Context", Go Blog* (https://go.dev/blog/context)
13. **`http.DefaultClient` has no timeout; production clients must set one and be
    reused.** — *`net/http` package documentation* (https://pkg.go.dev/net/http)
    and *"The complete guide to Go net/http timeouts", Cloudflare Blog*
14. **Shut down HTTP servers gracefully with `http.Server.Shutdown`.** — *`net/http`
    package documentation* (https://pkg.go.dev/net/http#Server.Shutdown)
15. **Embed static assets with `//go:embed`** rather than loading them via
    working-directory-relative paths. — *`embed` package documentation*
    (https://pkg.go.dev/embed)
16. **Use the `slices` package and `min`/`max` builtins** where they replace
    hand-written loops; avoid shadowing predeclared identifiers (`len`, `min`,
    `max`). — *Go 1.21 Release Notes* (https://go.dev/doc/go1.21) and *Effective Go*
17. **Keep configuration and credentials in the environment, not in the codebase.**
    — *The Twelve-Factor App, "III. Config"* (https://12factor.net/config)
18. **Run `gofmt`, `go vet`, and a linter such as staticcheck/golangci-lint in
    CI.** — *Effective Go, "Formatting"* (https://go.dev/doc/effective_go) and
    *staticcheck documentation* (https://staticcheck.dev)
19. **Prefer table-driven tests and `net/http/httptest` fakes for HTTP-dependent
    code.** — *Go Wiki, "TableDrivenTests"* (https://go.dev/wiki/TableDrivenTests)
    and *`httptest` package documentation* (https://pkg.go.dev/net/http/httptest)
20. **Express durations with `time.Duration` unit constants** (e.g.
    `time.Duration(n) * time.Minute`), not raw integer arithmetic. — *`time`
    package documentation* (https://pkg.go.dev/time#Duration)

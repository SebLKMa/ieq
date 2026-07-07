# SKILLS.md — Reusable Go Review and Refactoring Skills

Distilled from the Go 1.22.5 upgrade review in [TODO.md](./TODO.md). Where TODO.md
lists one-time fixes for this repo, this file generalizes them into repeatable
skills: what to look for, why it matters, and how to apply the fix in any Go
codebase. Sources for each practice are cited in TODO.md section 8.

## Skill 1: Modernize a legacy Go module layout

**When to use:** taking over any pre-2021 Go codebase, or one with multiple
`go.mod` files stitched together by `replace` directives.

- Check every directory for missing `go.mod` files — GOPATH-mode packages won't
  build under module mode (the default since Go 1.16), and `go get` no longer
  works in GOPATH mode as of Go 1.22.
- Prefer one module per repository. Multiple sub-modules with `replace`
  directives add versioning friction and break `go build ./...` from the root;
  only keep a sub-module when it genuinely needs independent versioning.
- Bump the `go` directive to the target toolchain, then `go mod tidy` and
  upgrade dependencies. Check each dependency's README for maintenance status
  (e.g. `lib/pq` recommends `pgx`) and replace unmaintained ones with the
  standard library where possible.
- Verify generated code (gqlgen, protobuf, mocks) is either committed or
  reproducibly generated in CI — an imported-but-missing generated package is a
  silent build blocker.
- Acceptance test for the skill: `go build ./...`, `go vet ./...` and
  `go test ./...` all pass from the repository root.

## Skill 2: Sweep for deprecated and superseded APIs

**When to use:** any toolchain upgrade that skips multiple Go releases.

- `io/ioutil` → `io` / `os` equivalents (deprecated since 1.16).
- `interface{}` → `any` (1.18).
- Hand-rolled method/path dispatch → `ServeMux` patterns like
  `mux.HandleFunc("GET /path/{id}", ...)` and `r.PathValue("id")` (1.22).
- Mixed `fmt.Println`/`log.Printf` operational logging → `log/slog` (1.21).
- Index-arithmetic loops over slices → `slices` package helpers
  (`slices.Reverse`, `slices.BinarySearchFunc`, 1.21).
- Rename locals that shadow newer predeclared identifiers (`min`, `max` became
  builtins in 1.21; `len` was always one).
- Run `staticcheck` after the sweep — it catches most of these mechanically.

## Skill 3: Audit error handling

**When to use:** every review; highest-value in code that stores or reports
computed results.

- Grep for discarded return values of functions that return `error` — an
  ignored validation error means downstream code runs on partial data.
- In `err`/`err2` pairs, verify the *right* variable is returned; returning the
  outer (nil) error after an inner failure silently converts failures into
  empty-but-successful results.
- Functions that print a message and return a zero value on failure hide errors
  from callers; return the error (or `ok` flag) and let the caller decide.
- Distinguish expected conditions from failures with sentinel errors
  (`ErrNoRecord`) matched by `errors.Is`, instead of string comparison.
- Wrap with `fmt.Errorf("context: %w", err)`; keep error strings lowercase and
  unpunctuated (ST1005).
- After `rows.Next()` loops, always check `rows.Err()`.

## Skill 4: Review a database/sql access layer

**When to use:** any code that touches `database/sql`.

- `sql.DB` is a connection pool: open once at startup, `Ping` to validate,
  share it (inject via a repository struct). A `connect()` + `defer Close()`
  per query defeats pooling and exhausts connections under load.
- Every query value must be a placeholder (`$1`, `$2`) — including `LIMIT` —
  never string concatenation, even for values that are "safe today".
- Replace `SELECT *` + positional `Scan` with explicit column lists; schema
  changes otherwise corrupt data silently instead of failing loudly.
- Use the `Context` variants (`QueryContext`, `ExecContext`) so callers can
  impose timeouts and cancellation.
- Near-duplicate `ReadLatestX` functions signal a missing generic row-scan
  helper; factor the loop once.

## Skill 5: Harden HTTP clients and servers

**When to use:** any code making outbound API calls or serving HTTP.

Clients:
- Never use `http.DefaultClient` for production calls — it has no timeout, so
  one hung upstream blocks the caller forever. Create one shared `http.Client`
  with a timeout and reuse it.
- Build requests with `http.NewRequestWithContext`; encode form data with
  `url.Values.Encode()`, not string concatenation.
- Check `resp.StatusCode` before decoding the body; decoding an error page as
  JSON produces misleading downstream failures.

Servers:
- Return errors with `http.Error` and correct status codes; never write
  internal error strings (especially DB errors) into a 200 response.
- Check the error from `http.ListenAndServe`; add graceful shutdown via
  `http.Server.Shutdown` on SIGINT/SIGTERM.
- Read query parameters with `r.URL.Query().Get(...)` — the URL is already
  parsed; don't re-parse `r.URL.String()`.
- Hoist per-request invariants (e.g. `time.LoadLocation`) to startup.

## Skill 6: Keep secrets out of the repository

**When to use:** before any commit; as a dedicated pass when inheriting a repo.

- Search comments and dead code for credentials, not just live code — the
  giveaway pattern here was a commented-out sample request body containing a
  real username and password hash.
- A secret found in git history must be **rotated**, not merely deleted; the
  deletion doesn't reach clones and forks.
- Move hardcoded DB credentials and vendor tokens into environment variables or
  an ignored local config; commit placeholder-only examples.
- Config files that ship with the binary (YAML task configs) should carry no
  secrets — inject tokens from the environment at load time.

## Skill 7: Make code use its own abstractions

**When to use:** whenever a package defines interfaces that concrete call sites
bypass.

- Duplicated per-vendor/per-variant blocks (two ~25-line switch cases differing
  only in the constructor) mean an existing interface isn't being used —
  construct the implementation once from config, then run one shared path.
- While consolidating, diff the variants carefully: divergence hides bugs (here,
  a config `Name: awair-device` that the switch's `case "awair"` never matched).
- Setup-order traps (`SetScale` must precede `Setup`) should become constructor
  parameters; make invalid states unrepresentable rather than documented.
- Duplicated method bodies between a type and its wrapper (`MinIsGoodFormula.Setup`
  re-implementing `StandardFormula.Setup`) should delegate instead.

## Skill 8: Question hand-rolled data structures and schedulers

**When to use:** custom trees/lists/queues, or loops that wait for wall-clock
conditions.

- A binary search tree fed keys in sorted order degenerates into a linked list;
  for small build-once/read-many tables prefer a sorted slice with
  `sort.Search`/`slices.BinarySearchFunc` over balancing a custom tree.
- Replace busy-wait polling ("sleep 1s until minute%5==0") with a single
  computed sleep to the next boundary, then a `time.Ticker`.
- Express durations as `time.Duration(n) * time.Minute` — raw integer
  arithmetic multiplied by a unit later reads as (and eventually becomes) a bug.
- Embed static assets with `//go:embed` instead of `../`-relative paths that
  break when the binary runs from a different working directory.

## Skill 9: Establish a verification baseline before refactoring

**When to use:** at the start of any upgrade or refactoring effort, not the end.

- First make `go build ./... && go vet ./... && go test ./...` runnable from
  the root; without a single green command, no refactor can be proven safe.
- Add `staticcheck` or `golangci-lint`; most deprecation, shadowing and
  error-string findings in this review were mechanically detectable.
- Land every bug fix with a regression test that fails before the fix.
- For untested I/O layers, start with `httptest` fakes for external APIs and
  table-driven tests for pure logic (formulas, ratings) — highest coverage per
  line of test code.

## References

See [TODO.md section 8](./TODO.md#8-cited-best-practices) for the full list of
cited sources: Go release notes (1.16–1.22), the Go Blog (slog, routing, errors,
context), Effective Go, Go Code Review Comments, go.dev database tutorials,
OWASP, staticcheck, and The Twelve-Factor App.

# Phase 0: Go and the groundwork

**Goal:** you can read, write, test and lint Go, and you can explain every file in this
repository as it stands today.

**Time:** about 1.5 weeks at 10–12 h/week. Most of it is the Go tour, then the exercises.

**Done when:** you've finished the exercises in §6 and can answer the quiz in §7 without
peeking.

---

## 1. Learn the language first (about 8 h)

Do the official tour at **https://go.dev/tour**, in this order:

| Section | Why it matters for this project |
|---|---|
| Basics (packages, variables, functions) | Everything |
| Flow control (for, if, switch, defer) | `defer` closes every file and connection we open |
| More types (structs, slices, maps) | Chunks are `[]byte` slices; the DHT routing table is a slice of buckets |
| Methods and interfaces | `io.Reader` and `io.Writer` are how all our bytes move |
| Generics | Skim it; we'll rarely need it |
| Concurrency (goroutines, channels, `select`) | **The most important one.** Every peer connection is a goroutine |

Then read **"Effective Go"** (https://go.dev/doc/effective_go), the sections on *Names*,
*Errors* and *Concurrency*. It explains why Go code looks the way it does.

**Mental model to keep in mind.** Go is deliberately small. There are no classes, no
inheritance and no exceptions:
- you get structs, methods and interfaces;
- errors are ordinary values you return and check;
- concurrency comes from goroutines (cheap threads) that talk over channels.

When something feels missing, that's usually on purpose.

---

## 2. Walkthrough: every file in the repo

### `go.mod`: the module definition
```
module github.com/aman99dex/Decentralised-P2P-File-Sharing
go 1.26
toolchain go1.27.2
```
- **`module`** is this project's import path. It matches the GitHub URL, so
  `go install github.com/aman99dex/Decentralised-P2P-File-Sharing/cmd/p2p@latest` works for
  anyone.
- **`go 1.26`** is the *minimum* Go version needed to build. We picked one version behind the
  latest (N-1) so people on slightly older Go can still build it.
- **`toolchain go1.27.2`** is the version *we* develop with. When your installed Go is older,
  the `go` command downloads this one automatically. You saw this happen: the
  "downloading go1.27.2" line.
- There's no `require` block yet because we have **zero dependencies**. Every dependency we
  add later needs a line in an ADR saying why. The rule is: standard library first.

### `cmd/p2p/main.go`: the program
```go
package main
```
`package main` together with `func main()` makes an **executable**. Any other package name is
a library. By convention, binaries live in `cmd/<name>/` and the real logic goes in
`internal/…`. Code in `internal/` can't be imported by other projects, which keeps our public
surface small.

```go
var version = "dev"
```
This is a package-level variable. At release time GoReleaser runs
`go build -ldflags "-X main.version=v0.1.0"`, and the **linker** overwrites the value. So the
binary knows its own version without us editing code for each release.

```go
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
```
This is the **"thin main" pattern**, the most important idea in this file:
- `main` does nothing except call `run` and exit with its result.
- `run` receives its inputs as parameters (args, stdout, stderr) instead of reaching for the
  globals `os.Args` and `os.Stdout`.
- So a test can call `run` with fake args and in-memory buffers, with no process and no
  terminal. Code that takes its dependencies as parameters is easy to test. Code that grabs
  globals isn't.
- `os.Args[1:]` drops `os.Args[0]`, which is the program's own name.

```go
func run(args []string, stdout, stderr io.Writer) int {
```
**`io.Writer` is your first interface.** It's defined in the standard library as:
```go
type Writer interface { Write(p []byte) (n int, err error) }
```
Anything with that `Write` method *is* a Writer: files, network connections, `os.Stdout`,
in-memory buffers, hash functions. You never declare "implements". If the method exists, the
type satisfies the interface. This is called **structural typing**, and it's why the same
`run` works with a real terminal and with a test buffer.

```go
switch args[0] {
case "send", "receive", "share", "get", "relay":
```
A Go `switch` doesn't fall through to the next case (no `break` needed), and one case can
list several values.

**Exit codes follow Unix convention:**
- `0` means success;
- `1` means a runtime failure;
- `2` means you used the command wrongly.

Scripts and CI rely on these numbers.

### `cmd/p2p/main_test.go`: the tests
- **Test files:** files ending in `_test.go` are compiled only by `go test`.
- **Test functions:** a test is `func TestXxx(t *testing.T)`.
- **Table-driven tests**, the Go idiom:
  - a slice of anonymous structs, one per case, and a single loop that checks them all;
  - adding a case is one line;
  - you'll write dozens of these, for example for the SPAKE2 test vectors.
- **Subtests:** `t.Run(tt.name, …)` creates a named subtest, so a failure prints *which*
  case broke. Run one on its own with `go test -run 'TestRun/version' ./cmd/p2p`.
- **`bytes.Buffer`** has a `Write` method, so it's an `io.Writer`. That's the interface paying
  off.
- **`t.Errorf` vs `t.Fatalf`:** `t.Errorf` records a failure and keeps going, so you see every
  wrong field at once. `t.Fatalf` stops the test immediately. Use it when continuing would
  panic.

### `.github/workflows/ci.yml`: continuous integration
On every push and pull request, GitHub runs three jobs:
1. **test**, on Linux, macOS and Windows:
   - `go vet` catches suspicious code: printf mistakes, unreachable code, copied locks.
   - `go test -race` runs the tests with the **race detector**. It instruments memory
     accesses and reports two goroutines touching the same data without synchronisation.
     Data races are the worst P2P bugs because they show up only under load.
   - Windows skips `-race` because it needs a C compiler on that runner.
2. **lint** runs golangci-lint (§3).
3. **vulncheck** runs `govulncheck`. It checks for known CVEs in Go and in our dependencies,
   and reports only the ones in code we actually *call*, which avoids noise.

Three security habits are in that file. Interviewers notice them:
- **Actions pinned to a commit SHA**, not a tag like `@v7`. A tag can be moved to point at
  malicious code (this has happened, e.g. tj-actions in 2025). A SHA can't change. The
  `# v7.0.1` comment keeps it readable.
- **`permissions: contents: read`.** The workflow token can't push code or touch releases.
- **`persist-credentials: false`.** The git token isn't left on disk for later steps to steal.

### `.golangci.yml`: the linter
golangci-lint runs many linters in one pass.
- `standard` gives the core set: errcheck (an ignored error is a bug), staticcheck, unused,
  and more.
- We added three:
  - **gosec**: security smells. It matters a lot for a crypto and networking project.
  - **errorlint**: makes sure wrapped errors are compared with `errors.Is` and `errors.As`.
  - **misspell**.

**A real failure from this phase.** The first CI run went red: errcheck flagged every
`fmt.Fprintf(stderr, …)` in `main.go`, because Go functions return errors and ignoring one is
usually a bug. Writing to the terminal is the exception: if that write fails, there is nothing
useful to do. So `.golangci.yml` exempts exactly `fmt.Fprint`, `fmt.Fprintf` and
`fmt.Fprintln`, with a comment explaining why. We **didn't** exempt everything. When we later
write to network connections with `conn.Write`, errcheck will force us to handle every error,
and that's what we want. The lesson: **tune a linter narrowly and write down why. Never
silence it wholesale.**

### `.gitignore`, `.gitattributes`, `LICENSE`, `README.md`
- **`.gitattributes`** with `eol=lf` stores every file with Unix line endings, even when you
  edit on Windows. Without it, `gofmt` in CI on Linux would complain about the CRLF line
  endings Windows adds.
- **MIT LICENSE.** It's permissive: anyone may use the code, and it's the norm for developer
  tools.

### Git: tag, branch, pull request
- **`v0-tracker`** is an *annotated tag* on the last Python commit. The old code is gone from
  `main` but preserved forever at
  https://github.com/aman99dex/Decentralised-P2P-File-Sharing/tree/v0-tracker.
- **We worked on a branch (`phase-0`) and opened a pull request** instead of pushing to `main`.
  CI proves the change is green *before* it lands. Every phase will be its own PR, so your
  GitHub history reads like a professional project.

---

## 3. The commands you'll use every day

```
go build ./cmd/p2p           # compile → p2p.exe
go run ./cmd/p2p version     # compile + run in one step
go test ./...                # all tests in every package (./... = "this dir and below")
go test -race ./...          # with the race detector (needs gcc on Windows; CI does it)
go test -run TestRun -v ./cmd/p2p   # one test, verbose
go vet ./...                 # suspicious-code check
gofmt -l .                   # list files that aren't formatted (empty = good)
go doc io.Writer             # read the docs for anything, offline
```

**Formatting is not a style debate in Go:** `gofmt` decides, and everyone's code looks the
same. Set your editor (VS Code plus the Go extension) to format on save.

---

## 4. Go idioms you met in this phase

| Idiom | Where | One-line meaning |
|---|---|---|
| Thin `main`, testable `run` | `main.go` | Inject dependencies so code is testable |
| Accept interfaces (`io.Writer`) | `run` signature | Depend on behaviour, not on concrete types |
| Table-driven tests + `t.Run` | `main_test.go` | One loop, many named cases |
| `cmd/` + `internal/` | layout | Binaries vs private library code |
| Linker-injected version | `var version` | The build stamps metadata into the binary |
| Errors are values | (from Phase 1) | `if err != nil { return err }`, no exceptions |

---

## 5. Why Phase 0 exists at all

Writing no features for a week can feel slow. It isn't:
- **CI from day one** means every future line is tested on three operating systems
  automatically.
- **Tests first** means a refactor in Phase 4 can't silently break Phase 1.
- **The tag** means you can always show "before vs after" in an interview.

---

## 6. Exercises (you write these, not me)

Do them on a new branch: `git switch -c phase-0-exercises`. Open a PR when you're done and
let CI judge it.

**E1. Test first: a `-v` alias for version.**
1. Add a row to the test table, `{"short version flag", []string{"-v"}, 0, "p2p dev", ""}`.
2. Run `go test ./...` and **watch it fail**. Read the failure message carefully.
3. Change `main.go` so it passes.

This red → green loop is test-driven development, and it's how we'll build the crypto.

**E2. A richer version line.**
- Make `p2p version` print `p2p dev (go1.27.2 windows/amd64)`. Use the `runtime` package:
  `runtime.Version()`, `runtime.GOOS`, `runtime.GOARCH`. Find them with `go doc runtime`.
- Update the test so it checks for `"go1."` but doesn't hardcode your OS. CI runs on three
  operating systems, so a hardcoded OS would fail on two of them.

**E3. Break it on purpose.**
- Change the unknown-command exit code from `2` to `0`, then run the tests.
- Which case fails, and what does the message say? Revert.
- Then delete the whole `default:` branch. What does `go vet` say? What does the test say?

**E4. Meet the race detector.** In a scratch folder *outside* the repo:
1. Create a module with `go mod init scratch`.
2. Write a program that starts 1000 goroutines, each doing `counter++` on a shared `int`, and
   waits for them all with `sync.WaitGroup`.
3. Print `counter`, and run it several times. Why isn't it always 1000?
4. Run `go run -race .`. You need gcc on Windows; if `-race` won't run, push the scratch code
   to a branch and let CI run it on Linux. Read the report.
5. Fix it twice, two ways:
   - with a `sync.Mutex`;
   - with `sync/atomic` (`atomic.Int64`).

   Which would you use for a byte counter in the relay, and why?

**E5. Explain the decisions.** Read ADR-0001 and ADR-0002. Then, in 3–4 sentences each in your
own words (just for you, not committed), answer:
- *Why Go?*
- *Why not libp2p?*

If you can't, tell me which part is unclear. That's the point of this exercise.

---

## 7. Quiz (answer out loud, then check)

<details><summary>1. What makes a Go file part of an executable rather than a library?</summary>

`package main` plus a `func main()`. Every other package name is a library that something
else imports.
</details>

<details><summary>2. Why does main() only call run()?</summary>

So the logic takes its dependencies (args, stdout, stderr) as parameters. Tests can call
`run` with fake input and in-memory buffers, with no process to start and no globals to
patch.
</details>

<details><summary>3. How does bytes.Buffer become usable where an io.Writer is expected, without declaring anything?</summary>

Go interfaces are satisfied implicitly (structural typing). `*bytes.Buffer` has a method
`Write([]byte) (int, error)`, so it *is* an `io.Writer`.
</details>

<details><summary>4. What's the difference between the `go` and `toolchain` lines in go.mod?</summary>

`go` is the minimum version required to build the module, and also the language version
used. `toolchain` is the preferred toolchain; an older local `go` command downloads and uses
it automatically.
</details>

<details><summary>5. How does the binary know its version if the source says "dev"?</summary>

The linker overwrites the package variable at build time:
`-ldflags "-X main.version=v0.1.0"`.
</details>

<details><summary>6. What does the race detector find, and why does it matter more in a P2P program?</summary>

It finds unsynchronised concurrent access to the same memory, from two goroutines with at
least one write. A P2P node runs many goroutines at once (one per peer, plus timers and the
DHT). Races there cause corruption that only shows up under load, which is impossible to
debug from logs.
</details>

<details><summary>7. Why pin GitHub Actions to a commit SHA instead of @v7?</summary>

Tags are mutable. If an action's repository is compromised, the attacker can move the tag to
malicious code, and every workflow using `@v7` runs it with your token. This happened to
`tj-actions/changed-files` in 2025. A commit SHA is immutable.
</details>

<details><summary>8. What do exit codes 0, 1 and 2 mean here, and who cares?</summary>

0 is success, 1 is a runtime failure, and 2 is a usage error. Shell scripts, CI and other
programs branch on them. Text output is for humans; exit codes are for machines.
</details>

<details><summary>9. Why did we delete the Python code instead of keeping it in a folder?</summary>

The annotated tag `v0-tracker` preserves it permanently and browsably. Keeping dead code on
`main` confuses readers and tools (linters, search, dependency scanners). Git history *is*
the archive.
</details>

<details><summary>10. Why open a pull request for your own solo project?</summary>

CI runs on the PR, so `main` only ever receives green code. Every phase also becomes a
reviewable unit with a description, which is the professional workflow interviewers expect
to see in your history.
</details>

<details><summary>11. Why does .gitattributes force LF line endings?</summary>

Windows editors write CRLF line endings. `gofmt` and the CI linters on Linux would then
report every file as unformatted, and diffs would show every line as changed.
`* text=auto eol=lf` stores LF in the repository on every platform.
</details>

---

**Next phase:** Phase 1, *bytes on the wire*. Why TCP is a stream and not a sequence of
messages (the old code's biggest bug), length-prefixed framing, Merkle trees, and your first
real transfer.

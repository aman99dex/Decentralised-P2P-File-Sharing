# Progress

## Current phase
**Phase 0: Go + groundwork.** The skeleton, CI and the `v0-tracker` tag are pushed on branch
`phase-0`.

## Next step
1. The owner merges the `phase-0` PR once CI is green.
2. The owner works through `docs/learn/phase-0-go.md` (the Go tour, exercises, quiz).
3. When the quiz is answered, Phase 0 is done → start Phase 1 (framing + Merkle).

## Open questions
- None. Schedule decided 2026-10-16: keep the full scope, about 25 weeks; v0.1 at about week 10.

## Log
- 2026-10-16: Read the old Python codebase. Interviewed the owner; confirmed the intent
  (`docs/intent/p2p-share.md`): Go, send/receive by code phrase with a P2P engine inside,
  relay anyone can run, teaching via ADR/LEARN/exercises/quiz, 2–3 months, ₹0–500/month.
  Go 1.27.0 is installed locally. Added the progress log.
- 2026-10-16: Researched crypto, NAT traversal, DHTs and tooling (`docs/research/`). Wrote SPEC.md and ADRs 0001–0008.
  An adversarial review found:
  - a CRITICAL gap: path traversal → new ADR-0009;
  - the symmetric SPAKE2 variant isn't buildable or testable → ADR-0003 switched to the
    asymmetric variant;
  - nonce reuse across QUIC streams, and lossy, unauthenticated path migration → ADR-0004
    revised;
  - squatting → ADR-0008 revised;
  - the wrong CVE explanation and tree shape → ADR-0006 switched to RFC 6962, resume by
    re-hashing;
  - unclear share links → SPEC §4.
  All fixed. Deleted the old plan.md.
- 2026-10-16: Phase 0 started.
  - Tagged the Python code `v0-tracker` and removed it from the branch.
  - Added the Go module (go 1.26, toolchain 1.27.2) and `cmd/p2p` with a testable `run()`.
  - CI: vet, race tests on 3 OSes, golangci-lint v2, govulncheck; actions pinned by SHA.
  - Added a Go `.gitignore`, `.gitattributes` (LF), MIT LICENSE and a new README.

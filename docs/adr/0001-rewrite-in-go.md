# ADR-0001: Rewrite in Go instead of evolving the Python code

- **Status:** Accepted, 2026-10-16

## Context
- The old code is a threaded Python tracker backed by MongoDB, which is not decentralised.
- Its framing is broken: a single `recv(1024)` reads each message, so messages split and merge.
- The sender loads whole files into memory.
- It has no encryption or integrity checks, and no tests.
- Almost nothing carries over into the target architecture.
- The product has to be installed by ordinary users on Windows, Mac and Linux.

## Options
| Option | For | Against |
|---|---|---|
| Keep Python (asyncio) | Continuity with the old code; fast to write | Users must install Python and its deps; distribution is weak; GIL limits for crypto/IO |
| **Go** | One static binary per OS; goroutines suit many connections; the space's tools (croc, Syncthing, libp2p, Tailscale) are in Go; small language | The owner is learning it fresh |
| Rust | Strongest hiring signal; no GC | Learning Rust *and* P2P networking at once risks stalling for months |

## Decision
Go. The old Python code is tagged `v0-tracker` and kept as the documented "before".

## Consequences
- Phase 0 includes learning Go.
- Cross-compiling with `CGO_ENABLED=0` is trivial.
- The README tells the v0 → v1 story: hybrid tracker → actually decentralised.

## What an interviewer might ask
- *Why not just fix the Python?*
  - The problems were architectural (a central tracker, no framing, no crypto), not cosmetic.
  - Distribution decided it: a single binary beats "first install Python 3.12".
- *Why not Rust?*
  - Learning one hard thing at a time. Go's simplicity left room to go deep on networking and
    crypto.

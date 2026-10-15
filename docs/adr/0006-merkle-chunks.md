# ADR-0006: 256 KiB chunks under an RFC 6962 Merkle tree; resume by re-hashing

- **Status:** Accepted, 2026-10-16. Revised after the adversarial review: corrected the CVE
  explanation, fixed the tree shape, defined the multi-file root, resume by re-hashing,
  limits.
- **Evidence:** `docs/research/crypto.md`, BEP 52, RFC 6962 §2.1

## Context
For a single sender, AEAD records plus a final hash would be enough; magic-wormhole works this
way. But we also want two things:
- **resume:** knowing which chunks already on disk are good;
- **swarm:** verifying chunks from *untrusted* peers (Phase 6).

Both need each chunk to be verifiable on its own, against one trusted root.

## Options
| Option | For | Against |
|---|---|---|
| A whole-file hash at the end | Simplest | Corruption is found only at the end; no resume; useless for a swarm |
| A flat list of chunk hashes (BEP 3) | Simple | The manifest grows linearly (100 GB needs 12.8 MB of hashes) |
| **Merkle tree, RFC 6962 shape** | One 32-byte root; a log₂N-hash proof for any chunk; a published, exact algorithm | A little more code |
| BLAKE3 / bao | Faster; verified streaming | An extra dependency; SHA-256 is fast enough at network speed |

## Decision
- **Chunks:** 256 KiB, except the last chunk of each file, which may be shorter.
- **Tree:** exactly RFC 6962 §2.1, `MTH`.
  - leaf = `SHA-256(0x00 ‖ chunk)`;
  - node = `SHA-256(0x01 ‖ L ‖ R)`;
  - the tree splits at the largest power of two below n. This is the published algorithm,
    not bottom-up promotion of odd nodes, so our roots are deterministic and checkable
    against known test vectors.
- **Several files:**
  - each file has its own root;
  - the **content root** = `MTH(file entries)`, where each entry is
    `SHA-256(0x02 ‖ len‖path ‖ size ‖ file_root)`, sorted by path;
  - so the names and sizes are authenticated too.
- **Verify before write:** a chunk is hashed and checked before it reaches its final
  position.
- **Resume by re-hashing.** On resume, the receiver hashes the chunks already on disk
  against the manifest's leaves and keeps the ones that match.
  - No state file can fall out of sync with the data, which survives crashes and torn writes.
  - Re-hashing is CPU work at GB/s, much faster than the network.
  - This replaces the old hardcoded JSON offset file.
- **Sending the leaves:** they're streamed in batches of records, not in one frame, so large
  transfers fit the 1 MiB frame limit.
- **Limits, enforced before anything is allocated:**
  - at most 100,000 files (configurable);
  - total size no more than the available disk space, and the receiver confirms the size
    before accepting;
  - a receiver bitmap whose length doesn't match the chunk count is rejected.

## Consequences
In Phases 1–4 the full list of leaves is sent. Per-chunk proofs only matter once peers are
untrusted (Phase 6).

## What an interviewer might ask
- *Why the 0x00 and 0x01 prefixes?* So a leaf hash can never equal an internal-node hash.
  Without them, an attacker could present an internal node's 64 bytes as if they were a
  leaf's data (a second-preimage attack on the tree).
- *What was Bitcoin's CVE-2012-2459?* Bitcoin pads odd levels by **duplicating the last
  node**, so `[a,b,c]` and `[a,b,c,c]` give the same root. RFC 6962's split rule never
  duplicates a node, so the attack can't happen here. This is a separate bug from the
  leaf/node confusion that the prefixes prevent.
- *Why re-hash on resume instead of saving progress?* A saved progress file can claim chunks
  are good after a crash tore them. Re-hashing checks the truth on disk.

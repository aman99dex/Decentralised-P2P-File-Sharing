# ADR-0004: Our own end-to-end record layer, independent of the transport

- **Status:** Accepted, 2026-10-16. Revised after the adversarial review: one stream per path,
  path confirmation, migration, encrypted frame types.
- **Evidence:** `docs/research/crypto.md`, RFC 9266

## Context
After SPAKE2 both peers share a key `Ke`. Bytes travel either over a direct QUIC connection or
through a relay that splices two QUIC connections together. QUIC already encrypts with
TLS 1.3, but that TLS uses throwaway self-signed certificates:
- on its own it authenticates nobody;
- on the relay path it ends **at the relay**.

## Options
| Option | For | Against |
|---|---|---|
| **Own AEAD record layer on top of any stream** | One security story for every path; the relay is untrusted by construction (magic-wormhole's approach) | Data is encrypted twice on the QUIC path (cheap: ChaCha20 runs at GB/s) |
| Bind the PAKE to QUIC's TLS via `tls-exporter` (RFC 9266) | Elegant; one encryption layer | Only covers the direct path; the relay path would need a second scheme |
| Noise NNpsk0 with `Ke` as the PSK | Forward secrecy; a known framework | flynn/noise hasn't been released since 2024; adds a dependency for little gain |

## Decision
1. **Exactly one bidirectional QUIC stream per path.** No multiplexing inside a session.
   - Control messages and chunk data are interleaved records on that one stream.
   - So each direction has exactly one counter, and ordering is well defined.
2. **`path_id`:** a counter assigned by the sender, starting at 0 for the relayed path and
   increasing for each new path. It **never repeats within a session.**
3. **Keys:** `k_dir = HKDF-SHA256(Ke, info = "p2p/1 rec" ‖ dir ‖ path_id)`, where `dir` is
   `s2r` or `r2s`.
4. **Records:**
   - on the wire: `uint32 length ‖ ciphertext`;
   - the plaintext is `type ‖ payload`, so **the frame type is encrypted too**;
   - the AEAD is ChaCha20-Poly1305 with a 96-bit counter nonce, and `aad = length`;
   - a record that fails to decrypt or arrives out of sequence kills the path.
5. **Path confirmation.** A new path is "up" only after both sides exchange
   `MAC(Ke-derived key, "path-ok" ‖ path_id ‖ TT)` **on that path**.
   - A QUIC handshake completing proves nothing: a malicious relay knows both public
     addresses and could dial in first.
6. **Migration from the relay to the direct path:**
   - the sender sends `MIGRATE(path_id_new, next_chunk)` on the old path, then `END`;
   - the receiver drains the old path up to `END`, then continues on the new path.
   - Nothing is lost or duplicated, because the chunk index carries the position.
7. **Completion.** The transfer is complete only when the receiver sends `DONE` with a
   **fully verified bitmap** (ADR-0006). A closed stream without `END` counts as a failure,
   never as success.

## Consequences
- The relay, DHT nodes and QUIC's TLS are all outside our trust boundary.
- The relay sees only record lengths and timing, not which records are control and which are
  data.
- Filenames and sizes travel inside the records (we learned this from croc's leaks).

## What an interviewer might ask
- *Why separate keys per direction, and per path?* Each (key, nonce) pair must be used once
  only. Two directions, or two paths, each starting their counter at 0 under one key would
  reuse nonces. With ChaCha20 that leaks the XOR of the two plaintexts, and it breaks
  Poly1305's authentication.
- *Why one stream rather than QUIC's multiplexing?* Several streams under one key either
  reuse nonces (each stream has its own counter) or deliver records out of order (shared
  counter). One stream per path keeps the security argument simple. Throughput is still
  bounded by the link, not by the stream.
- *How do you switch paths without losing data?* `MIGRATE`, then drain to `END`, then resume
  on the new path from a chunk index.

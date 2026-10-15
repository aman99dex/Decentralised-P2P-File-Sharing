# Research: PAKE, encrypted transport, code phrase, integrity (checked 2026-10-16)

## Versions
- **Go:** 1.27.2
- **`golang.org/x/crypto`:** v0.58.0 (2026-10-09)
- **croc:** v11.5.4
- **magic-wormhole:** 0.24.0 (2026-05)
- **wormhole-william (Go):** v1.0.8 (2025-03)

## PAKE
- **Researcher's pick: symmetric SPAKE2 (RFC 9382) over edwards25519, with RFC 9382 key confirmation.**
  - magic-wormhole uses SPAKE2, so this is interoperable with it.
  - RFC 9382 requires both sides to check confirmation MACs.
  - It requires M=N (the symmetric variant) when the two sides don't have fixed roles.
  - https://www.rfc-editor.org/rfc/rfc9382.html
- **CPace:** draft-irtf-cfrg-cpace-21, in the RFC Editor queue but not yet an RFC. Cleaner design. Its only Go library is `filippo.io/cpace` (2021, "weekend project", early draft).
- **OPAQUE:** RFC 9807. A client/server PAKE with a registration step, so the wrong shape for two peers sharing a one-time code.
- **croc:**
  - Uses `schollz/pake/v3`, a non-standard PAKE2 on the `siec` curve with no built-in key confirmation.
  - **History:** in v8, a rogue receiver could brute-force the PAKE (found by RedRocket, fixed in v9).
  - CVE-2023-43618: local IPs leaked in cleartext.
  - CVE-2023-43621: the code was visible on the command line, so other users could see it in the process list.
- **Go libraries, none audited:**
  - `salsa.debian.org/vasudev/gospake2` (2021, stale). wormhole-william uses it.
  - `github.com/bytemare/opaque` v0.18.0.
- **Stdlib:** has no PAKE. It does have `crypto/hkdf` (since Go 1.24).

## Transport after the PAKE
- **Researcher's pick: the wormhole approach.**
  - Use HKDF to derive **separate sender→receiver and receiver→sender keys**.
  - Encrypt length-prefixed records with secretbox or chacha20poly1305, using **counter nonces**.
  - Drop the connection on any out-of-order record.
  - https://magic-wormhole.readthedocs.io/en/latest/transit.html
- **Alternatives:**
  - Noise NNpsk0 with the PAKE output as the PSK. The last flynn/noise release was v1.1.0 (2024-02), and PSK patterns are unverified.
  - QUIC/TLS with PAKE-derived auth. Go's TLS doesn't support external PSKs (unverified).
- **croc:** AES-GCM with random nonces. Keys come from PBKDF2 at only **100 iterations**, a weakness.

## Code phrase
- **Wormhole's scheme:**
  - A code is a nameplate number plus 2 PGP words, which gives 16 bits, so one guess has a 1/65536 chance.
  - Words alternate between two lists, so a dropped word is detected.
  - A mailbox with more than 2 sides is "crowded".
  - A failed key confirmation makes the client release the nameplate and mailbox and stop.
  - https://magic-wormhole.readthedocs.io/en/latest/attacks.html
- **Researcher's pick:** 3 PGP words (24 bits), and a code is **single-use** on the server.

## Integrity
- **Wormhole:** each AEAD record is authenticated, plus a final whole-file SHA-256 ack.
- **Reference chunk sizes:**
  - BitTorrent v2: SHA-256 Merkle tree over 16 KiB blocks (BEP 52)
  - IPFS: 256 KiB chunks
  - croc: 64 KiB buffer
- **BLAKE3 libraries:** `lukechampine.com/blake3` v1.4.1 (includes `bao` verified streaming), `zeebo/blake3` v0.2.4.

## Pitfalls a reviewer would flag
- **Nonce reuse** when one key and counter are used in both directions → use per-direction keys.
- **No key confirmation.** Send no metadata until the confirmation check passes.
- **Acting on peer data before the key is confirmed.** This is how croc v8 was attacked.
- **Downgrades.** Bind version/capabilities into the authenticated transcript, and never fall back to plaintext.
- **Metadata leaks.** Encrypt filenames and sizes. The relay still sees IPs, timing and lengths.
- **Code on the command line.** Read it from a prompt or env var instead.
- **Low-entropy code straight into a KDF** (croc's PBKDF2). Only the PAKE output may be key material.

## Unverified
- flynn/noise PSK patterns.
- Go TLS external PSK.
- The EFF wordlist.
- The full contents of x/crypto.

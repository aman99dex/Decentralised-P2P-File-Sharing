# Research: DHT, libp2p vs our own, swarm (checked 2026-10-16)

## go-libp2p vs our own Kademlia
- **go-libp2p:** v0.50.0 (2026-09-21), go-libp2p-kad-dht v0.42.2.
  - Still 0.x, with frequent breaking changes: 8 minor releases in about 13 months.
- **Maintenance risk:**
  - Interplanetary Shipyard handed go-libp2p to the community after 2025-09-30.
  - Shipyard wound down its IPFS work on 2026-09-30 after Protocol Labs funding ended.
  - https://cointelegraph.com/news/shipyard-winds-down-ipfs-protocol-labs-funding
  - Who funds the current maintainers is **unverified**.
- **Weight (measured locally, windows/amd64):**
  - libp2p + DHT: **28 MB stripped, 248 modules**.
  - quic-go + noise: **6.4 MB, 21 modules**.
- **What it hides:** the handshake, multiplexing, NAT traversal, client/server DHT mode and diversity filters. Those are exactly the parts we want to learn.
- **Recommendation:** our own Kademlia and wire protocol on top of quic-go. Read go-libp2p-kad-dht, don't import it.

## Kademlia essentials (paper: https://www.scs.stanford.edu/~dm/home/papers/kpos.pdf)
- **Parameters:**
  - XOR metric.
  - k-buckets ordered least-recently-seen. When a bucket is full, ping the oldest and keep it if it answers.
  - k=20, α=3.
  - Refresh idle buckets hourly; republish hourly; values expire after 24 h.
- **S/Kademlia hardening:**
  - Node ID = hash of the public key, plus a crypto puzzle. This raises the cost of Sybil attacks but doesn't stop them.
  - d disjoint lookup paths.
  - Signed records.
- **Mainline DHT:**
  - BEP 5: k=8; announces need a token tied to the IP.
  - BEP 44: ed25519-signed mutable items with `seq`/`salt`/`cas`, values up to 1000 bytes, about 2 h expiry.
  - BEP 42: node ID tied to the IP.
- **IPFS DHT:**
  - k=20, α=10.
  - Provider records expire after 48 h and are republished every 22 h.
  - **Client/server mode:** only publicly reachable nodes enter routing tables.
  - https://github.com/libp2p/specs/blob/master/kad-dht/README.md
- **Pitfalls and fixes:**
  - **Poisoning:** only insert nodes that answered you directly.
  - **NAT pollution:** use client/server mode.
  - **Eclipse:** cap peers per /24 subnet, prefer long-lived nodes, use disjoint lookups.
  - **Records:** sign and validate them.

## Code phrase and the DHT
- **DHT key = H("app" || nameplate) only.** Never derive it from the secret words.
  - If the key were H(full code), anyone could enumerate all 2^16 codes and run a full man-in-the-middle attack.
  - Any DHT record holding a MAC or ciphertext derived from the code allows offline brute force.
- The words feed only the online PAKE, which aborts after one failure.
- **Risks:** nodes near the key see IP addresses and can censor the record; nameplates can collide or be squatted.

## Minimal correct swarm (BEP 3 / BEP 52, Cohen 2003)
- Verify every piece's hash before writing; ban any peer that sends a bad piece.
- **Piece selection:**
  - Random first piece, then rarest-first.
  - Finish pieces already started before starting new ones.
- **Pipelining:** several outstanding block requests per peer.
- **Endgame:** request the remaining blocks from every peer, and cancel as they arrive.
- **Skip:** choking and tit-for-tat while peers aren't untrusted public strangers.

## Go code to read
| Repo | Covers | Status |
|---|---|---|
| `anacrolix/dht` | BEP 5 + 44 | v2.24.0, maintained |
| `anacrolix/torrent` | piece picker, endgame, v2 hashing | v1.61.0, maintained |
| `nictuku/dht` | small and readable | unmaintained since 2020, read it first |
| `go-libp2p-kad-dht` | client/server mode, providers, diversity filter | maintained |

## Unverified
- Who funds the libp2p maintainers.
- Cohen's exact pipeline depth.

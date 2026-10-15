# SPEC: p2p, encrypted peer-to-peer file sending

The intent is in `docs/intent/p2p-share.md`, the evidence in `docs/research/`, and the
reasoning behind each decision in `docs/adr/`. This file says **what** we build and **in what
order**.

## 1. The product in one screen

```
$ p2p send ./holiday-photos
Sending 'holiday-photos' (412 MB, 38 files)
Code: 4-tiger-orbit-quantum          ← read this to your friend
Waiting for receiver…
Connected (direct, 31 ms) ▕██████████████▏ 412 MB  58 MB/s  done ✓

$ p2p receive
Code: 4-tiger-orbit-quantum          ← typed at a prompt, never in argv (ADR-0008)
Receive 'holiday-photos' (412 MB, 38 files) from peer? [y/N] y
▕██████████████▏ verified ✓  saved to ./holiday-photos
```

Other commands:
- `p2p share <path>` prints a link such as `p2p://<root>#<key>`. Anyone holding the link
  downloads from every peer that has the file (swarm, Phase 6).
- `p2p get <link>` downloads from such a link.
- `p2p relay` runs the public bootstrap/relay/DHT node.
- `p2p version`.

## 2. Guarantees (what we promise, and what we test)

1. **Confidentiality and integrity, end to end.** Only the two code holders can read or
   change the content, filenames or sizes. Relays, DHT nodes and network observers see
   ciphertext, IP addresses, timing and lengths only.
2. **One guess per attempt.** An active attacker gets at most one online guess at the code,
   about a 1-in-16.7M chance (24 bits). A failed key confirmation aborts both sides. Nobody
   can brute-force the code offline.
3. **Every byte is verified.** Each chunk is checked against a Merkle root that has been
   authenticated, before it is written to disk. A corrupt or malicious chunk is rejected and
   fetched again.
4. **Resumable.** An interrupted transfer continues from the chunks already verified on disk.
5. **No required central server.** The binary is the same everywhere, any public node can be
   a relay and DHT node, and LAN transfers work with zero infrastructure (mDNS).
6. **A hostile sender can't hurt the receiver.** Names are sanitised and confined to the
   destination folder, nothing is overwritten without asking, there are no symlinks, and
   sizes and file counts are capped (ADR-0009, ADR-0006).

**Non-goals:** anonymity (we don't hide IP addresses; use Tor for that); a GUI, mobile or
browser client; accounts; tokens.

## 3. Architecture

```
            cmd/p2p  (send · receive · share · get · relay)
                 │
   ┌─────────────┼──────────────────────────────────────────┐
   │ session     │ the PAKE handshake → E2E record layer →   │  internal/session
   │             │ offer/accept → transfer                   │
   ├─────────────┼──────────────────────────────────────────┤
   │ spake2      │ RFC 9382, P-256 (filippo.io/nistec)       │  internal/spake2
   │ wordcode    │ code phrase ↔ nameplate + secret words    │  internal/wordcode
   │ secure      │ HKDF per-direction keys, ChaCha20-Poly1305│  internal/secure
   │             │ records, counter nonces                   │
   ├─────────────┼──────────────────────────────────────────┤
   │ manifest    │ 256 KiB chunks, SHA-256 Merkle tree       │  internal/manifest
   │             │ (RFC 6962 style), resume bitmap           │
   │ swarm       │ rarest-first, pipelining, endgame (Ph 6)  │  internal/swarm
   ├─────────────┼──────────────────────────────────────────┤
   │ conn        │ ONE UDP socket → quic-go Transport;       │  internal/conn
   │             │ hole punch + relay race; mDNS on LAN      │
   │ dht         │ Kademlia over QUIC, k=20 α=3, client/     │  internal/dht
   │             │ server mode, signed records (Ph 5)        │
   │ relay       │ rendezvous + address echo + blind stream  │  internal/relay
   │             │ splice + abuse limits                     │
   └─────────────┴──────────────────────────────────────────┘
```

**Layering rule:** everything below `secure` is treated as hostile. QUIC's TLS is there only
so the transport works; our trust comes from the PAKE (ADR-0004).

### Connection flow (`send` / `receive`, from Phase 4)
1. **Sender:**
   - opens its UDP socket and connects to a relay over QUIC;
   - the relay tells it its observed public address;
   - the relay allocates a nameplate (Phases 3–4). From Phase 5 the sender instead publishes
     a signed rendezvous record in the DHT, keyed by `H(nameplate)`.
2. **Sender** prints the code `<nameplate>-<w1>-<w2>-<w3>`.
3. **Receiver:**
   - parses the code;
   - finds the sender's relay (the shared default relay in Phases 3–4; from Phase 5, a DHT
     lookup on the nameplate);
   - claims the nameplate. The relay splices the two peers' streams together.
4. **Both peers, over the relayed stream:**
   - run SPAKE2 and confirm the key;
   - derive the record keys;
   - from here on everything is end-to-end encrypted.
5. **Inside the encrypted channel:**
   - the peers exchange address candidates (LAN, observed IPv4, global IPv6);
   - they run a DCUtR-style synchronised hole punch (up to 3 attempts);
   - a punched path is used only after **path confirmation** (a MAC bound to the transcript,
     sent on that path). The transfer then **migrates** (`MIGRATE`, drain to `END`) with
     fresh keys. Otherwise it stays relayed (ADR-0004, ADR-0005).
6. **Transfer:**
   - the sender sends an offer (manifest: names, sizes, Merkle roots), encrypted;
   - the receiver validates the offer against the limits and `safePath`, then the user
     accepts;
   - the receiver re-hashes any partial files and sends its "have" bitmap;
   - the sender streams the missing chunks; the receiver verifies each one before writing;
   - the receiver sends `DONE` with the full bitmap, and only then do both sides report
     success.

**LAN with no internet:** the sender also advertises `_p2p._udp` over mDNS, keyed by
`H(nameplate)`, so the receiver finds it locally and skips step 3's relay.

## 4. Wire formats (v1)

- **ALPN** (the protocol name negotiated at connect time): `p2p/1`. Unknown versions are
  rejected; there is no plaintext fallback.
- **Framing:** each frame is `uint32 BE length | ciphertext`, with a maximum of 1 MiB.
  Exceeding the maximum closes the connection. The plaintext is `uint8 type | payload`, so
  even the message type is hidden from the relay.
- **Payloads:** control messages are JSON; chunk data is raw binary.
- **Record layer:** `ChaCha20-Poly1305(k_dir_path, nonce = 96-bit counter, aad = length)`.
  - There is one QUIC stream per path, and fresh keys per path.
  - The last record is an explicit `END`.
  - See ADR-0004.
- **Merkle tree:**
  - RFC 6962 `MTH`, with leaf `0x00`, node `0x01`;
  - chunk size is 256 KiB;
  - the content root covers the per-file roots plus names and sizes (ADR-0006).
- **Share links (Phase 6):**
  - the format is `p2p:<root>.<key>`;
  - the key is a **random 256-bit key per share**, so content isn't convergent and nobody
    can confirm a file's presence by computing its root;
  - chunks are encrypted with nonce = chunk index;
  - **the root is over the ciphertext**, so seeders verify chunks without being able to
    read them;
  - the link is a bearer token: anyone who has it can read the file, and it can't be
    revoked. The CLI says so when it prints the link.

## 5. Phases

Each phase ships something runnable. Each one ends with its ADRs, `docs/learn/phase-N-*.md`,
your exercises and the quiz. Weeks assume about 10–12 hours a week.

| # | Phase | You'll be able to… | Ships | Weeks |
|---|---|---|---|---|
| 0 | **Go + groundwork:** the Go tour, module layout, CI (vet, race, lint, govulncheck), tag `v0-tracker` | write, test and lint Go | skeleton binary | 1.5 |
| 1 | **Bytes on the wire:** framing, manifest + Merkle, chunk store, resume; a plain TCP `--direct host:port` dev mode | explain why TCP needs framing and how Merkle proofs work | LAN transfer (dev only) | 2.5 |
| 2 | **Crypto:** SPAKE2 against the RFC 9382 vectors, wordcode, HKDF, record layer, fuzz tests | explain PAKE, AEAD, nonces, key confirmation | encrypted transfer | 3 |
| 3 | **Relay + first release:** QUIC, relay rendezvous + splice, abuse limits, GoReleaser, cosign, brew/scoop, deploy to Oracle | ship and operate a service | **v0.1, usable over the internet** | 3 |
| 4 | **Direct paths:** a **docker NAT lab** (cone, restricted and symmetric NATs, netem) built first; then address echo, candidates, hole punch + relay race, path confirmation + migration, mDNS | explain NAT types and why punching works | v0.2 | 5 |
| 5 | **Kademlia:** routing table, iterative lookups, signed records, client/server mode; DHT rendezvous across many relays | explain DHTs, Sybil and eclipse attacks | v0.3, no single relay required | 5 |
| 6 | **Swarm:** `share`/`get`, encrypted links, provider records, rarest-first, endgame | explain BitTorrent's design choices | v0.4 | 3 |
| 7 | **Proof:** 50-node docker + netem simulation, churn, corruption injection, benchmark table, README | defend the numbers | v1.0 | 2 |

Total: about **25 weeks (about 6 months)**, revised after the review. **v0.1 is deployed
and usable over the internet at about week 10.** If time runs short, cut Phase 6 first;
the product stands without it.

## 6. Benchmarks to fill in (Phase 7)

| Claim | Target | Measured |
|---|---|---|
| Nodes in the simulated network | 50 | |
| Median DHT lookup hops at 50 nodes | ≤ log₂N ≈ 6 | |
| Hole-punch success in the NAT lab, **per NAT type** | report it, with real-world context: libp2p measured 70% ± 7.1% | |
| Throughput: direct vs relayed (same link) | report both | |
| Swarm, 4 seeders vs 1 | report the speed-up | |
| Corrupt chunks detected and refetched | 100% | |
| Transfers completing under 30% churn (swarm) | 100% | |

## 7. Production checklist (by v1.0)

- **Releases:** signed (cosign keyless + GitHub attestations); `go install`, brew cask, scoop.
- **CI:** `go vet`, `go test -race`, golangci-lint v2, govulncheck, fuzzing in nightly CI.
  Actions are pinned by SHA.
- **Relay limits:**
  - per-IP concurrency;
  - accept-rate token bucket;
  - 10-minute unpaired timeout;
  - per-session byte and time caps;
  - a daily egress budget;
  - an optional relay password.
- **Relay operations:** systemd, `/metrics` + `/healthz` on a private port, slog JSON logs,
  graceful drain on SIGTERM.
- **Docs:** a threat model in the README, known limitations (no anonymity; symmetric NAT and
  CGNAT fall back to the relay; idle reclaim on Oracle's free tier), a runbook for the relay.

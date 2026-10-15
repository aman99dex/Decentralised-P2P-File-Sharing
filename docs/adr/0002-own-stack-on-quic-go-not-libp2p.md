# ADR-0002: Build our own P2P stack on quic-go, not go-libp2p

- **Status:** Accepted, 2026-10-16
- **Evidence:** `docs/research/dht.md`, `docs/research/nat-traversal.md`

## Context
We need a transport, NAT traversal, relaying and a DHT. go-libp2p provides all of them.

## Options
| Option | For | Against |
|---|---|---|
| go-libp2p + kad-dht | Batteries included and battle-tested | 28 MB binary, 248 modules (measured); 0.x with frequent breaking changes; maintenance uncertain since Shipyard wound down (2026-09); hides exactly the parts we're meant to learn |
| Everything from scratch, including crypto and congestion control | Maximum learning | Not production-grade; we would get crypto and congestion control wrong |
| **Own protocol + Kademlia on quic-go** | quic-go (v0.63, monthly releases) gives TLS 1.3, streams and congestion control, and can run on our own UDP socket; about 6 MB binary | We write and test hole punching, relaying and the DHT ourselves |

## Decision
Use quic-go as the transport. Write our own relay, hole punching and Kademlia, and read
go-libp2p-kad-dht, anacrolix/dht and the libp2p specs (DCUtR, Circuit Relay v2) as reference
designs.

## Consequences
- More code to own, but every line is explainable.
- We copy proven designs (DCUtR timing, client/server DHT mode, relay limits) rather than
  inventing our own.
- quic-go supports the last two Go releases, so we have to keep Go current.

## What an interviewer might ask
- *Isn't this not-invented-here?*
  - We reuse what's hard and well understood (QUIC, TLS, AEAD, P-256 arithmetic).
  - We build what's specific to our product (rendezvous, punching, DHT records), following
    published specs.
- *Why QUIC rather than TCP?*
  - It runs over UDP, which hole-punches far more easily.
  - It multiplexes streams without head-of-line blocking.
  - One UDP socket can carry the relay connection, the address probe and the peer connection,
    and that's what makes hole punching work (ADR-0005).

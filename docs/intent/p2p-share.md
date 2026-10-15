# Intent: P2P encrypted file sharing (confirmed 2026-10-16)

- **Outcome:** Go CLI — `p2p send <path>` prints a code phrase; `p2p receive` (code typed at a
  prompt, see ADR-0008) pulls the files directly from the sender. End-to-end encrypted,
  resumable, verified per chunk.
- **Engine:** PAKE from the code phrase, an end-to-end encrypted record layer (ADR-0004
  replaced the original "Noise" idea), content-addressed chunks + Merkle root, NAT hole punching with relay fallback, Kademlia DHT for peer discovery,
  multi-peer (swarm) downloads.
- **Runs on:** users' own computers, one binary for every role. Works on a LAN with zero
  infrastructure. Across the internet, `p2p relay` (bootstrap + relay) only ever sees
  encrypted bytes; a default one is hosted, and anyone can run their own.
- **Deployed means:** that default relay on a cheap or free VM, signed cross-platform
  releases (brew / scoop / go install), CI.
- **How it's learned:** phase by phase. Each phase has ADRs, a LEARN.md, exercises the
  owner writes, and an interview quiz. Go is learned from scratch along the way.
- **Success:** genuinely decentralised, real people can install and use it, benchmark
  numbers in the README, and the owner can defend every line in an interview.
- **Constraints:** about 6 months part-time at 10–12 h/week (extended from 2–3 months after the
  review, to keep the full scope); v0.1 is deployed at about week 10; ₹0–500/month hosting.
- **Out of scope:** chat room, user accounts, MongoDB, GUI or mobile app, browser support,
  tokens or blockchain. The old Python code is tagged `v0-tracker` as the "before".

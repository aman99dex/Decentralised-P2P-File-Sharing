# p2p

Send files directly between two computers, end-to-end encrypted, with no account and no
server that can read your data.

```
$ p2p send ./photos                 $ p2p receive
Code: 4-tiger-orbit-quantum         Code: 4-tiger-orbit-quantum
                                    ▕██████████████▏ verified ✓
```

> **Status: under construction (Phase 0 of 8).** The roadmap is in [SPEC.md](SPEC.md) §5.

## How it works (the short version)

- **Short code → strong key.** SPAKE2 (RFC 9382), a password-authenticated key exchange. An
  attacker gets one online guess, and nothing to brute-force offline.
- **End-to-end encryption.** ChaCha20-Poly1305 records with separate keys per direction and
  per path. Relays only ever see ciphertext.
- **Direct when possible.** UDP hole punching over QUIC, with an encrypted relay as the
  fallback.
- **Every byte verified.** Files are split into 256 KiB chunks under an RFC 6962 Merkle root,
  so transfers resume and corruption is caught.
- **No central server (from v0.3).** A Kademlia DHT lets peers find each other through any
  relay. Anyone can run one with `p2p relay`.

Design decisions and the trade-offs behind them are in [docs/adr/](docs/adr/).

## Build

```
go build ./cmd/p2p
./p2p version
```

## History

`v0-tracker` is the original version of this repository. It was a Python LAN file share with
a central tracker and MongoDB, modelled on DC++. This rewrite replaces the tracker with a
genuinely decentralised design. Compare the two in
[the v0 tree](https://github.com/aman99dex/Decentralised-P2P-File-Sharing/tree/v0-tracker).

## License

MIT

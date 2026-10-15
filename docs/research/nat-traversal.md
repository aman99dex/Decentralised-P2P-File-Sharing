# Research: NAT traversal & relay (checked 2026-10-16)

## How existing tools do it
- **croc v11.5.4** (2026-09-26)
  - PAKE code phrase.
  - `--transport auto` sets up "Tailcat" (WireGuard over Tailscale userspace). It starts on DERP and upgrades to direct UDP.
  - It races croc's TCP relay (ports 9009-9013) as a fallback.
  - The relay can be password-protected (`CROC_PASS`).
  - https://github.com/schollz/croc/releases
- **magic-wormhole Transit**
  - TCP only: both sides dial each other's IP:port hints.
  - After a few seconds both go to the relay with `please relay <HKDF token>`, and the relay pairs them and forwards blindly.
  - **No hole punching.**
  - https://magic-wormhole.readthedocs.io/en/latest/transit.html
- **magic-wormhole Dilation**
  - Durable, multiplexed connections keyed with Noise NNpsk0 from the code.
  - Still TCP plus UPnP.
  - https://magic-wormhole.readthedocs.io/en/latest/dilation-protocol.html
- **Tailscale**
  - Every connection starts relayed through DERP, which also carries the "disco" probes. It then upgrades to direct UDP, then a peer relay, otherwise stays on DERP.
  - https://tailscale.com/blog/how-nat-traversal-works (**read this one in full**)
- **libp2p (go-libp2p v0.50.0)**
  - Circuit Relay v2: reservations with duration/data limits.
  - DCUtR: CONNECT/SYNC over the relay, then a half-RTT-timed simultaneous dial, up to 3 tries.
  - AutoNAT v2: reachability checks via dial-back, with anti-amplification dial data.
  - https://github.com/libp2p/specs/blob/master/relay/DCUtR.md

## Published hole-punch success rates
- **libp2p (IMC '26, Trautwein et al.):** **70% ± 7.1%** across 4.4M attempts and 85k networks.
  - TCP and QUIC were about equal.
  - 97.6% of successes came on the first attempt.
  - The figure is conditional on the relay reservation having succeeded first.
  - https://arxiv.org/html/2510.27500v1
- **ProbeLab (2026-08-03):** port-restricted cone NAT 82.9%, symmetric NAT 39.7% (the authors doubt this figure).
  - About 36% of NATed peers never got a relay reservation.
  - https://probelab.io/blog/can-libp2p-punch-through-nats/
- **Tailscale:** "over 90%" direct. This is an estimate, not a measurement.
- **Expected relay share for us: about 10-30%** of transfers (inferred).

## Transport facts
- UDP or TCP doesn't change punch success (libp2p measured them as equal). UDP with QUIC on top is simpler than TCP simultaneous open.
- **quic-go:** `github.com/quic-go/quic-go` v0.63.0 (2026-09-22). Released about monthly. Its `Transport` can run on our own `net.PacketConn`.
- **No STUN server needed:** the relay echoes back the UDP address it observes.
  - It **must be the same socket** the transfer uses, because each socket gets its own NAT mapping.
- **pion:** `pion/ice/v4` v4.4.7 is usable standalone, but we would write the signaling ourselves. v5 is at rc.
- **Birthday-paradox port prediction:**
  - One hard NAT: 256 probes give 64%.
  - Both sides hard: about 170k probes and about 28 minutes.
  - Can exhaust router session tables.
  - **Not worth it.**
- **CGNAT:** double NAT, usually ends up on the relay.
- **IPv6:** about 50% of Google's traffic (2026). Firewalls still need a simultaneous open.

## Relay abuse references
- **go-libp2p relay defaults:**
  - circuits of 2 min / 128 KB
  - 8 reservations per IP, 32 per ASN, 128 in total
- **derper:** rate-limits inbound only (never STUN), with `--verify-clients`.

## mDNS libraries
| Module | Latest | Status |
|---|---|---|
| `github.com/pion/mdns/v2` | v2.2.2 (2026-10-01) | **Active**, has DNS-SD |
| `github.com/hashicorp/mdns` | v1.0.7 (2026-06) | Slow but alive |
| `github.com/grandcat/zeroconf` | v1.0.0 (2020) | Stale |
| `github.com/libp2p/zeroconf/v2` | v2.2.0 (2022) | Stale |

## Recommended design (researcher's proposal, to be confirmed in an ADR)
1. **Rendezvous:** TCP+TLS to `p2p relay`, with sessions keyed by a hash of the code (like wormhole).
2. **One UDP socket per peer**, shared by the quic-go `Transport` and the probe packets.
3. **Observed address:** the relay echoes each peer's observed UDP address. No STUN server, no pion.
4. **Candidates:** LAN IPs, the relay-observed IPv4 and global IPv6, exchanged via the relay.
5. **Punching:** a DCUtR-style half-RTT SYNC, then spray to every candidate and dial QUIC. Up to 3 attempts.
6. **Race the relay path from the start** and switch over if the punch wins (as croc and Tailscale do).
7. **Relay fallback:** a blind TCP pipe paired by token, carrying data that is already end-to-end encrypted.
8. **Skip:** birthday probing, TCP simultaneous open, UPnP/PCP, TURN.
9. **Relay limits:**
   - per-IP concurrency
   - accept token bucket (`x/time/rate`)
   - 60 s unpaired timeout
   - per-session byte and time caps
   - idle timeout
   - global daily egress budget
   - optional relay password
10. **mDNS:** skip it in v1, since step 4 covers the LAN. Later, `pion/mdns/v2`.

## Unverified
- Tailscale's direct vs relayed percentages.
- quic-go's maintainer and funding.
- The `x/time` version.
- Whether pion/stun v4 includes a server.

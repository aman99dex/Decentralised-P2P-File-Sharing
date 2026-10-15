# ADR-0005: NAT traversal: one UDP socket, relay address echo, DCUtR-style punch, relay race

- **Status:** Accepted, 2026-10-16
- **Evidence:** `docs/research/nat-traversal.md`

## Context
Most home computers sit behind a NAT that drops unsolicited inbound packets. The published
data for punching success:
- libp2p measured **70% ± 7.1%** across 4.4M attempts (IMC '26).
- Symmetric NATs and CGNAT mostly fail.

So we need direct connections where possible and a relay for the rest (an expected 10–30% of
transfers).

## Decision
1. **One UDP socket per process**, wrapped in a quic-go `Transport`. The relay connection,
   the address probe and the peer connections all share it. A NAT mapping belongs to a
   socket, so the public address the relay sees is the one the peer has to hit.
2. **The relay echoes the observed address** (`ip:port`) back to the client. This replaces a
   separate STUN server (about 30 lines of code).
3. **Candidates:**
   - LAN addresses;
   - the observed IPv4 address;
   - global IPv6 addresses.

   They're exchanged *inside the end-to-end channel* (ADR-0004), so the relay never sees our
   LAN IPs. croc leaked them in CVE-2023-43618.
4. **Punch, DCUtR-style:**
   - measure the RTT over the relayed path;
   - send `SYNC` and wait RTT/2, so both sides fire together;
   - both send UDP packets to every candidate and dial QUIC;
   - up to 3 attempts.
5. **Race, don't wait.** The transfer starts on the relay immediately. When the punch
   succeeds, the direct path must pass **path confirmation** (a MAC bound to the PAKE
   transcript, sent on that path) before any data moves. A completed QUIC handshake alone
   proves nothing: the relay knows both public addresses and could dial in first. Then the
   transfer migrates without loss (ADR-0004 §6) and gets fresh keys.
6. **Built-in NAT lab (Phase 4, not Phase 7).** Linux network namespaces plus `iptables`
   rules that emulate full-cone, port-restricted and symmetric NAT, plus `tc netem`, all in
   docker. Punching can't be developed without reproducible NATs.
7. **mDNS on the LAN** (`pion/mdns/v2`), so two machines on the same Wi-Fi work with no
   internet.
8. **Skipped, deliberately:**
   - birthday-paradox port prediction (about 170k probes when both NATs are hard, and it can
     exhaust router session tables);
   - TCP simultaneous open (no better success rate than UDP);
   - UPnP/PCP and TURN. Revisit if the benchmarks show a large relayed share.

## Consequences
- Users never wait on a punch that's going to fail.
- The relay carries the hard 10–30% of transfers, so it needs abuse limits and a bandwidth
  budget (ADR-0007, SPEC §7).
- UDP-blocking networks (some corporate ones) are a known limitation in v1. A TCP+TLS relay
  fallback can be added later.

## What an interviewer might ask
- *Why does hole punching work at all?* A NAT accepts inbound packets from an address you've
  recently sent to. If both sides send at the same moment, each side's outbound packet opens
  the door for the other's.
- *Why does it fail on symmetric NAT?* That NAT picks a new external port for each
  destination, so the port the relay observed isn't the port the peer will see.
- *Why one socket?* NAT mappings are per socket. A separate STUN socket would report an
  address that's useless for the peer.

# ADR-0007: Host the default relay on Oracle Cloud Always Free (Mumbai/Hyderabad)

- **Status:** Accepted, 2026-10-16
- **Evidence:** `docs/research/tooling-hosting.md`

## Context
The relay carries the 10–30% of transfers that can't punch through, so **outbound bandwidth
is the real cost**. The budget is ₹0–500 a month, and the relay needs UDP (QUIC).

## Options
| Option | Egress | Verdict |
|---|---|---|
| **Oracle Always Free A1** (about 2 OCPU / 12 GB ARM) | 10 TB/month free | Fits the budget with a lot of headroom |
| GCP e2-micro | 1 GB/month | Useless |
| Fly.io | No free tier; $0.12/GB in India | Grows with use |
| Hetzner CX23 | 20 TB (EU) | About ₹600 a month (unverified), and far from Indian users |
| AWS | Credits for 6 months | Temporary |

## Decision
Oracle A1 (arm64) in Mumbai or Hyderabad, run as a systemd service. GoReleaser already builds
`linux/arm64`. Hetzner is the paid fallback.

## Risks and mitigations
- **Idle reclaim:** Oracle reclaims an instance only when **all three** of p95 CPU, network
  and memory stay below 20% for 7 days.
  - **Mitigation:** keep one metric above 20%. Memory is the easy one: the relay's buffer
    pool plus `GOMEMLIMIT` sized to hold more than 20% of RAM. Uptime pings don't raise p95
    CPU.
  - Whether pay-as-you-go exempts the instance from reclaim isn't documented, so treat it as
    **unverified**.
- **Signing up from India:** use a Visa or Mastercard **credit** card. RuPay, prepaid and
  virtual cards fail, and signing up over a VPN gets rejected.
- **Single relay = single point of failure for internet transfers** until Phase 5. Any user
  can run `p2p relay`, and the DHT then spreads rendezvous across all relays.

## What an interviewer might ask
- *Doesn't a hosted relay make it centralised?* It's a default, not a requirement:
  - LAN transfers need no relay at all;
  - anyone can run one;
  - from Phase 5, rendezvous goes through the DHT;
  - the relay only ever sees end-to-end ciphertext.

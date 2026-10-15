# ADR-0008: Code phrase: nameplate + 3 PGP words, single use, never in argv

- **Status:** Accepted, 2026-10-16
- **Evidence:** `docs/research/crypto.md`, `docs/research/dht.md`

## Context
The code has to be short enough to read aloud and strong enough against online guessing. It
also has to work as a lookup key for rendezvous without leaking the secret part.

## Decision
- **Format:** `<nameplate>-<w1>-<w2>-<w3>`, for example `4-tiger-orbit-quantum`.
  - The **nameplate** is a short public number, used only to find the other peer.
  - The **3 words** come from the PGP wordlist, alternating its even and odd lists so that a
    dropped or swapped word is detected. That gives 3 × 8 = **24 bits**, a 1-in-16.7M chance
    per online guess.
- **Only the words go into SPAKE2.** Rendezvous keys (the relay mailbox, the DHT key
  `H("p2p/1" ‖ nameplate)`) use the nameplate alone. **Nothing derived from the words is ever
  stored or published.** If it were, an attacker could enumerate 2^24 candidates offline.
- **Single use.** A failed key confirmation burns the nameplate on both sides. The relay
  refuses a third party on a nameplate ("crowded", as in wormhole).
- **Never in argv.** `p2p receive` prompts for the code, or reads `P2P_CODE`. Command-line
  arguments are visible to other local users through the process list (croc CVE-2023-43621).
- **Nameplate allocation (revised after the adversarial review):**
  - **Relay mode (Phases 3–4).** The relay picks a *random* free number from
    `[1, max(100, 10 × active sessions)]`. Codes stay short, but they can't be predicted the
    way "the smallest free number" can.
  - **Claims are rate-limited per IP** (for example 5 a minute). An attacker can't race to
    claim every live nameplate the moment it appears.
  - **Unpaired timeout: 10 minutes.** The sender has to wait while the code is read out, and
    the claim rate limit, not the timeout, is what stops squatting.
  - **DHT mode (Phase 5):**
    - nameplates are random 6-digit numbers. Collisions reach 50% only at about 1,200
      simultaneous sends, and even then they cost one retry.
    - **A DHT key can hold several records.** Each record is `{sender ephemeral public key,
      relay address, seq, expiry ≤ 10 min}`, signed by that key.
    - **That public key is bound into SPAKE2 identity A.** A receiver that follows a forged
      or replayed record runs the PAKE against the wrong identity, and confirmation fails.
- **Accepted residual risk:** an attacker can still burn individual codes (a DoS). They get
  one guess each, and the sender simply retries with a new code.

## Consequences
- An attacker who squats a nameplate or impersonates a sender still gets only one online guess
  (the same guarantee as wormhole).
- Typing the code at a prompt adds a small amount of friction compared with croc. We accept
  that.
- DHT records reveal the sender's relay address, not the sender's own IP. The IP is
  exchanged only inside the encrypted channel (ADR-0005).

## What an interviewer might ask
- *Is 24 bits enough?* Yes, because guessing is online-only and every failure aborts the
  transfer. An attacker gets one attempt, and the legitimate pair notices it.
- *Why not put the code in the DHT key?* It would make offline enumeration possible, which
  turns a 1-in-16.7M online guess into a certain offline match.

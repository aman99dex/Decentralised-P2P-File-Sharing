# Research: Go tooling, releases, CI, relay hosting (checked 2026-10-16)

## Go & libraries
- **Go version:** stable is go1.27.2 (2026-10-08); 1.26.x is still supported. https://go.dev/doc/devel/release
  - In go.mod, set `go 1.26` (N-1, so more users can build it) and `toolchain go1.27.2`. https://go.dev/doc/toolchain
- **Layout:** `cmd/p2p/main.go` plus `internal/`, the official recommendation. https://go.dev/doc/modules/layout
- **CLI:** stdlib `flag` with a subcommand switch. cobra v1.10.2 or urfave/cli v3.14 only if we later
  need shell completion or nested commands.
- **Logging:** `log/slog`. JSONHandler for the relay, TextHandler for the CLI.
- **Testing:**
  - `testing/synctest` has been GA since Go 1.25; use it for timeouts and keepalives without real sleeps. https://go.dev/doc/go1.25
  - `go test -fuzz` for the parsers; `-race` in CI.
- **Lint:** golangci-lint v2.14.0. The v2 config uses `version: "2"`, `linters.default: standard`, and a
  separate `formatters:` block. https://golangci-lint.run/docs/configuration/file/

## Releases
- **GoReleaser:** OSS v2.18.3 is enough; Pro ($15/mo) only adds installers and native notarize.
- **Targets:** linux/darwin/windows × amd64/arm64, `CGO_ENABLED=0`.
- **Homebrew:** `homebrew_casks` (`brews` is deprecated), plus a post-install `xattr` hook while the binary
  isn't notarized. https://goreleaser.com/customization/homebrew_casks/
- **Scoop:** the `scoops:` key, with a PAT that can write to the bucket repo.
- **Signing:** cosign keyless `sign-blob --bundle` on checksums.txt, plus `actions/attest@v4`
  provenance (users verify with `gh attestation verify`). Both are free.
  - https://goreleaser.com/customization/sign/
  - https://docs.github.com/en/actions/how-tos/secure-your-work/use-artifact-attestations/use-artifact-attestations
- **macOS notarization:** $99/yr Apple Developer. Skip it at first; curl, go install and the brew cask
  with `xattr` cover it.

## CI
- checkout → setup-go (`go-version-file: go.mod`) → vet → `test -race` → golangci-lint →
  govulncheck-action.
- Pin actions by SHA, kept current with Renovate/Dependabot.
- A tag push runs GoReleaser + cosign + attest.

## Relay hosting (bandwidth is the real cost)
| Option | Egress | Verdict |
|---|---|---|
| **Oracle Always Free** A1 (≈2 OCPU / 12 GB) | **10 TB/mo** | **Pick.** Mumbai/Hyderabad region |
| GCP e2-micro | 1 GB/mo | Useless for a relay |
| Fly.io | No free tier for new orgs; $0.12/GB India | Too costly |
| Hetzner CX23 | 20 TB (EU) | Paid fallback, ≈₹600/mo (unverified), EU latency |
| AWS | $200 credits, ends after 6 months | Short-term only |

Sources:
- Oracle: https://docs.oracle.com/en-us/iaas/Content/FreeTier/freetier_topic-Always_Free_Resources.htm
- GCP: https://docs.cloud.google.com/free/docs/free-cloud-features
- Fly.io: https://docs.fly.io/about/pricing
- Hetzner: https://www.hetzner.com/cloud/regular-performance/
- AWS: https://docs.aws.amazon.com/awsaccountbilling/latest/aboutv2/free-tier.html

Caveats:
- **Oracle idle reclaim:** Oracle reclaims instances idle for 7 days (p95 CPU/net/mem < 20%). Whether
  pay-as-you-go is exempt is unverified.
- **Signing up from India:** use a Visa or Mastercard credit card. RuPay, prepaid and virtual cards
  fail, and signing up over a VPN gets rejected.
  https://dev.to/ashish-codejourney/getting-a-free-oracle-cloud-vps-from-india-the-full-guide-and-the-rupay-wall-nobody-warns-you-4p5o

## Relay ops
- **Deployment:** a systemd unit, not Docker: `Restart=on-failure`, `DynamicUser=yes`, logs to journald.
- **Admin port (localhost/private):** `/metrics` via prometheus/client_golang v1.25.0 and `/healthz`.
- **Graceful shutdown:** `signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)`, then stop
  accepting, `Shutdown` with a deadline, and drain the relays.

## Unverified
- The cobra and urfave release years.
- The latest golangci-lint-action version.
- Hetzner's post-June-2026 prices.
- Whether Oracle pay-as-you-go is exempt from idle reclaim.
- The SLSA level of GitHub attestations.

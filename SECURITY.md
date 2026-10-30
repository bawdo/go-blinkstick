# Security policy

## Supported versions

Until `v1.0.0`, only the latest minor release gets security fixes. Upgrade to it before
reporting.

## Verifying a release

Every release from the next one on carries the module zip, the file `go get` downloads, and a
signed SLSA build provenance attestation for it (`go-blinkstick-vX.Y.Z.intoto.jsonl`). The
attestation is made by the release workflow when it runs for the tag. To check the zip in your
module cache is the one it attested, with the [GitHub CLI](https://cli.github.com/):

```sh
v=vX.Y.Z
go mod download github.com/bawdo/go-blinkstick@$v
gh attestation verify "$(go env GOMODCACHE)/cache/download/github.com/bawdo/go-blinkstick/@v/$v.zip" \
  --repo bawdo/go-blinkstick \
  --signer-workflow bawdo/go-blinkstick/.github/workflows/release.yml \
  --source-ref refs/tags/$v --deny-self-hosted-runners
```

Add `--bundle go-blinkstick-$v.intoto.jsonl`, downloaded from the release, to check it without
GitHub's attestation API. `go` itself already checks the zip against
[sum.golang.org](https://sum.golang.org), so the attestation adds which workflow and tag produced
it, not tamper evidence.

## Reporting a vulnerability

Please do not open a public issue for a security problem.

Report it privately through GitHub instead:
[open a private vulnerability report](https://github.com/bawdo/go-blinkstick/security/advisories/new).

Include what you found, how to reproduce it, and which version, OS and device you used.

You should hear back within 7 days. Once a fix is ready, it goes out in a release and the
advisory is published with credit to you, unless you would rather stay anonymous.

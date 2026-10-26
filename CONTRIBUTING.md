# Contributing

Thanks for helping out. Bug reports, fixes and new device support are all welcome.

## Before you start

- For anything bigger than a small fix, open an issue first so we can agree on the approach.
- Security problems go through [SECURITY.md](SECURITY.md), not a public issue.
- Everyone taking part follows the [code of conduct](CODE_OF_CONDUCT.md).

## Making a change

1. Fork the repo and branch from `main`.
2. Make your change, with tests.
3. Run `make test`. It runs `go vet` and the unit tests with the race detector, and needs no
   hardware.
4. CI also runs `staticcheck` and `govulncheck`, on macOS and Linux. To run them yourself:

   ```sh
   go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 ./...
   go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
   ```

5. For anything users will notice, add a line under an `Unreleased` heading at the top of
   [CHANGELOG.md](CHANGELOG.md), creating the heading if it is not there.
6. Open a pull request.

## Hardware tests

`make test-hardware` and `make test-reconnect` need a BlinkStick Nano and a Square attached.
CI cannot run them. If your change touches device I/O, run them if you can and say so in the
pull request.

**Tests must never write EEPROM on a real stick.** EEPROM wears out. Hardware tests may read
names and info blocks and set LEDs (RAM, no wear), but must not call `SetInfoBlock`, `SetName`,
`SetMode` or anything else that writes EEPROM. Use the in-memory fake for those.

## Commits

- `main` only accepts signed commits. Set up
  [commit signing](https://docs.github.com/en/authentication/managing-commit-signature-verification)
  before you push.
- Write commit subjects in the past tense, around 50 characters, for example
  `Added SetMode with allowed modes per model`.
- Keep each commit to one logical change. Pull requests are merged with a merge commit, so
  your commits land as you wrote them.

## Licence

By contributing, you agree your work is released under the [MIT licence](LICENSE).

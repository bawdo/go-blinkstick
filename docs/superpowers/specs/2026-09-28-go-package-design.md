# go-blinkstick package design

Date: 2026-09-28
Branch: `feature/go-package`
Module: `github.com/bawdo/go-blinkstick`, package `blinkstick`

## Summary

The repository is converted from a `go/` subdirectory holding a driver plus a `blink` CLI into a
single Go library module at the repository root. The first release, `v0.1.0`, supports the
BlinkStick Nano and Square on macOS and covers the MVP feature set listed below. The design
keeps adding other BlinkStick models cheap: a model is a row of data, not a type.

## Decisions

| Topic | Decision |
|---|---|
| Repository | Renamed on GitHub to `bawdo/go-blinkstick` (done). Local `origin` updated to match. |
| Module path | `github.com/bawdo/go-blinkstick`, package `blinkstick` at the repository root. |
| CLI | Dropped. `cmd/blink` is deleted. Usage lives in godoc examples. |
| Structure | One concrete `Device` type plus a `Model` table (approach A). |
| HID library | `github.com/sstallion/go-hid` (cgo, bundles hidapi). |
| Go directive | `go 1.26` (the two supported releases at the time of writing are 1.26 and 1.27). |
| Versioning | Semver tags with a `v` prefix. First release `v0.1.0`. Stay on `v0`/`v1` to avoid a `/v2` import path. |
| Makefile | `help`, `test`, `build`, `test-hardware`. No `run` or `install`. |
| Licence | MIT, GitHub's template text, copyright Keith Bawden. |
| Docs | `README.md` (600 words or less), `CAPABILITIES.md`, `CHANGELOG.md`, `doc.go`. |
| CI | GitHub Actions on `macos-latest`. |

## Hardware facts this design relies on

Taken from `ioreg` captures of the user's devices (`scratch_pad/ioreg_*.txt`) and from the
upstream Python, Ruby, .NET and firmware sources.

| | Nano | Square |
|---|---|---|
| USB ID | `20a0:41e5` | `20a0:41e5` |
| Serial | `BS072777-3.0` | `BS073788-3.1` |
| `bcdDevice` | `0x0202` | `0x0201` |
| Product string | `BlinkStick Nano` | `BlinkStick` |
| Manufacturer string | `Agile Innovative Ltd` | `Agile Innovative Ltd` |
| LEDs | 2 (0 top, 1 bottom) | 8 |
| Feature reports | 1 to 9 | 1 to 9, plus 10 (2 bytes, undocumented) |
| Report 0x81 (LED count) | absent | absent |

Feature report sizes from the descriptor (payload bytes, excluding the report ID byte): 1 = 3,
2 = 32, 3 = 32, 4 = 1, 5 = 5, 6 = 25, 7 = 49, 8 = 97, 9 = 193, 10 = 2 (Square only).

Report 6 layout: `[0x06, channel, 8 x (G, R, B)]`. Colour order on the wire is GRB. Reports 1
and 5 use RGB. The v3 firmware source is not published, so behaviour not observed on hardware
is treated as unverified.

Upstream Python maps `bcdDevice 0x0201` to the Strip. The user's Square reports `0x0201`, so
Python's rule is wrong for at least this Square, and Square and Strip may share the value.

## Feature scope

This table seeds `CAPABILITIES.md`. "Device" means implemented by the firmware, "host" means
implemented in this package.

| Feature | Kind | Nano | Square | v0.1.0 |
|---|---|---|---|---|
| List attached sticks | device | yes | yes | yes |
| Open first, open by serial | device | yes | yes | yes |
| Several sticks open at once | device | yes | yes | yes |
| Serial and firmware version | device | yes | yes | yes |
| Manufacturer and product strings | device | yes | yes | yes |
| Model detection | host | yes | yes | yes |
| Set all LEDs | device | yes | yes | yes |
| Set one LED by index | device | yes | yes | yes |
| Set a frame of per-LED colours | device | yes | yes | yes |
| Read LED colours back | device | yes | yes | yes |
| Info blocks 1 and 2 (read, write) | device | yes | yes | yes |
| Parse colour from hex and `r,g,b` | host | yes | yes | yes |
| Brightness limit | host | yes | yes | yes |
| Blink, pulse, morph effects | host | yes | yes | yes |
| Single LED via report 5 (mode 2) | device | unverified | unverified | no |
| Device mode get and set (report 4) | device | unverified | unverified | no |
| Report 10 | device | n/a | unknown | no |
| CSS colour names, random colour | host | yes | yes | no |
| Inverse colours | host | yes | yes | no |
| Reconnect after unplug | host | yes | yes | no |
| LED count register (0x81) | device | n/a | n/a | n/a |
| Reports 7 to 9 (16, 32, 64 LEDs) | device | n/a | n/a | n/a |
| macOS | platform | yes | yes | yes |
| Linux | platform | yes | yes | no |

## Repository layout

```
go-blinkstick/
  go.mod  go.sum
  doc.go            package docs and quick start
  colour.go         RGB, ParseRGB, brightness scaling
  model.go          Model table and detection
  device.go         Device, List, Open, OpenSerial, LED methods
  infoblock.go      info block read and write
  effects.go        Blink, Pulse, Morph
  report.go         unexported report encoding and decoding
  hid.go            unexported go-hid adaptor, init/exit reference counting
  *_test.go         unit tests against a fake device
  example_test.go   runnable godoc examples (no Output lines, never touch hardware)
  hardware_test.go  //go:build hardware
  Makefile  LICENSE  README.md  CAPABILITIES.md  CHANGELOG.md
  .github/workflows/ci.yml
  .gitignore
```

## Public API

```go
type RGB struct{ R, G, B uint8 }
var Off, White RGB
func ParseRGB(s string) (RGB, error) // "#f80", "#ff8800", "255,136,0"

type Model struct {
    Name string // "Nano", "Square", "unknown"
    LEDs int
}
var Nano, Square Model

type Info struct {
    Serial       string // "BS072777-3.0"
    Version      string // "3.0", taken from the serial
    Manufacturer string
    Product      string
    Model        Model
}

var (
    ErrNotFound    = errors.New(...)
    ErrUnsupported = errors.New(...)
    ErrOutOfRange  = errors.New(...)
    ErrClosed      = errors.New(...)
)

func List() ([]Info, error)
func Open() (*Device, error)
func OpenSerial(serial string) (*Device, error)

func (d *Device) Info() Info
func (d *Device) SetAll(c RGB) error
func (d *Device) SetLED(i int, c RGB) error
func (d *Device) SetFrame(cs []RGB) error // len(cs) must equal Model.LEDs
func (d *Device) Frame() ([]RGB, error)   // len equals Model.LEDs
func (d *Device) LED(i int) (RGB, error)
func (d *Device) Off() error
func (d *Device) SetBrightnessLimit(max uint8) // 255 means no limit
func (d *Device) InfoBlock(n int) ([]byte, error)       // n is 1 or 2, trailing NULs trimmed
func (d *Device) SetInfoBlock(n int, data []byte) error // at most 32 bytes, zero padded
func (d *Device) Blink(ctx context.Context, c RGB, period time.Duration, repeats int) error
func (d *Device) Pulse(ctx context.Context, c RGB, duration time.Duration, repeats int) error
func (d *Device) Morph(ctx context.Context, to RGB, duration time.Duration) error
func (d *Device) Close() error
```

Identifiers avoid the words colour and color. Comments and docs use Australian spelling.

## Behaviour

### Discovery and opening

- `List` calls `hid.Enumerate(0x20A0, 0x41E5, ...)`, filtered to BlinkSticks, and returns an
  `Info` per stick. Sticks with an unrecognised model are listed with `Model.Name == "unknown"`.
- `Open` opens the first stick found. `OpenSerial` uses `hid.Open(vid, pid, serial)`.
- Opening a stick whose model is unknown returns `ErrUnsupported`. No stick returns
  `ErrNotFound`.
- hidapi `Init` and `Exit` are reference counted behind a mutex: init on the first open, exit
  when the last device closes. The current code calls `hid.Exit()` on every close, which would
  break other open devices.
- hidapi opens devices exclusively on macOS, so only one process can hold a stick. Documented.

### Model detection

A table in `model.go` maps (serial major version, `bcdDevice`) to a `Model`:

- major 3, `0x0202`: Nano
- major 3, `0x0201`: Square. The godoc notes that the Strip may share this value.
- anything else: unknown

No other file branches on model. Model differences are expressed as data (LED count).

### LEDs

- `SetAll` and `SetFrame` send report 6 on channel 0. Unused LED slots are zero. The Nano
  receives a full 8-LED frame and ignores slots 2 to 7.
- `Frame` reads report 6 and returns the first `Model.LEDs` entries.
- `SetLED` reads the frame, changes one entry and writes the frame back. Report 5 is not used
  because it requires firmware mode 2, which is unverified on v3.
- `LED` reads the frame and returns one entry.
- An index outside `0..Model.LEDs-1`, or a frame of the wrong length, returns an error
  wrapping `ErrOutOfRange`.
- Each transfer is retried up to 3 attempts in total, because the firmware can be busy
  straight after a write and macOS reports intermittent `IOHIDDeviceSetReport` failures
  (go-hid issue #15). The final error wraps the underlying one.
- Report buffers are sized exactly from the descriptor (payload plus one byte for the report
  ID). The first byte is always the report ID.

### Brightness limit

Every channel is scaled as `v * max / 255` before sending, including effects. `Frame` and `LED`
return the raw values stored on the device, and the godoc says so. Default is 255 (no limit).

### Info blocks

Report 2 is info block 1, report 3 is info block 2, 32 bytes each, stored in EEPROM.
`SetInfoBlock` zero pads to 32 bytes and rejects longer data with `ErrOutOfRange`. `InfoBlock`
trims trailing NUL bytes. `n` outside 1 or 2 returns `ErrOutOfRange`.

### Effects

All effects apply to every LED, block until finished, and return `ctx.Err()` if cancelled.
Step interval is 20 ms.

- `Blink`: on for `period / 2`, off for `period / 2`, `repeats` times. Ends off.
- `Pulse`: off, morph up to `c` over `duration / 2`, morph down to off over `duration / 2`,
  `repeats` times. Ends off.
- `Morph`: reads the current frame, then interpolates each LED linearly to `to` over
  `duration`. Ends at `to`.

### Concurrency

`Device` methods are safe to call from several goroutines. A mutex guards each transfer and
each read-modify-write in `SetLED`. Two effects running on one device at the same time will
interleave their frames. The package does not prevent this, and the godoc says so.

## Risks, checked first in implementation

1. **Enumerate panic (go-hid issue #18).** It panicked on macOS when a Focusrite interface was
   attached. The expectation is that the panic comes from Go-side string conversion of
   unrelated devices, and that a vendor and product filtered enumerate avoids it. A spike runs
   `List` with the Focusrite, Nano and Square all attached. If it still panics, stop and
   redesign `List` before building on it. `OpenSerial` does not depend on enumeration.
2. **Report 6 read back.** Setting via report 6 is verified on hardware. Reading it back is
   not. The spike writes a known frame to each stick and reads it back. If it does not match,
   stop and redesign `Frame`, `LED`, `SetLED` and `Morph`.
3. **Report 10 on the Square.** Probed read only (a single get feature report), result recorded
   in `CAPABILITIES.md`. Never written.

## Testing

- **Unit tests** use a fake device that behaves like the firmware: it stores the last report 6
  frame and info blocks, returns them on get, and can be told to fail the next N transfers.
  Table-driven, run with `-race`. They never touch hardware, so they never write EEPROM.
- **Example tests** in `example_test.go` compile under `go test` but have no `// Output:`
  line, so they never run against hardware.
- **Hardware tests** in `hardware_test.go` behind `//go:build hardware`, run with
  `make test-hardware`. They need the Nano and Square attached and a person watching the LEDs.
  They cover every v0.1.0 feature on both sticks, and a multi-device check: both sticks open
  at once, driven from separate goroutines, closing one leaves the other working.
- **EEPROM rule:** hardware tests read info blocks but never call `SetInfoBlock` or anything
  else that writes EEPROM. A comment at the top of `hardware_test.go` states this. LED colours
  live in RAM, so LED tests cause no wear.
- Hardware tests are run 3 times in a row before release.

## Makefile

```
make help           list targets (self-documenting from ## comments)
make test           go vet ./... and go test -race ./...
make build          go build ./...
make test-hardware  go test -tags hardware -count=1 -v ./...
```

## CI

`.github/workflows/ci.yml` on push and pull request, `macos-latest`, Go 1.26 and 1.27:
`go vet`, `go test -race ./...`, `staticcheck`, `govulncheck`. No hardware needed.

## Documentation

- **README.md:** 600 words or less, Australian English, no em dashes, no emoji. Covers what it
  is, a plain statement that it is unofficial and not affiliated with Agile Innovative, the
  cgo and Xcode command line tools requirement, install, a short quick start, platform support,
  a link to `CAPABILITIES.md`, versioning and licence. pkg.go.dev and Go Report Card badges.
- **CAPABILITIES.md:** the feature table above, with a "Supported since" column updated as
  features land, split into device, host and platform sections, linking back to the README.
- **CHANGELOG.md:** Keep a Changelog format, starting with `v0.1.0`.
- **doc.go:** package overview and quick start, the landing text on pkg.go.dev.

## Migration from the current tree

- Move `go/blinkstick/*.go` and `go/go.mod`, `go/go.sum` to the repository root, rewriting
  them to the design above. Tests are rewritten against the new API.
- Delete `go/cmd/blink`, `go/README.md` and `go/bin/`.
- Change the module path to `github.com/bawdo/go-blinkstick` and the directive to `go 1.26`.
- `.gitignore`: fix `scratchpad/` to `scratch_pad/` (the existing entry never matched), and
  drop `go/bin/`.
- The current untracked root `README.md` (the multi-goal roadmap) is overwritten by the new
  package README, which is tracked in git from the first implementation commit. The user holds
  a backup of the old content.
- `git remote set-url origin git@github.com:bawdo/go-blinkstick.git`.

## Release

1. All unit tests pass, CI green on the branch.
2. Hardware tests pass 3 times in a row on both sticks, with the Focusrite attached.
3. `CAPABILITIES.md` and `CHANGELOG.md` updated.
4. Merge to `main`, tag `v0.1.0` (signed tag). Pushing and tagging are handed to the user.
5. Prime pkg.go.dev with
   `GOPROXY=proxy.golang.org go list -m github.com/bawdo/go-blinkstick@v0.1.0`.

## Out of scope for v0.1.0

Linux, other BlinkStick models, device modes, report 5, report 10 beyond a read-only probe,
CSS colour names, random colour, inverse, reconnect, a CLI, `SECURITY.md`, `CONTRIBUTING.md`.

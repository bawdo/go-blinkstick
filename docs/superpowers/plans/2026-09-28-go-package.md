# go-blinkstick Package Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Convert the repository into the root-level Go library `github.com/bawdo/go-blinkstick` and ship `v0.1.0`, supporting the BlinkStick Nano and Square on macOS with the MVP feature set.

**Architecture:** One concrete `Device` type plus a `Model` table (models differ only in data). Device access goes through a small unexported `transport` interface, and discovery through an unexported `backend` interface, so every unit test runs against fakes. The real implementations wrap `github.com/sstallion/go-hid`, with hidapi `Init`/`Exit` reference counted so several sticks can be open at once.

**Tech Stack:** Go 1.26+, cgo, `github.com/sstallion/go-hid` v0.15.0 (bundles hidapi), GNU make, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-09-28-go-package-design.md`

## Global Constraints

- Module path: `github.com/bawdo/go-blinkstick`. Package name: `blinkstick`, at the repository root.
- `go.mod` directive: `go 1.26`.
- No CLI. `cmd/blink` is deleted and nothing replaces it.
- USB IDs: vendor `0x20A0`, product `0x41E5`.
- Report 6 frame: `[0x06, channel 0, 8 x (G, R, B)]`, 26 bytes including the report ID. GRB order.
- Info block reports: 2 (block 1) and 3 (block 2), 33 bytes including the report ID.
- Model table: serial major `3` with `bcdDevice 0x0202` is Nano (2 LEDs); serial major `3` with `bcdDevice 0x0201` is Square (8 LEDs); anything else is unknown. No file other than `model.go` branches on model (test code excepted).
- Transfers retry up to 3 attempts in total.
- Effects step every 20 ms.
- Identifiers avoid the words colour and color. Comments and docs use Australian English.
- No em dashes or en dashes anywhere (code comments, docs, commit messages). No emoji.
- Run `gofmt -l .` before every commit. It must print nothing (the code blocks below may need realigning by `gofmt -w`).
- **EEPROM rule:** hardware tests never call `SetInfoBlock` or anything else that writes EEPROM.
- **Commits:** ask the user for permission before every commit. Sign every commit with gpg (`git commit -S`). The sandbox blocks the gpg agent, so run commits with the sandbox disabled. If signing still fails, stop, report it loudly, and ask the user to "refresh the key". Subjects are past tense, 60 characters or fewer. No agent attribution of any kind: no `Co-Authored-By`, no "Generated with", no session URLs.
- Writing `.git/config` (for example `git remote set-url`) also needs the sandbox disabled.
- Never push, tag or open a pull request. Hand those commands to the user.

## File Structure

| File | Responsibility |
|---|---|
| `go.mod`, `go.sum` | Module definition (moved from `go/`) |
| `doc.go` | Package documentation, the pkg.go.dev landing text |
| `colour.go` | `RGB`, `Off`, `White`, `ParseRGB`, brightness scaling |
| `model.go` | `Model`, `Nano`, `Square`, model table, `Info`, `deviceInfo`, `newInfo` |
| `report.go` | USB IDs, report IDs and sizes, frame encode and decode |
| `device.go` | Errors, `transport`, `Device`, LED methods, `backend`, `List`, `Open`, `OpenSerial` |
| `hid.go` | go-hid adaptor: `hidBackend`, `hidTransport`, reference-counted `Init`/`Exit` |
| `infoblock.go` | `InfoBlock`, `SetInfoBlock` |
| `effects.go` | `Blink`, `Pulse`, `Morph`, cancellable sleep |
| `fake_test.go` | `fakeTransport`, `fakeBackend`, test helpers |
| `*_test.go` | Unit tests per file |
| `example_test.go` | Runnable godoc examples |
| `hardware_test.go` | `//go:build hardware` tests against real sticks |
| `Makefile` | `help`, `test`, `build`, `test-hardware` |
| `LICENSE` | MIT, GitHub's text |
| `README.md`, `CAPABILITIES.md`, `CHANGELOG.md` | User docs |
| `.github/workflows/ci.yml` | CI on macOS |
| `.gitignore` | Ignores `scratch_pad/` |

---

### Task 1: Restructure into a root module

**Files:**
- Move: `go/go.mod` to `go.mod`, `go/go.sum` to `go.sum`
- Delete: `go/blinkstick/`, `go/cmd/`, `go/README.md`, `go/bin/`
- Create: `doc.go`, `Makefile`, `LICENSE`, `README.md` (interim, replaces the untracked roadmap README)
- Modify: `.gitignore`

**Interfaces:**
- Consumes: nothing
- Produces: an empty `package blinkstick` at the root, `make test` and `make build` working

- [ ] **Step 1: Move the module files and delete the old code**

```bash
cd /Users/bawdo/working/projects/blinkstick-playground/blinkstick-go
git mv go/go.mod go.mod
git mv go/go.sum go.sum
git rm -r -q go/blinkstick go/cmd go/README.md
rm -rf go
```

The old code (`stick.go`, `report.go`, `hid.go`) is superseded by the tasks below. Its knowledge (GRB order, retry count, the `Enumerate` panic) is carried into the new files.

- [ ] **Step 2: Rewrite `go.mod`**

Replace the whole file with:

```
module github.com/bawdo/go-blinkstick

go 1.26

require github.com/sstallion/go-hid v0.15.0

require golang.org/x/sys v0.8.0 // indirect
```

Do not run `go mod tidy` yet: nothing imports go-hid until Task 8, and tidy would drop it.

- [ ] **Step 3: Create a stub `doc.go`**

```go
// Package blinkstick drives BlinkStick USB LED devices.
package blinkstick
```

Task 11 replaces this with the full package documentation.

- [ ] **Step 4: Replace `.gitignore`**

The old entry `scratchpad/` never matched the real `scratch_pad/` directory, and `go/bin/` no longer exists. Replace the whole file with:

```
scratch_pad/
```

- [ ] **Step 5: Create `Makefile`**

Recipe lines must start with a real tab.

```make
.DEFAULT_GOAL := help

.PHONY: help test build test-hardware

help: ## List make targets
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run go vet and unit tests with the race detector
	go vet ./...
	go test -race ./...

build: ## Compile the package
	go build ./...

test-hardware: ## Run hardware tests (needs a Nano and a Square attached)
	go test -tags hardware -race -count=1 -v -run '^TestHardware' ./...
```

- [ ] **Step 6: Download GitHub's MIT licence text**

```bash
gh api /licenses/mit --jq .body \
  | sed -e 's/\[year\]/2026/' -e 's/\[fullname\]/Keith Bawden/' > LICENSE
head -3 LICENSE
```

Expected: `MIT License`, a blank line, then `Copyright (c) 2026 Keith Bawden`. If `gh` is not authenticated or has no network, stop and ask the user.

- [ ] **Step 7: Write the interim `README.md`**

This overwrites the untracked roadmap README (the user has a backup). Task 13 replaces it with the full README.

```markdown
# go-blinkstick

A Go package for BlinkStick Nano and Square USB LED devices. Unofficial, and not affiliated
with Agile Innovative.

Work in progress. Not released yet.

## Licence

MIT. See [LICENSE](LICENSE).
```

- [ ] **Step 8: Update the git remote**

The GitHub repository was renamed. Needs the sandbox disabled (writes `.git/config`).

```bash
git remote set-url origin git@github.com:bawdo/go-blinkstick.git
git remote -v
```

Expected: both lines show `git@github.com:bawdo/go-blinkstick.git`.

- [ ] **Step 9: Verify**

```bash
make help
make build
make test
git status --short
```

Expected: `make help` lists the four targets. `make build` and `make test` succeed (`test` prints `no test files`). `git status` shows the renames, deletions and the new files. `scratch_pad/` does not appear.

- [ ] **Step 10: Commit (after the user approves)**

```bash
git add -A go go.mod go.sum doc.go Makefile LICENSE README.md .gitignore
git commit -S -m "Restructured repo into a root-level Go module"
git log -1 --format='%h %G? %s'
```

Expected: `G` in the second column.

---

### Task 2: Hardware spike (not committed)

Checks the two risks in the spec before anything is built on them: the filtered `Enumerate` does not panic with the Focusrite attached, and report 6 reads back what was written. It also probes report 10 on the Square, read only.

**Files:**
- Create then delete: `spike_test.go` (never committed)

**Interfaces:**
- Consumes: go-hid directly
- Produces: findings recorded in the task report (no code)

- [ ] **Step 1: Ask the user to attach the hardware**

The Nano, the Square and the Focusrite interface all need to be plugged in. Wait for confirmation.

- [ ] **Step 2: Write `spike_test.go`**

```go
//go:build spike

package blinkstick

import (
	"bytes"
	"testing"

	"github.com/sstallion/go-hid"
)

func retry(f func() (int, error)) (int, error) {
	var n int
	var err error
	for range 3 {
		if n, err = f(); err == nil {
			return n, nil
		}
	}
	return n, err
}

func TestSpike(t *testing.T) {
	if err := hid.Init(); err != nil {
		t.Fatal(err)
	}
	defer hid.Exit()

	var infos []*hid.DeviceInfo
	err := hid.Enumerate(0x20A0, 0x41E5, func(i *hid.DeviceInfo) error {
		infos = append(infos, i)
		return nil
	})
	if err != nil {
		t.Fatalf("Enumerate: %v", err)
	}
	for _, i := range infos {
		t.Logf("found %s release=%#04x mfr=%q product=%q usage=%#x/%#x path=%s",
			i.SerialNbr, i.ReleaseNbr, i.MfrStr, i.ProductStr, i.UsagePage, i.Usage, i.Path)
	}
	if len(infos) != 2 {
		t.Fatalf("found %d BlinkSticks, want 2", len(infos))
	}

	for _, i := range infos {
		leds := 8
		if i.ReleaseNbr == 0x0202 {
			leds = 2
		}
		dev, err := hid.Open(0x20A0, 0x41E5, i.SerialNbr)
		if err != nil {
			t.Fatalf("Open(%s): %v", i.SerialNbr, err)
		}

		want := make([]byte, 26)
		want[0] = 6
		for j := 2; j < len(want); j++ {
			want[j] = byte(j * 9)
		}
		if _, err := retry(func() (int, error) { return dev.SendFeatureReport(want) }); err != nil {
			t.Errorf("%s send: %v", i.SerialNbr, err)
		}

		got := make([]byte, 26)
		got[0] = 6
		n, err := retry(func() (int, error) { return dev.GetFeatureReport(got) })
		t.Logf("%s get report 6: n=%d err=%v bytes=% x", i.SerialNbr, n, err, got)
		if end := 2 + leds*3; !bytes.Equal(got[:end], want[:end]) {
			t.Errorf("%s readback mismatch in first %d LEDs\n got % x\nwant % x",
				i.SerialNbr, leds, got[:end], want[:end])
		}

		if i.ReleaseNbr == 0x0201 {
			r10 := make([]byte, 3)
			r10[0] = 10
			n, err := dev.GetFeatureReport(r10)
			t.Logf("%s get report 10 (read only): n=%d err=%v bytes=% x", i.SerialNbr, n, err, r10)
		}

		off := make([]byte, 26)
		off[0] = 6
		retry(func() (int, error) { return dev.SendFeatureReport(off) })
		dev.Close()
	}
}
```

- [ ] **Step 3: Run it**

```bash
go test -tags spike -count=1 -v -run TestSpike .
```

Expected: no panic, two `found` lines (one `0x0202` Nano, one `0x0201` Square), no readback mismatch, and the report 10 bytes logged. Both sticks briefly show a mixed pattern, then go dark.

- [ ] **Step 4: Record the findings and gate**

Report to the user: the full `-v` output, the `n` value returned by `GetFeatureReport` (26 or 25), whether the Nano read back all 8 slots or only 2, and the report 10 bytes.

**STOP conditions.** If any of these happen, do not continue. Report to the user and wait for a redesign:
- `Enumerate` panics, or does not return exactly the two sticks. Duplicate entries for the same serial also count.
- Report 6 readback does not match for the first `leds` LEDs on either stick.

- [ ] **Step 5: Delete the spike**

```bash
rm spike_test.go
git status --short
```

Expected: nothing new to commit.

---

### Task 3: Colour values

**Files:**
- Create: `colour.go`
- Test: `colour_test.go`

**Interfaces:**
- Consumes: nothing
- Produces:
  - `type RGB struct{ R, G, B uint8 }`
  - `var Off, White RGB`
  - `func ParseRGB(s string) (RGB, error)`
  - unexported `func (c RGB) scale(limit uint8) RGB`

- [ ] **Step 1: Write the failing tests**

`colour_test.go`:

```go
package blinkstick

import "testing"

func TestParseRGB(t *testing.T) {
	tests := []struct {
		in   string
		want RGB
	}{
		{"#ff8800", RGB{255, 136, 0}},
		{"#FF8800", RGB{255, 136, 0}},
		{"#f80", RGB{255, 136, 0}},
		{"#000", Off},
		{"#ffffff", White},
		{"255,136,0", RGB{255, 136, 0}},
		{" 1, 2 ,3 ", RGB{1, 2, 3}},
	}
	for _, tt := range tests {
		got, err := ParseRGB(tt.in)
		if err != nil {
			t.Errorf("ParseRGB(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseRGB(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseRGBRejects(t *testing.T) {
	for _, in := range []string{
		"", "#", "#ff88", "#ff88000", "#gg0000", "ff8800",
		"1,2", "1,2,3,4", "256,0,0", "-1,0,0", "a,b,c",
	} {
		if got, err := ParseRGB(in); err == nil {
			t.Errorf("ParseRGB(%q) = %v, want error", in, got)
		}
	}
}

func TestScale(t *testing.T) {
	tests := []struct {
		c     RGB
		limit uint8
		want  RGB
	}{
		{White, 255, White},
		{White, 0, Off},
		{RGB{255, 128, 0}, 128, RGB{128, 64, 0}},
		{RGB{10, 20, 30}, 255, RGB{10, 20, 30}},
	}
	for _, tt := range tests {
		if got := tt.c.scale(tt.limit); got != tt.want {
			t.Errorf("%v.scale(%d) = %v, want %v", tt.c, tt.limit, got, tt.want)
		}
	}
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test -run 'ParseRGB|Scale' .`
Expected: FAIL to compile with `undefined: RGB`.

- [ ] **Step 3: Write `colour.go`**

```go
package blinkstick

import (
	"fmt"
	"strconv"
	"strings"
)

// RGB is a colour with 8 bits per channel.
type RGB struct{ R, G, B uint8 }

// Common values.
var (
	Off   = RGB{}
	White = RGB{R: 255, G: 255, B: 255}
)

// ParseRGB parses "#rgb", "#rrggbb" or "r,g,b" with decimal channels from
// 0 to 255.
func ParseRGB(s string) (RGB, error) {
	s = strings.TrimSpace(s)
	if hex, ok := strings.CutPrefix(s, "#"); ok {
		return parseHex(s, hex)
	}
	parts := strings.Split(s, ",")
	if len(parts) != 3 {
		return RGB{}, invalid(s)
	}
	var v [3]uint8
	for i, p := range parts {
		n, err := strconv.ParseUint(strings.TrimSpace(p), 10, 8)
		if err != nil {
			return RGB{}, invalid(s)
		}
		v[i] = uint8(n)
	}
	return RGB{R: v[0], G: v[1], B: v[2]}, nil
}

func parseHex(s, hex string) (RGB, error) {
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return RGB{}, invalid(s)
	}
	n, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return RGB{}, invalid(s)
	}
	return RGB{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n)}, nil
}

func invalid(s string) error {
	return fmt.Errorf("blinkstick: invalid colour %q", s)
}

// scale caps each channel at limit, keeping the hue: v * limit / 255.
func (c RGB) scale(limit uint8) RGB {
	f := func(v uint8) uint8 { return uint8(uint16(v) * uint16(limit) / 255) }
	return RGB{R: f(c.R), G: f(c.G), B: f(c.B)}
}
```

- [ ] **Step 4: Run the tests to see them pass**

Run: `go test -race -run 'ParseRGB|Scale' -v .`
Expected: PASS for all three tests.

- [ ] **Step 5: Commit (after the user approves)**

```bash
git add colour.go colour_test.go
git commit -S -m "Added RGB type with hex and decimal parsing"
```

---

### Task 4: Model table and device identity

**Files:**
- Create: `model.go`
- Test: `model_test.go`

**Interfaces:**
- Consumes: nothing
- Produces:
  - `type Model struct{ Name string; LEDs int }`
  - `var Nano, Square Model`
  - unexported `var unknownModel Model`
  - `type Info struct{ Serial, Version, Manufacturer, Product string; Model Model }`
  - unexported `type deviceInfo struct{ serial, manufacturer, product string; release uint16 }`
  - unexported `func newInfo(di deviceInfo) Info`

- [ ] **Step 1: Write the failing tests**

`model_test.go`:

```go
package blinkstick

import "testing"

func TestNewInfo(t *testing.T) {
	tests := []struct {
		name        string
		di          deviceInfo
		wantModel   Model
		wantVersion string
	}{
		{"nano", deviceInfo{serial: "BS072777-3.0", release: 0x0202}, Nano, "3.0"},
		{"square", deviceInfo{serial: "BS073788-3.1", release: 0x0201}, Square, "3.1"},
		{"pro", deviceInfo{serial: "BS000001-2.0", release: 0x0200}, unknownModel, "2.0"},
		{"flex", deviceInfo{serial: "BS000002-3.0", release: 0x0203}, unknownModel, "3.0"},
		{"v1 with square release", deviceInfo{serial: "BS000003-1.0", release: 0x0201}, unknownModel, "1.0"},
		{"garbage serial", deviceInfo{serial: "garbage", release: 0x0202}, unknownModel, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := newInfo(tt.di)
			if got.Model != tt.wantModel {
				t.Errorf("Model = %v, want %v", got.Model, tt.wantModel)
			}
			if got.Version != tt.wantVersion {
				t.Errorf("Version = %q, want %q", got.Version, tt.wantVersion)
			}
			if got.Serial != tt.di.serial {
				t.Errorf("Serial = %q, want %q", got.Serial, tt.di.serial)
			}
		})
	}
}

func TestNewInfoCopiesStrings(t *testing.T) {
	got := newInfo(deviceInfo{
		serial:       "BS072777-3.0",
		manufacturer: "Agile Innovative Ltd",
		product:      "BlinkStick Nano",
		release:      0x0202,
	})
	if got.Manufacturer != "Agile Innovative Ltd" || got.Product != "BlinkStick Nano" {
		t.Errorf("got %+v", got)
	}
}

func TestModelLEDs(t *testing.T) {
	if Nano.LEDs != 2 || Square.LEDs != 8 || unknownModel.LEDs != 0 {
		t.Errorf("LEDs: Nano %d, Square %d, unknown %d", Nano.LEDs, Square.LEDs, unknownModel.LEDs)
	}
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test -run 'NewInfo|ModelLEDs' .`
Expected: FAIL to compile with `undefined: deviceInfo`.

- [ ] **Step 3: Write `model.go`**

```go
package blinkstick

import "strings"

// Model describes a BlinkStick product. Models differ only in data, so
// supporting another product means adding a row to the models table.
type Model struct {
	Name string // "Nano", "Square" or "unknown"
	LEDs int    // addressable LEDs on channel 0
}

// Supported models.
var (
	Nano   = Model{Name: "Nano", LEDs: 2}
	Square = Model{Name: "Square", LEDs: 8}

	unknownModel = Model{Name: "unknown"}
)

type modelKey struct {
	major   string // serial major version, the "3" in "BS072777-3.0"
	release uint16 // USB bcdDevice
}

// models maps hardware identity to a Model. Upstream libraries map
// bcdDevice 0x0201 to the Strip, but a Square (BS073788-3.1) reports 0x0201
// too, so the Strip may share this value.
var models = map[modelKey]Model{
	{major: "3", release: 0x0202}: Nano,
	{major: "3", release: 0x0201}: Square,
}

// Info identifies an attached BlinkStick.
type Info struct {
	Serial       string // for example "BS072777-3.0"
	Version      string // firmware version from the serial, for example "3.0"
	Manufacturer string
	Product      string
	Model        Model
}

// deviceInfo is what the HID layer reports about a device.
type deviceInfo struct {
	serial, manufacturer, product string
	release                       uint16
}

func newInfo(di deviceInfo) Info {
	_, version, _ := strings.Cut(di.serial, "-")
	major, _, _ := strings.Cut(version, ".")
	m, ok := models[modelKey{major: major, release: di.release}]
	if !ok {
		m = unknownModel
	}
	return Info{
		Serial:       di.serial,
		Version:      version,
		Manufacturer: di.manufacturer,
		Product:      di.product,
		Model:        m,
	}
}
```

- [ ] **Step 4: Run the tests to see them pass**

Run: `go test -race -run 'NewInfo|ModelLEDs' -v .`
Expected: PASS.

- [ ] **Step 5: Commit (after the user approves)**

```bash
git add model.go model_test.go
git commit -S -m "Added model table for Nano and Square detection"
```

---

### Task 5: Report encoding

**Files:**
- Create: `report.go`
- Test: `report_test.go`

**Interfaces:**
- Consumes: `RGB` (Task 3)
- Produces:
  - unexported constants `vendorID`, `productID` (uint16)
  - `reportInfo1 = 2`, `reportInfo2 = 3`, `reportFrame = 6`
  - `frameLEDs = 8`, `frameReportSize = 26`, `infoBlockSize = 32`, `infoReportSize = 33`
  - `func encodeFrame(leds []RGB) []byte` (len(leds) must be 8 or fewer)
  - `func decodeFrame(buf []byte, n int) []RGB`

- [ ] **Step 1: Write the failing tests**

`report_test.go`:

```go
package blinkstick

import (
	"slices"
	"testing"
)

func TestEncodeFrameLayout(t *testing.T) {
	got := encodeFrame([]RGB{{R: 0x11, G: 0x22, B: 0x33}, {R: 0x44, G: 0x55, B: 0x66}})
	if len(got) != frameReportSize || frameReportSize != 26 {
		t.Fatalf("len = %d, want 26", len(got))
	}
	want := []byte{0x06, 0x00, 0x22, 0x11, 0x33, 0x55, 0x44, 0x66}
	if !slices.Equal(got[:8], want) {
		t.Errorf("head = % x, want % x", got[:8], want)
	}
	for i, v := range got[8:] {
		if v != 0 {
			t.Fatalf("byte %d = %#x, want 0 for unused slots", i+8, v)
		}
	}
}

func TestDecodeFrameRoundTrip(t *testing.T) {
	leds := []RGB{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}, {10, 11, 12}, {13, 14, 15}, {16, 17, 18}, {19, 20, 21}, {22, 23, 24}}
	if got := decodeFrame(encodeFrame(leds), 8); !slices.Equal(got, leds) {
		t.Errorf("round trip = %v, want %v", got, leds)
	}
	if got := decodeFrame(encodeFrame(leds), 2); !slices.Equal(got, leds[:2]) {
		t.Errorf("first two = %v, want %v", got, leds[:2])
	}
}

func TestReportSizes(t *testing.T) {
	if infoReportSize != 33 || infoBlockSize != 32 {
		t.Errorf("info report %d, block %d", infoReportSize, infoBlockSize)
	}
	if vendorID != 0x20A0 || productID != 0x41E5 {
		t.Errorf("USB ID %04x:%04x", vendorID, productID)
	}
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test -run 'Frame|ReportSizes' .`
Expected: FAIL to compile with `undefined: encodeFrame`.

- [ ] **Step 3: Write `report.go`**

```go
package blinkstick

// USB identifiers shared by every BlinkStick model.
const (
	vendorID  uint16 = 0x20A0
	productID uint16 = 0x41E5
)

// Feature report IDs and sizes. Sizes include the leading report ID byte
// and match the device's HID report descriptor exactly.
const (
	reportInfo1 = 2
	reportInfo2 = 3
	reportFrame = 6

	frameLEDs       = 8
	frameReportSize = 2 + frameLEDs*3 // ID, channel, 8 x (G, R, B)
	infoBlockSize   = 32
	infoReportSize  = 1 + infoBlockSize
)

// encodeFrame builds report 6 for channel 0. The device expects each LED as
// G, R, B. Slots beyond len(leds) are zero. len(leds) must not exceed
// frameLEDs.
func encodeFrame(leds []RGB) []byte {
	buf := make([]byte, frameReportSize)
	buf[0] = reportFrame
	for i, c := range leds {
		buf[2+i*3], buf[3+i*3], buf[4+i*3] = c.G, c.R, c.B
	}
	return buf
}

// decodeFrame reads the first n LEDs out of a report 6 buffer.
func decodeFrame(buf []byte, n int) []RGB {
	leds := make([]RGB, n)
	for i := range leds {
		leds[i] = RGB{G: buf[2+i*3], R: buf[3+i*3], B: buf[4+i*3]}
	}
	return leds
}
```

- [ ] **Step 4: Run the tests to see them pass**

Run: `go test -race -run 'Frame|ReportSizes' -v .`
Expected: PASS.

- [ ] **Step 5: Commit (after the user approves)**

```bash
git add report.go report_test.go
git commit -S -m "Added report 6 frame encoding in GRB order"
```

---

### Task 6: Device writes, retries and close

**Files:**
- Create: `device.go`, `fake_test.go`
- Test: `device_test.go`

**Interfaces:**
- Consumes: `RGB`, `scale` (Task 3), `Info`, `Model` (Task 4), `encodeFrame`, report constants (Task 5)
- Produces:
  - `var ErrNotFound, ErrUnsupported, ErrOutOfRange, ErrClosed error`
  - unexported `type transport interface{ SendFeatureReport([]byte) (int, error); GetFeatureReport([]byte) (int, error); Close() error }`
  - `type Device struct` with unexported `mu sync.Mutex`, `t transport`, `info Info`, `limit uint8`
  - unexported `func newDevice(t transport, info Info) *Device`
  - `(*Device).Info() Info`, `Close() error`, `SetBrightnessLimit(limit uint8)`, `SetAll(c RGB) error`, `Off() error`, `SetFrame(leds []RGB) error`
  - unexported `(*Device).writeFrameLocked(leds []RGB) error` (no scaling), `sendLocked(p []byte) error`, `getLocked(p []byte) error`
  - unexported `func fill(n int, c RGB) []RGB`
  - test helpers in `fake_test.go`: `newFakeTransport()`, `(*fakeTransport).frame(n int) []RGB`, `.sentFrames(n int) [][]RGB`, `.sendCount() int`, fields `fail int`, `closed bool`; `openFake(m Model) (*Device, *fakeTransport)`; `var errBusy`

- [ ] **Step 1: Write the fake**

`fake_test.go`:

```go
package blinkstick

import (
	"bytes"
	"errors"
	"sync"
)

var errBusy = errors.New("general error")

// fakeTransport behaves like BlinkStick firmware: a get returns the last
// report sent with the same ID, or zeros.
type fakeTransport struct {
	mu      sync.Mutex
	reports map[byte][]byte
	sends   [][]byte
	fail    int // fail this many transfers before succeeding
	closed  bool
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{reports: map[byte][]byte{}}
}

func (f *fakeTransport) SendFeatureReport(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sends = append(f.sends, bytes.Clone(p))
	if f.fail > 0 {
		f.fail--
		return 0, errBusy
	}
	f.reports[p[0]] = bytes.Clone(p)
	return len(p), nil
}

func (f *fakeTransport) GetFeatureReport(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.fail > 0 {
		f.fail--
		return 0, errBusy
	}
	clear(p[1:])
	if stored := f.reports[p[0]]; stored != nil {
		copy(p[1:], stored[1:])
	}
	return len(p), nil
}

func (f *fakeTransport) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

// frame decodes the last report 6 the fake accepted.
func (f *fakeTransport) frame(n int) []RGB {
	f.mu.Lock()
	defer f.mu.Unlock()
	buf := f.reports[reportFrame]
	if buf == nil {
		buf = make([]byte, frameReportSize)
	}
	return decodeFrame(buf, n)
}

// sentFrames decodes every report 6 send attempt, in order.
func (f *fakeTransport) sentFrames(n int) [][]RGB {
	f.mu.Lock()
	defer f.mu.Unlock()
	var frames [][]RGB
	for _, p := range f.sends {
		if p[0] == reportFrame {
			frames = append(frames, decodeFrame(p, n))
		}
	}
	return frames
}

func (f *fakeTransport) sendCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sends)
}

func openFake(m Model) (*Device, *fakeTransport) {
	ft := newFakeTransport()
	return newDevice(ft, Info{Serial: "BS000001-3.0", Version: "3.0", Model: m}), ft
}
```

- [ ] **Step 2: Write the failing tests**

`device_test.go`:

```go
package blinkstick

import (
	"errors"
	"slices"
	"testing"
)

func TestSetAllSendsFullFrame(t *testing.T) {
	d, ft := openFake(Square)
	c := RGB{1, 2, 3}
	if err := d.SetAll(c); err != nil {
		t.Fatalf("SetAll: %v", err)
	}
	if ft.sendCount() != 1 {
		t.Fatalf("sends = %d, want 1", ft.sendCount())
	}
	if got := ft.frame(8); !slices.Equal(got, fill(8, c)) {
		t.Errorf("frame = %v, want all %v", got, c)
	}
}

func TestSetAllNanoLeavesUnusedSlotsZero(t *testing.T) {
	d, ft := openFake(Nano)
	if err := d.SetAll(White); err != nil {
		t.Fatalf("SetAll: %v", err)
	}
	want := append(fill(2, White), fill(6, Off)...)
	if got := ft.frame(8); !slices.Equal(got, want) {
		t.Errorf("frame = %v, want %v", got, want)
	}
}

func TestOff(t *testing.T) {
	d, ft := openFake(Square)
	d.SetAll(White)
	if err := d.Off(); err != nil {
		t.Fatalf("Off: %v", err)
	}
	if got := ft.frame(8); !slices.Equal(got, fill(8, Off)) {
		t.Errorf("frame = %v, want all off", got)
	}
}

func TestSetFrame(t *testing.T) {
	d, ft := openFake(Nano)
	leds := []RGB{{R: 255}, {B: 255}}
	if err := d.SetFrame(leds); err != nil {
		t.Fatalf("SetFrame: %v", err)
	}
	if got := ft.frame(2); !slices.Equal(got, leds) {
		t.Errorf("frame = %v, want %v", got, leds)
	}
}

func TestSetFrameWrongLength(t *testing.T) {
	d, ft := openFake(Nano)
	for _, leds := range [][]RGB{nil, fill(1, White), fill(3, White)} {
		if err := d.SetFrame(leds); !errors.Is(err, ErrOutOfRange) {
			t.Errorf("SetFrame(%d LEDs) = %v, want ErrOutOfRange", len(leds), err)
		}
	}
	if ft.sendCount() != 0 {
		t.Errorf("sends = %d, want 0", ft.sendCount())
	}
}

func TestRetriesTransientFailure(t *testing.T) {
	d, ft := openFake(Square)
	ft.fail = 2
	if err := d.SetAll(White); err != nil {
		t.Fatalf("SetAll: %v", err)
	}
	if ft.sendCount() != 3 {
		t.Errorf("sends = %d, want 3", ft.sendCount())
	}
}

func TestGivesUpAfterThreeAttempts(t *testing.T) {
	d, ft := openFake(Square)
	ft.fail = 5
	err := d.SetAll(White)
	if !errors.Is(err, errBusy) {
		t.Fatalf("err = %v, want wrapping %v", err, errBusy)
	}
	if ft.sendCount() != 3 {
		t.Errorf("sends = %d, want 3", ft.sendCount())
	}
}

func TestBrightnessLimitScalesWrites(t *testing.T) {
	d, ft := openFake(Square)
	d.SetBrightnessLimit(128)
	if err := d.SetAll(White); err != nil {
		t.Fatalf("SetAll: %v", err)
	}
	if got := ft.frame(8); !slices.Equal(got, fill(8, RGB{128, 128, 128})) {
		t.Errorf("frame = %v, want all 128", got)
	}
}

func TestClose(t *testing.T) {
	d, ft := openFake(Square)
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !ft.closed {
		t.Error("transport not closed")
	}
	if err := d.Close(); !errors.Is(err, ErrClosed) {
		t.Errorf("second Close = %v, want ErrClosed", err)
	}
	if err := d.SetAll(White); !errors.Is(err, ErrClosed) {
		t.Errorf("SetAll after Close = %v, want ErrClosed", err)
	}
}

func TestInfo(t *testing.T) {
	d, _ := openFake(Nano)
	if got := d.Info(); got.Model != Nano || got.Serial != "BS000001-3.0" {
		t.Errorf("Info = %+v", got)
	}
}
```

- [ ] **Step 3: Run them to see them fail**

Run: `go test .`
Expected: FAIL to compile with `undefined: newDevice`.

- [ ] **Step 4: Write `device.go`**

```go
package blinkstick

import (
	"errors"
	"fmt"
	"sync"
)

// Errors returned by this package. Check them with errors.Is.
var (
	ErrNotFound    = errors.New("blinkstick: device not found")
	ErrUnsupported = errors.New("blinkstick: unsupported model")
	ErrOutOfRange  = errors.New("blinkstick: out of range")
	ErrClosed      = errors.New("blinkstick: device closed")
)

// transferAttempts covers the firmware being busy straight after a write and
// intermittent IOHIDDeviceSetReport failures on macOS (go-hid issue #15).
const transferAttempts = 3

// transport is the slice of a HID device that Device needs.
type transport interface {
	SendFeatureReport(p []byte) (int, error)
	GetFeatureReport(p []byte) (int, error)
	Close() error
}

// Device is an open BlinkStick. Its methods are safe for concurrent use,
// but two effects running at once on one Device interleave their frames.
type Device struct {
	mu    sync.Mutex
	t     transport // nil once closed
	info  Info
	limit uint8
}

func newDevice(t transport, info Info) *Device {
	return &Device{t: t, info: info, limit: 255}
}

// Info returns the identity of the device.
func (d *Device) Info() Info {
	return d.info
}

// Close releases the device. Using a Device after Close returns ErrClosed.
func (d *Device) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.t == nil {
		return ErrClosed
	}
	err := d.t.Close()
	d.t = nil
	return err
}

// SetBrightnessLimit caps every channel written from now on, scaling each
// as v * limit / 255. 255, the default, means no limit. Frame and LED return
// the scaled values stored on the device.
func (d *Device) SetBrightnessLimit(limit uint8) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.limit = limit
}

// SetAll sets every LED to c.
func (d *Device) SetAll(c RGB) error {
	return d.SetFrame(fill(d.info.Model.LEDs, c))
}

// Off turns every LED off.
func (d *Device) Off() error {
	return d.SetAll(Off)
}

// SetFrame sets every LED at once. len(leds) must equal Info().Model.LEDs.
func (d *Device) SetFrame(leds []RGB) error {
	if n := d.info.Model.LEDs; len(leds) != n {
		return fmt.Errorf("%w: frame has %d LEDs, %s has %d",
			ErrOutOfRange, len(leds), d.info.Model.Name, n)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	scaled := make([]RGB, len(leds))
	for i, c := range leds {
		scaled[i] = c.scale(d.limit)
	}
	return d.writeFrameLocked(scaled)
}

// writeFrameLocked sends LED values as given, without scaling. d.mu must be
// held.
func (d *Device) writeFrameLocked(leds []RGB) error {
	return d.sendLocked(encodeFrame(leds))
}

func (d *Device) sendLocked(p []byte) error {
	return d.transferLocked("send", p[0], func(t transport) (int, error) {
		return t.SendFeatureReport(p)
	})
}

func (d *Device) getLocked(p []byte) error {
	return d.transferLocked("get", p[0], func(t transport) (int, error) {
		return t.GetFeatureReport(p)
	})
}

// transferLocked runs f up to transferAttempts times. d.mu must be held.
func (d *Device) transferLocked(op string, id byte, f func(transport) (int, error)) error {
	if d.t == nil {
		return ErrClosed
	}
	var err error
	for range transferAttempts {
		if _, err = f(d.t); err == nil {
			return nil
		}
	}
	return fmt.Errorf("blinkstick: %s report %d failed after %d attempts: %w",
		op, id, transferAttempts, err)
}

// fill returns n copies of c.
func fill(n int, c RGB) []RGB {
	leds := make([]RGB, n)
	for i := range leds {
		leds[i] = c
	}
	return leds
}
```

- [ ] **Step 5: Run the tests to see them pass**

Run: `go test -race -v .`
Expected: PASS for every test so far.

- [ ] **Step 6: Commit (after the user approves)**

```bash
git add device.go device_test.go fake_test.go
git commit -S -m "Added Device with frame writes, retries and close"
```

---

### Task 7: Reading LEDs back and setting one LED

**Files:**
- Modify: `device.go` (append methods)
- Test: `device_test.go` (append tests)

**Interfaces:**
- Consumes: `Device` internals from Task 6, `decodeFrame` (Task 5)
- Produces:
  - `(*Device).Frame() ([]RGB, error)`, `LED(i int) (RGB, error)`, `SetLED(i int, c RGB) error`
  - unexported `(*Device).readFrameLocked() ([]RGB, error)`, `checkIndex(i int) error`

- [ ] **Step 1: Write the failing tests**

Append to `device_test.go` (add `"sync"` to its imports):

```go
func TestFrameReadsBack(t *testing.T) {
	d, _ := openFake(Nano)
	leds := []RGB{{R: 10}, {G: 20}}
	d.SetFrame(leds)
	got, err := d.Frame()
	if err != nil {
		t.Fatalf("Frame: %v", err)
	}
	if !slices.Equal(got, leds) {
		t.Errorf("Frame = %v, want %v", got, leds)
	}
}

func TestLED(t *testing.T) {
	d, _ := openFake(Nano)
	d.SetFrame([]RGB{{R: 10}, {G: 20}})
	got, err := d.LED(1)
	if err != nil {
		t.Fatalf("LED: %v", err)
	}
	if got != (RGB{G: 20}) {
		t.Errorf("LED(1) = %v, want {0 20 0}", got)
	}
}

func TestSetLEDKeepsOthers(t *testing.T) {
	d, ft := openFake(Square)
	d.SetAll(RGB{B: 50})
	if err := d.SetLED(3, RGB{R: 255}); err != nil {
		t.Fatalf("SetLED: %v", err)
	}
	want := fill(8, RGB{B: 50})
	want[3] = RGB{R: 255}
	if got := ft.frame(8); !slices.Equal(got, want) {
		t.Errorf("frame = %v, want %v", got, want)
	}
}

func TestSetLEDDoesNotRescaleOthers(t *testing.T) {
	d, ft := openFake(Square)
	d.SetBrightnessLimit(128)
	d.SetAll(White)
	if err := d.SetLED(0, White); err != nil {
		t.Fatalf("SetLED: %v", err)
	}
	if got := ft.frame(8); !slices.Equal(got, fill(8, RGB{128, 128, 128})) {
		t.Errorf("frame = %v, want all 128", got)
	}
}

func TestIndexOutOfRange(t *testing.T) {
	d, ft := openFake(Nano)
	for _, i := range []int{-1, 2, 8} {
		if _, err := d.LED(i); !errors.Is(err, ErrOutOfRange) {
			t.Errorf("LED(%d) = %v, want ErrOutOfRange", i, err)
		}
		if err := d.SetLED(i, White); !errors.Is(err, ErrOutOfRange) {
			t.Errorf("SetLED(%d) = %v, want ErrOutOfRange", i, err)
		}
	}
	if ft.sendCount() != 0 {
		t.Errorf("sends = %d, want 0", ft.sendCount())
	}
}

func TestSetLEDConcurrent(t *testing.T) {
	d, ft := openFake(Square)
	var wg sync.WaitGroup
	for i := range Square.LEDs {
		wg.Go(func() {
			if err := d.SetLED(i, White); err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
	if got := ft.frame(8); !slices.Equal(got, fill(8, White)) {
		t.Errorf("frame = %v, want all white (a lost update means SetLED is not atomic)", got)
	}
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test .`
Expected: FAIL to compile with `d.Frame undefined`.

- [ ] **Step 3: Append to `device.go`**

```go
// Frame reads every LED back from the device.
func (d *Device) Frame() ([]RGB, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.readFrameLocked()
}

// LED reads LED i back from the device.
func (d *Device) LED(i int) (RGB, error) {
	if err := d.checkIndex(i); err != nil {
		return RGB{}, err
	}
	leds, err := d.Frame()
	if err != nil {
		return RGB{}, err
	}
	return leds[i], nil
}

// SetLED sets LED i to c and leaves the others as they are. It reads the
// frame back and rewrites it, because setting a single LED directly (report
// 5) needs a firmware mode that is unverified on v3 hardware.
func (d *Device) SetLED(i int, c RGB) error {
	if err := d.checkIndex(i); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	leds, err := d.readFrameLocked()
	if err != nil {
		return err
	}
	leds[i] = c.scale(d.limit)
	return d.writeFrameLocked(leds)
}

// readFrameLocked reads report 6. d.mu must be held.
func (d *Device) readFrameLocked() ([]RGB, error) {
	buf := make([]byte, frameReportSize)
	buf[0] = reportFrame
	if err := d.getLocked(buf); err != nil {
		return nil, err
	}
	return decodeFrame(buf, d.info.Model.LEDs), nil
}

func (d *Device) checkIndex(i int) error {
	if n := d.info.Model.LEDs; i < 0 || i >= n {
		return fmt.Errorf("%w: LED %d, %s has %d", ErrOutOfRange, i, d.info.Model.Name, n)
	}
	return nil
}
```

- [ ] **Step 4: Run the tests to see them pass**

Run: `go test -race -count=3 -v .`
Expected: PASS, including `TestSetLEDConcurrent` with no race reports.

- [ ] **Step 5: Commit (after the user approves)**

```bash
git add device.go device_test.go
git commit -S -m "Added LED read back and single LED updates"
```

---

### Task 8: Discovery and the go-hid adaptor

**Files:**
- Create: `hid.go`, `hid_test.go`, `discovery_test.go`
- Modify: `device.go` (append discovery), `fake_test.go` (append fake backend)
- Modify: `go.mod`, `go.sum` (via `go mod tidy`)

**Interfaces:**
- Consumes: `newInfo`, `deviceInfo`, `unknownModel` (Task 4), `vendorID`, `productID` (Task 5), `newDevice`, `transport`, errors (Task 6)
- Produces:
  - `func List() ([]Info, error)`, `func Open() (*Device, error)`, `func OpenSerial(serial string) (*Device, error)`
  - unexported `type backend interface{ list() ([]deviceInfo, error); open(serial string) (transport, deviceInfo, error) }`, `var sys backend`
  - unexported `hidBackend`, `hidTransport`, `acquireHID() error`, `releaseHID() error`, `var hidInit, hidExit func() error`
  - test helpers: `fakeBackend`, `fakeDevice`, `useBackend(t, b)`, `nanoInfo`, `squareInfo`, `flexInfo`

- [ ] **Step 1: Append the fake backend to `fake_test.go`**

Add `"fmt"` and `"testing"` to its imports, then append:

```go
type fakeDevice struct {
	info deviceInfo
	t    *fakeTransport
}

type fakeBackend struct {
	devices []fakeDevice
}

func (b *fakeBackend) list() ([]deviceInfo, error) {
	var dis []deviceInfo
	for _, d := range b.devices {
		dis = append(dis, d.info)
	}
	return dis, nil
}

func (b *fakeBackend) open(serial string) (transport, deviceInfo, error) {
	for _, d := range b.devices {
		if serial == "" || d.info.serial == serial {
			return d.t, d.info, nil
		}
	}
	return nil, deviceInfo{}, fmt.Errorf("%w: %s", ErrNotFound, serial)
}

// useBackend swaps the package backend for the length of the test.
func useBackend(t *testing.T, b backend) {
	orig := sys
	sys = b
	t.Cleanup(func() { sys = orig })
}

var (
	nanoInfo = deviceInfo{serial: "BS072777-3.0", manufacturer: "Agile Innovative Ltd",
		product: "BlinkStick Nano", release: 0x0202}
	squareInfo = deviceInfo{serial: "BS073788-3.1", manufacturer: "Agile Innovative Ltd",
		product: "BlinkStick", release: 0x0201}
	flexInfo = deviceInfo{serial: "BS000002-3.0", manufacturer: "Agile Innovative Ltd",
		product: "BlinkStick Flex", release: 0x0203}
)

// threeSticks returns a backend holding a Nano, a Square and a Flex.
func threeSticks() (*fakeBackend, map[string]*fakeTransport) {
	ts := map[string]*fakeTransport{
		"nano": newFakeTransport(), "square": newFakeTransport(), "flex": newFakeTransport(),
	}
	return &fakeBackend{devices: []fakeDevice{
		{nanoInfo, ts["nano"]}, {squareInfo, ts["square"]}, {flexInfo, ts["flex"]},
	}}, ts
}
```

- [ ] **Step 2: Write the failing discovery tests**

`discovery_test.go`:

```go
package blinkstick

import (
	"errors"
	"slices"
	"testing"
)

func TestList(t *testing.T) {
	b, _ := threeSticks()
	useBackend(t, b)
	infos, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var got []string
	for _, i := range infos {
		got = append(got, i.Serial+" "+i.Model.Name)
	}
	want := []string{"BS072777-3.0 Nano", "BS073788-3.1 Square", "BS000002-3.0 unknown"}
	if !slices.Equal(got, want) {
		t.Errorf("List = %v, want %v", got, want)
	}
}

func TestOpenFirst(t *testing.T) {
	b, _ := threeSticks()
	useBackend(t, b)
	d, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if d.Info().Model != Nano {
		t.Errorf("Open model = %v, want Nano", d.Info().Model)
	}
}

func TestOpenSerial(t *testing.T) {
	b, _ := threeSticks()
	useBackend(t, b)
	d, err := OpenSerial("BS073788-3.1")
	if err != nil {
		t.Fatalf("OpenSerial: %v", err)
	}
	if d.Info().Model != Square || d.Info().Product != "BlinkStick" {
		t.Errorf("Info = %+v", d.Info())
	}
}

func TestOpenSerialNotFound(t *testing.T) {
	b, _ := threeSticks()
	useBackend(t, b)
	for _, s := range []string{"BS999999-3.0", ""} {
		if _, err := OpenSerial(s); !errors.Is(err, ErrNotFound) {
			t.Errorf("OpenSerial(%q) = %v, want ErrNotFound", s, err)
		}
	}
}

func TestOpenNoDevices(t *testing.T) {
	useBackend(t, &fakeBackend{})
	if _, err := Open(); !errors.Is(err, ErrNotFound) {
		t.Errorf("Open = %v, want ErrNotFound", err)
	}
}

func TestOpenUnsupportedClosesTransport(t *testing.T) {
	b, ts := threeSticks()
	useBackend(t, b)
	if _, err := OpenSerial("BS000002-3.0"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("OpenSerial(flex) = %v, want ErrUnsupported", err)
	}
	if !ts["flex"].closed {
		t.Error("flex transport left open")
	}
}

func TestTwoDevicesAreIndependent(t *testing.T) {
	b, ts := threeSticks()
	useBackend(t, b)
	nano, err := OpenSerial("BS072777-3.0")
	if err != nil {
		t.Fatal(err)
	}
	square, err := OpenSerial("BS073788-3.1")
	if err != nil {
		t.Fatal(err)
	}
	nano.SetAll(RGB{R: 255})
	square.SetAll(RGB{G: 255})
	if err := nano.Close(); err != nil {
		t.Fatal(err)
	}
	if err := square.SetAll(RGB{B: 255}); err != nil {
		t.Fatalf("square after closing nano: %v", err)
	}
	if got := ts["nano"].frame(2); !slices.Equal(got, fill(2, RGB{R: 255})) {
		t.Errorf("nano frame = %v", got)
	}
	if got := ts["square"].frame(8); !slices.Equal(got, fill(8, RGB{B: 255})) {
		t.Errorf("square frame = %v", got)
	}
}
```

- [ ] **Step 3: Write the failing reference counting tests**

`hid_test.go`:

```go
package blinkstick

import (
	"errors"
	"testing"
)

// stubHID replaces hidapi Init and Exit with counters.
func stubHID(t *testing.T) (inits, exits *int) {
	origInit, origExit := hidInit, hidExit
	var i, e int
	hidInit = func() error { i++; return nil }
	hidExit = func() error { e++; return nil }
	t.Cleanup(func() {
		hidInit, hidExit = origInit, origExit
		hidRefs = 0
	})
	return &i, &e
}

func TestHIDRefCounting(t *testing.T) {
	inits, exits := stubHID(t)
	acquireHID()
	acquireHID()
	if *inits != 1 {
		t.Fatalf("inits = %d after two acquires, want 1", *inits)
	}
	releaseHID()
	if *exits != 0 {
		t.Fatalf("exits = %d with one device still open, want 0", *exits)
	}
	releaseHID()
	if *exits != 1 {
		t.Fatalf("exits = %d after last release, want 1", *exits)
	}
	acquireHID()
	if *inits != 2 {
		t.Errorf("inits = %d after reacquire, want 2", *inits)
	}
}

func TestHIDInitFailureDoesNotCount(t *testing.T) {
	stubHID(t)
	boom := errors.New("boom")
	hidInit = func() error { return boom }
	if err := acquireHID(); !errors.Is(err, boom) {
		t.Fatalf("acquireHID = %v, want %v", err, boom)
	}
	if hidRefs != 0 {
		t.Errorf("hidRefs = %d, want 0", hidRefs)
	}
}

func TestHIDReleaseWithoutAcquire(t *testing.T) {
	_, exits := stubHID(t)
	if err := releaseHID(); err != nil {
		t.Fatalf("releaseHID: %v", err)
	}
	if *exits != 0 || hidRefs != 0 {
		t.Errorf("exits = %d, hidRefs = %d, want 0, 0", *exits, hidRefs)
	}
}
```

- [ ] **Step 4: Run them to see them fail**

Run: `go test .`
Expected: FAIL to compile with `undefined: sys` and `undefined: hidInit`.

- [ ] **Step 5: Append discovery to `device.go`**

```go
// backend finds and opens BlinkSticks. hidBackend is the real one; tests
// swap in a fake.
type backend interface {
	list() ([]deviceInfo, error)
	open(serial string) (transport, deviceInfo, error) // "" opens the first found
}

var sys backend = hidBackend{}

// List returns every attached BlinkStick, including models this package does
// not support yet (their Model.Name is "unknown").
func List() ([]Info, error) {
	dis, err := sys.list()
	if err != nil {
		return nil, err
	}
	infos := make([]Info, len(dis))
	for i, di := range dis {
		infos[i] = newInfo(di)
	}
	return infos, nil
}

// Open opens the first BlinkStick found.
func Open() (*Device, error) {
	return open("")
}

// OpenSerial opens the BlinkStick with the given serial, as reported by
// List.
func OpenSerial(serial string) (*Device, error) {
	if serial == "" {
		return nil, fmt.Errorf("%w: empty serial", ErrNotFound)
	}
	return open(serial)
}

func open(serial string) (*Device, error) {
	t, di, err := sys.open(serial)
	if err != nil {
		return nil, err
	}
	info := newInfo(di)
	if info.Model == unknownModel {
		t.Close()
		return nil, fmt.Errorf("%w: %s (release %#04x)", ErrUnsupported, info.Serial, di.release)
	}
	return newDevice(t, info), nil
}
```

- [ ] **Step 6: Write `hid.go`**

```go
package blinkstick

import (
	"errors"
	"fmt"
	"sync"

	"github.com/sstallion/go-hid"
)

// hidapi must be initialised before use and finalised after the last device
// closes. Several Devices can be open at once, so Init and Exit are
// reference counted. hidInit and hidExit are swapped in tests.
var (
	hidMu   sync.Mutex
	hidRefs int
	hidInit = hid.Init
	hidExit = hid.Exit
)

func acquireHID() error {
	hidMu.Lock()
	defer hidMu.Unlock()
	if hidRefs == 0 {
		if err := hidInit(); err != nil {
			return fmt.Errorf("blinkstick: hid init: %w", err)
		}
	}
	hidRefs++
	return nil
}

func releaseHID() error {
	hidMu.Lock()
	defer hidMu.Unlock()
	if hidRefs == 0 {
		return nil
	}
	hidRefs--
	if hidRefs == 0 {
		return hidExit()
	}
	return nil
}

type hidBackend struct{}

// list enumerates BlinkSticks only. Filtering by vendor and product ID inside
// hidapi keeps other devices' strings out of Go, which avoids a panic on
// macOS when an unrelated device has a malformed string (go-hid issue #18).
func (hidBackend) list() ([]deviceInfo, error) {
	if err := acquireHID(); err != nil {
		return nil, err
	}
	defer releaseHID()
	var dis []deviceInfo
	err := hid.Enumerate(vendorID, productID, func(hi *hid.DeviceInfo) error {
		dis = append(dis, fromHID(hi))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("blinkstick: enumerate: %w", err)
	}
	return dis, nil
}

func (hidBackend) open(serial string) (transport, deviceInfo, error) {
	if err := acquireHID(); err != nil {
		return nil, deviceInfo{}, err
	}
	var dev *hid.Device
	var err error
	if serial == "" {
		dev, err = hid.OpenFirst(vendorID, productID)
	} else {
		dev, err = hid.Open(vendorID, productID, serial)
	}
	if err != nil {
		releaseHID()
		return nil, deviceInfo{}, fmt.Errorf("%w: %q: %v", ErrNotFound, serial, err)
	}
	hi, err := dev.GetDeviceInfo()
	if err != nil {
		dev.Close()
		releaseHID()
		return nil, deviceInfo{}, fmt.Errorf("blinkstick: device info: %w", err)
	}
	return hidTransport{dev}, fromHID(hi), nil
}

func fromHID(hi *hid.DeviceInfo) deviceInfo {
	return deviceInfo{
		serial:       hi.SerialNbr,
		manufacturer: hi.MfrStr,
		product:      hi.ProductStr,
		release:      hi.ReleaseNbr,
	}
}

// hidTransport releases hidapi along with the device.
type hidTransport struct {
	*hid.Device
}

func (t hidTransport) Close() error {
	return errors.Join(t.Device.Close(), releaseHID())
}
```

- [ ] **Step 7: Tidy the module and run the tests**

```bash
go mod tidy
go vet ./...
go test -race -count=3 -v .
```

Expected: `go.mod` still requires `github.com/sstallion/go-hid v0.15.0`. Vet is clean. All tests pass three times with no race reports.

- [ ] **Step 8: Commit (after the user approves)**

```bash
git add device.go hid.go hid_test.go discovery_test.go fake_test.go go.mod go.sum
git commit -S -m "Added device discovery and reference-counted hidapi"
```

---

### Task 9: Info blocks

**Files:**
- Create: `infoblock.go`
- Test: `infoblock_test.go`

**Interfaces:**
- Consumes: `getLocked`, `sendLocked` (Task 6), `reportInfo1`, `reportInfo2`, `infoBlockSize`, `infoReportSize` (Task 5)
- Produces: `(*Device).InfoBlock(n int) ([]byte, error)`, `(*Device).SetInfoBlock(n int, data []byte) error`

- [ ] **Step 1: Write the failing tests**

`infoblock_test.go`:

```go
package blinkstick

import (
	"bytes"
	"errors"
	"testing"
)

func TestInfoBlockRoundTrip(t *testing.T) {
	d, _ := openFake(Square)
	if err := d.SetInfoBlock(1, []byte("office")); err != nil {
		t.Fatalf("SetInfoBlock: %v", err)
	}
	if err := d.SetInfoBlock(2, []byte("desk")); err != nil {
		t.Fatalf("SetInfoBlock: %v", err)
	}
	for n, want := range map[int]string{1: "office", 2: "desk"} {
		got, err := d.InfoBlock(n)
		if err != nil {
			t.Fatalf("InfoBlock(%d): %v", n, err)
		}
		if string(got) != want {
			t.Errorf("InfoBlock(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestSetInfoBlockReport(t *testing.T) {
	d, ft := openFake(Nano)
	d.SetInfoBlock(2, []byte("hi"))
	sent := ft.sends[0]
	want := make([]byte, 33)
	want[0], want[1], want[2] = 3, 'h', 'i'
	if !bytes.Equal(sent, want) {
		t.Errorf("sent % x, want % x", sent, want)
	}
}

func TestInfoBlockEmpty(t *testing.T) {
	d, _ := openFake(Nano)
	got, err := d.InfoBlock(1)
	if err != nil {
		t.Fatalf("InfoBlock: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("InfoBlock = %q, want empty", got)
	}
}

func TestSetInfoBlockFull(t *testing.T) {
	d, _ := openFake(Nano)
	data := bytes.Repeat([]byte("x"), 32)
	if err := d.SetInfoBlock(1, data); err != nil {
		t.Fatalf("SetInfoBlock(32 bytes): %v", err)
	}
	if got, _ := d.InfoBlock(1); !bytes.Equal(got, data) {
		t.Errorf("InfoBlock = %q, want %q", got, data)
	}
}

func TestInfoBlockOutOfRange(t *testing.T) {
	d, ft := openFake(Nano)
	if err := d.SetInfoBlock(1, make([]byte, 33)); !errors.Is(err, ErrOutOfRange) {
		t.Errorf("SetInfoBlock(33 bytes) = %v, want ErrOutOfRange", err)
	}
	for _, n := range []int{0, 3} {
		if _, err := d.InfoBlock(n); !errors.Is(err, ErrOutOfRange) {
			t.Errorf("InfoBlock(%d) = %v, want ErrOutOfRange", n, err)
		}
		if err := d.SetInfoBlock(n, nil); !errors.Is(err, ErrOutOfRange) {
			t.Errorf("SetInfoBlock(%d) = %v, want ErrOutOfRange", n, err)
		}
	}
	if ft.sendCount() != 0 {
		t.Errorf("sends = %d, want 0", ft.sendCount())
	}
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test .`
Expected: FAIL to compile with `d.SetInfoBlock undefined`.

- [ ] **Step 3: Write `infoblock.go`**

```go
package blinkstick

import (
	"bytes"
	"fmt"
)

// InfoBlock reads user data block n (1 or 2) from the device's EEPROM, with
// trailing NUL bytes removed.
func (d *Device) InfoBlock(n int) ([]byte, error) {
	id, err := infoReport(n)
	if err != nil {
		return nil, err
	}
	buf := make([]byte, infoReportSize)
	buf[0] = id
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.getLocked(buf); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf[1:], "\x00"), nil
}

// SetInfoBlock writes up to 32 bytes to user data block n (1 or 2), padded
// with zeros. It writes EEPROM, which wears with use, so do not call it in a
// loop.
func (d *Device) SetInfoBlock(n int, data []byte) error {
	id, err := infoReport(n)
	if err != nil {
		return err
	}
	if len(data) > infoBlockSize {
		return fmt.Errorf("%w: info block data is %d bytes, max %d", ErrOutOfRange, len(data), infoBlockSize)
	}
	buf := make([]byte, infoReportSize)
	buf[0] = id
	copy(buf[1:], data)
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.sendLocked(buf)
}

func infoReport(n int) (byte, error) {
	switch n {
	case 1:
		return reportInfo1, nil
	case 2:
		return reportInfo2, nil
	}
	return 0, fmt.Errorf("%w: info block %d, want 1 or 2", ErrOutOfRange, n)
}
```

- [ ] **Step 4: Run the tests to see them pass**

Run: `go test -race -v .`
Expected: PASS.

- [ ] **Step 5: Commit (after the user approves)**

```bash
git add infoblock.go infoblock_test.go
git commit -S -m "Added info block read and write"
```

---

### Task 10: Effects

**Files:**
- Create: `effects.go`
- Test: `effects_test.go`

**Interfaces:**
- Consumes: `SetAll`, `Off`, `Frame`, `writeFrameLocked`, `mu`, `limit` (Tasks 6 and 7), `scale` (Task 3)
- Produces:
  - `(*Device).Blink(ctx context.Context, c RGB, period time.Duration, repeats int) error`
  - `(*Device).Pulse(ctx context.Context, c RGB, duration time.Duration, repeats int) error`
  - `(*Device).Morph(ctx context.Context, to RGB, duration time.Duration) error`
  - unexported `const effectStep = 20 * time.Millisecond`, `var sleep func(context.Context, time.Duration) error`

- [ ] **Step 1: Write the failing tests**

`effects_test.go`:

```go
package blinkstick

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

// stubSleep records sleeps without waiting.
func stubSleep(t *testing.T) *[]time.Duration {
	orig := sleep
	var slept []time.Duration
	sleep = func(ctx context.Context, d time.Duration) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		slept = append(slept, d)
		return nil
	}
	t.Cleanup(func() { sleep = orig })
	return &slept
}

func TestBlink(t *testing.T) {
	slept := stubSleep(t)
	d, ft := openFake(Square)
	if err := d.Blink(context.Background(), White, 200*time.Millisecond, 2); err != nil {
		t.Fatalf("Blink: %v", err)
	}
	want := [][]RGB{fill(8, White), fill(8, Off), fill(8, White), fill(8, Off)}
	if got := ft.sentFrames(8); !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("frames = %v, want %v", got, want)
	}
	if want := slices.Repeat([]time.Duration{100 * time.Millisecond}, 4); !slices.Equal(*slept, want) {
		t.Errorf("slept = %v, want %v", *slept, want)
	}
}

func TestMorph(t *testing.T) {
	slept := stubSleep(t)
	d, ft := openFake(Nano)
	to := RGB{R: 100, G: 200}
	if err := d.Morph(context.Background(), to, 100*time.Millisecond); err != nil {
		t.Fatalf("Morph: %v", err)
	}
	frames := ft.sentFrames(2)
	if len(frames) != 5 {
		t.Fatalf("frames = %d, want 5 (100ms in 20ms steps)", len(frames))
	}
	if got := frames[0][0]; got != (RGB{R: 20, G: 40}) {
		t.Errorf("first step = %v, want {20 40 0}", got)
	}
	if got := frames[4]; !slices.Equal(got, fill(2, to)) {
		t.Errorf("last frame = %v, want all %v", got, to)
	}
	if len(*slept) != 4 {
		t.Errorf("sleeps = %d, want 4", len(*slept))
	}
}

func TestMorphStartsFromCurrentFrame(t *testing.T) {
	stubSleep(t)
	d, ft := openFake(Nano)
	d.SetFrame([]RGB{{R: 200}, {B: 100}})
	if err := d.Morph(context.Background(), Off, 40*time.Millisecond); err != nil {
		t.Fatalf("Morph: %v", err)
	}
	frames := ft.sentFrames(2)
	if got, want := frames[1], []RGB{{R: 100}, {B: 50}}; !slices.Equal(got, want) {
		t.Errorf("midpoint = %v, want %v", got, want)
	}
	if got := ft.frame(2); !slices.Equal(got, fill(2, Off)) {
		t.Errorf("end = %v, want off", got)
	}
}

func TestMorphZeroDurationJumps(t *testing.T) {
	stubSleep(t)
	d, ft := openFake(Nano)
	if err := d.Morph(context.Background(), White, 0); err != nil {
		t.Fatalf("Morph: %v", err)
	}
	if got := ft.sentFrames(2); len(got) != 1 || !slices.Equal(got[0], fill(2, White)) {
		t.Errorf("frames = %v, want one white frame", got)
	}
}

func TestMorphHonoursBrightnessLimit(t *testing.T) {
	stubSleep(t)
	d, ft := openFake(Nano)
	d.SetBrightnessLimit(128)
	d.Morph(context.Background(), White, 0)
	if got := ft.frame(2); !slices.Equal(got, fill(2, RGB{128, 128, 128})) {
		t.Errorf("frame = %v, want all 128", got)
	}
}

func TestPulse(t *testing.T) {
	stubSleep(t)
	d, ft := openFake(Nano)
	c := RGB{G: 200}
	if err := d.Pulse(context.Background(), c, 80*time.Millisecond, 1); err != nil {
		t.Fatalf("Pulse: %v", err)
	}
	// Off, two steps up, two steps down.
	frames := ft.sentFrames(2)
	if len(frames) != 5 {
		t.Fatalf("frames = %d, want 5", len(frames))
	}
	if !slices.Equal(frames[0], fill(2, Off)) {
		t.Errorf("first = %v, want off", frames[0])
	}
	if !slices.Equal(frames[2], fill(2, c)) {
		t.Errorf("peak = %v, want %v", frames[2], c)
	}
	if !slices.Equal(frames[4], fill(2, Off)) {
		t.Errorf("last = %v, want off", frames[4])
	}
}

func TestEffectsCancelled(t *testing.T) {
	stubSleep(t)
	d, ft := openFake(Nano)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := d.Blink(ctx, White, time.Second, 1); !errors.Is(err, context.Canceled) {
		t.Errorf("Blink = %v, want context.Canceled", err)
	}
	if err := d.Pulse(ctx, White, time.Second, 1); !errors.Is(err, context.Canceled) {
		t.Errorf("Pulse = %v, want context.Canceled", err)
	}
	if err := d.Morph(ctx, White, time.Second); !errors.Is(err, context.Canceled) {
		t.Errorf("Morph = %v, want context.Canceled", err)
	}
	if ft.sendCount() != 0 {
		t.Errorf("sends = %d, want 0 for an already cancelled context", ft.sendCount())
	}
}

func TestEffectsRejectBadArguments(t *testing.T) {
	stubSleep(t)
	d, _ := openFake(Nano)
	ctx := context.Background()
	checks := map[string]error{
		"Blink repeats 0": d.Blink(ctx, White, time.Second, 0),
		"Blink period 0":  d.Blink(ctx, White, 0, 1),
		"Pulse repeats 0": d.Pulse(ctx, White, time.Second, 0),
		"Pulse duration 0": d.Pulse(ctx, White, 0, 1),
		"Morph negative":  d.Morph(ctx, White, -time.Second),
	}
	for name, err := range checks {
		if !errors.Is(err, ErrOutOfRange) {
			t.Errorf("%s = %v, want ErrOutOfRange", name, err)
		}
	}
}

func TestSleepHonoursCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := sleep(ctx, time.Hour); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("sleep = %v, want DeadlineExceeded", err)
	}
	if time.Since(start) > time.Second {
		t.Error("sleep ignored cancellation")
	}
}
```

- [ ] **Step 2: Run them to see them fail**

Run: `go test .`
Expected: FAIL to compile with `undefined: sleep`.

- [ ] **Step 3: Write `effects.go`**

```go
package blinkstick

import (
	"context"
	"fmt"
	"time"
)

// effectStep is the time between frames in an effect.
const effectStep = 20 * time.Millisecond

// sleep waits for d, or until ctx is done. Tests replace it.
var sleep = func(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Blink turns every LED to c for period/2, then off for period/2, repeats
// times. It ends with the LEDs off. If ctx is cancelled the LEDs are left as
// they are.
func (d *Device) Blink(ctx context.Context, c RGB, period time.Duration, repeats int) error {
	if err := checkEffect(period, repeats); err != nil {
		return err
	}
	for range repeats {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := d.SetAll(c); err != nil {
			return err
		}
		if err := sleep(ctx, period/2); err != nil {
			return err
		}
		if err := d.Off(); err != nil {
			return err
		}
		if err := sleep(ctx, period/2); err != nil {
			return err
		}
	}
	return nil
}

// Pulse fades every LED from off up to c over duration/2 and back down over
// duration/2, repeats times. It ends with the LEDs off. If ctx is cancelled
// the LEDs are left as they are.
func (d *Device) Pulse(ctx context.Context, c RGB, duration time.Duration, repeats int) error {
	if err := checkEffect(duration, repeats); err != nil {
		return err
	}
	for range repeats {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := d.Off(); err != nil {
			return err
		}
		if err := d.Morph(ctx, c, duration/2); err != nil {
			return err
		}
		if err := d.Morph(ctx, Off, duration/2); err != nil {
			return err
		}
	}
	return nil
}

// Morph fades every LED from its current value to `to` over duration, in
// 20 ms steps. A duration under one step jumps straight to `to`. It ends with
// every LED at `to`. If ctx is cancelled the LEDs are left as they are.
func (d *Device) Morph(ctx context.Context, to RGB, duration time.Duration) error {
	if duration < 0 {
		return fmt.Errorf("%w: duration %v", ErrOutOfRange, duration)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	from, err := d.Frame()
	if err != nil {
		return err
	}
	d.mu.Lock()
	target := to.scale(d.limit)
	d.mu.Unlock()

	steps := max(1, int(duration/effectStep))
	leds := make([]RGB, len(from))
	for s := 1; s <= steps; s++ {
		for i, f := range from {
			leds[i] = lerp(f, target, s, steps)
		}
		d.mu.Lock()
		err := d.writeFrameLocked(leds)
		d.mu.Unlock()
		if err != nil {
			return err
		}
		if s < steps {
			if err := sleep(ctx, effectStep); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkEffect(d time.Duration, repeats int) error {
	if d <= 0 || repeats < 1 {
		return fmt.Errorf("%w: duration %v, repeats %d", ErrOutOfRange, d, repeats)
	}
	return nil
}

// lerp returns the point s/n of the way from a to b.
func lerp(a, b RGB, s, n int) RGB {
	f := func(x, y uint8) uint8 { return uint8(int(x) + (int(y)-int(x))*s/n) }
	return RGB{R: f(a.R, b.R), G: f(a.G, b.G), B: f(a.B, b.B)}
}
```

- [ ] **Step 4: Run the tests to see them pass**

Run: `go test -race -count=3 -v .`
Expected: PASS. Check the Morph numbers by hand if a test fails: 100 ms / 20 ms is 5 steps, and step 1 of 5 from 0 to 100 is 20.

- [ ] **Step 5: Commit (after the user approves)**

```bash
git add effects.go effects_test.go
git commit -S -m "Added blink, pulse and morph effects"
```

---

### Task 11: Package docs and runnable examples

**Files:**
- Modify: `doc.go` (replace)
- Create: `example_test.go`

**Interfaces:**
- Consumes: the whole public API
- Produces: pkg.go.dev landing text and examples

- [ ] **Step 1: Replace `doc.go`**

```go
// Package blinkstick drives BlinkStick USB LED devices. The Nano and Square
// are supported on macOS.
//
// Open a stick, set its LEDs and close it:
//
//	d, err := blinkstick.Open()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer d.Close()
//	err = d.SetAll(blinkstick.RGB{R: 255})
//
// Use [List] to find every attached stick and [OpenSerial] to open a
// particular one. Several sticks can be open at once, and a [Device] is safe
// for concurrent use.
//
// # Requirements
//
// The package uses cgo and a bundled copy of hidapi. On macOS, install the
// Xcode command line tools. macOS lets only one process open a stick at a
// time.
//
// # EEPROM
//
// [Device.SetInfoBlock] writes to EEPROM on the device, which wears with use.
// Setting LEDs does not.
//
// This package is unofficial and not affiliated with Agile Innovative.
package blinkstick
```

- [ ] **Step 2: Write `example_test.go`**

Only `ExampleParseRGB` has an `// Output:` line, because it is the only example that does not need hardware. The others are compiled by `go test` but never run.

```go
package blinkstick_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/bawdo/go-blinkstick"
)

func Example() {
	d, err := blinkstick.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	if err := d.SetAll(blinkstick.RGB{R: 255}); err != nil {
		log.Fatal(err)
	}
}

func ExampleList() {
	infos, err := blinkstick.List()
	if err != nil {
		log.Fatal(err)
	}
	for _, info := range infos {
		fmt.Println(info.Serial, info.Model.Name, info.Model.LEDs)
	}
}

func ExampleOpenSerial() {
	d, err := blinkstick.OpenSerial("BS072777-3.0")
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()
	fmt.Println(d.Info().Model.Name)
}

func ExampleParseRGB() {
	c, err := blinkstick.ParseRGB("#ff8800")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(c)
	// Output: {255 136 0}
}

func ExampleDevice_SetLED() {
	d, err := blinkstick.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	// Top LED red, bottom LED blue on a Nano.
	d.SetLED(0, blinkstick.RGB{R: 255})
	d.SetLED(1, blinkstick.RGB{B: 255})
}

func ExampleDevice_Pulse() {
	d, err := blinkstick.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := d.Pulse(ctx, blinkstick.RGB{G: 255}, time.Second, 3); err != nil {
		log.Fatal(err)
	}
}

func ExampleDevice_SetBrightnessLimit() {
	d, err := blinkstick.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	d.SetBrightnessLimit(64) // about a quarter of full brightness
	d.SetAll(blinkstick.White)
}
```

- [ ] **Step 3: Run the tests and check the docs render**

```bash
make test
go doc -all . | head -60
```

Expected: `make test` passes and `ExampleParseRGB` runs. `go doc` shows the package overview, then the exported identifiers with no stray internal names.

- [ ] **Step 4: Commit (after the user approves)**

```bash
git add doc.go example_test.go
git commit -S -m "Added package docs and runnable examples"
```

---

### Task 12: Hardware tests

**Files:**
- Create: `hardware_test.go`
- Record the report 10 probe result in the task report. Task 13 puts it in `CAPABILITIES.md`.

**Interfaces:**
- Consumes: the whole public API, plus unexported `getLocked` and `mu` for the report 10 probe
- Produces: `make test-hardware` coverage of every v0.1.0 feature on both sticks

- [ ] **Step 1: Write `hardware_test.go`**

```go
//go:build hardware

// Hardware tests need a BlinkStick Nano and a Square attached, and someone
// watching the LEDs. Run them with: make test-hardware
//
// EEPROM RULE: these tests must never call SetInfoBlock or anything else that
// writes EEPROM, because EEPROM wears with use. LED values live in RAM and
// cause no wear.

package blinkstick

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// watch gives a person time to see each step.
const watch = 700 * time.Millisecond

// openBoth opens the attached Nano and Square and turns them off and closes
// them when the test ends.
func openBoth(t *testing.T) (nano, square *Device) {
	t.Helper()
	infos, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, info := range infos {
		d, err := OpenSerial(info.Serial)
		if err != nil {
			t.Fatalf("OpenSerial(%s): %v", info.Serial, err)
		}
		t.Cleanup(func() {
			d.Off()
			d.Close()
		})
		switch info.Model {
		case Nano:
			nano = d
		case Square:
			square = d
		}
	}
	if nano == nil || square == nil {
		t.Fatalf("need a Nano and a Square attached, found %+v", infos)
	}
	return nano, square
}

func eachStick(t *testing.T, f func(t *testing.T, d *Device)) {
	nano, square := openBoth(t)
	for _, d := range []*Device{nano, square} {
		t.Run(d.Info().Model.Name, func(t *testing.T) { f(t, d) })
	}
}

func TestHardwareIdentity(t *testing.T) {
	eachStick(t, func(t *testing.T, d *Device) {
		info := d.Info()
		t.Logf("%+v", info)
		if !strings.HasPrefix(info.Serial, "BS") || info.Version == "" {
			t.Errorf("serial %q, version %q", info.Serial, info.Version)
		}
		if info.Manufacturer != "Agile Innovative Ltd" {
			t.Errorf("manufacturer %q", info.Manufacturer)
		}
	})
}

func TestHardwareFrameRoundTrip(t *testing.T) {
	eachStick(t, func(t *testing.T, d *Device) {
		leds := make([]RGB, d.Info().Model.LEDs)
		for i := range leds {
			leds[i] = RGB{R: uint8(40 + i*25), G: uint8(200 - i*20), B: uint8(i * 30)}
		}
		if err := d.SetFrame(leds); err != nil {
			t.Fatalf("SetFrame: %v", err)
		}
		time.Sleep(watch)
		got, err := d.Frame()
		if err != nil {
			t.Fatalf("Frame: %v", err)
		}
		if !slices.Equal(got, leds) {
			t.Errorf("Frame = %v, want %v", got, leds)
		}
	})
}

func TestHardwareSetLEDAndLED(t *testing.T) {
	eachStick(t, func(t *testing.T, d *Device) {
		last := d.Info().Model.LEDs - 1
		red := RGB{R: 255}
		d.Off()
		if err := d.SetLED(last, red); err != nil {
			t.Fatalf("SetLED: %v", err)
		}
		time.Sleep(watch)
		got, err := d.LED(last)
		if err != nil || got != red {
			t.Errorf("LED(%d) = %v, %v; want %v", last, got, err, red)
		}
		if first, _ := d.LED(0); first != Off {
			t.Errorf("LED(0) = %v, want off", first)
		}
	})
}

func TestHardwareBrightnessLimit(t *testing.T) {
	eachStick(t, func(t *testing.T, d *Device) {
		d.SetBrightnessLimit(64)
		defer d.SetBrightnessLimit(255)
		if err := d.SetAll(White); err != nil {
			t.Fatalf("SetAll: %v", err)
		}
		time.Sleep(watch)
		got, _ := d.Frame()
		if want := fill(d.Info().Model.LEDs, RGB{64, 64, 64}); !slices.Equal(got, want) {
			t.Errorf("Frame = %v, want %v", got, want)
		}
	})
}

func TestHardwareEffects(t *testing.T) {
	eachStick(t, func(t *testing.T, d *Device) {
		ctx := context.Background()
		t.Log("watch: two white blinks, one green pulse, then a fade to blue")
		if err := d.Blink(ctx, White, 400*time.Millisecond, 2); err != nil {
			t.Fatalf("Blink: %v", err)
		}
		if err := d.Pulse(ctx, RGB{G: 255}, time.Second, 1); err != nil {
			t.Fatalf("Pulse: %v", err)
		}
		if err := d.Morph(ctx, RGB{B: 255}, time.Second); err != nil {
			t.Fatalf("Morph: %v", err)
		}
		got, _ := d.Frame()
		if want := fill(d.Info().Model.LEDs, RGB{B: 255}); !slices.Equal(got, want) {
			t.Errorf("after Morph Frame = %v, want %v", got, want)
		}
		time.Sleep(watch)
	})
}

func TestHardwareInfoBlocksReadOnly(t *testing.T) {
	eachStick(t, func(t *testing.T, d *Device) {
		for n := 1; n <= 2; n++ {
			b, err := d.InfoBlock(n)
			if err != nil {
				t.Fatalf("InfoBlock(%d): %v", n, err)
			}
			t.Logf("info block %d: %q", n, b)
		}
	})
}

func TestHardwareMultiDevice(t *testing.T) {
	nano, square := openBoth(t)
	var wg sync.WaitGroup
	for _, d := range []*Device{nano, square} {
		wg.Go(func() {
			for i := range 20 {
				c := RGB{R: 255}
				if i%2 == 1 {
					c = RGB{B: 255}
				}
				if err := d.SetAll(c); err != nil {
					t.Errorf("%s SetAll: %v", d.Info().Model.Name, err)
					return
				}
				time.Sleep(50 * time.Millisecond)
			}
		})
	}
	wg.Wait()

	if err := nano.Close(); err != nil {
		t.Fatalf("close Nano: %v", err)
	}
	green := RGB{G: 255}
	if err := square.SetAll(green); err != nil {
		t.Fatalf("Square after closing Nano: %v", err)
	}
	time.Sleep(watch)
	if got, _ := square.Frame(); !slices.Equal(got, fill(8, green)) {
		t.Errorf("Square Frame = %v, want all green", got)
	}
}

// TestHardwareReport10Probe reads the Square's undocumented report 10. It
// never writes to it.
func TestHardwareReport10Probe(t *testing.T) {
	_, square := openBoth(t)
	buf := make([]byte, 3)
	buf[0] = 10
	square.mu.Lock()
	err := square.getLocked(buf)
	square.mu.Unlock()
	t.Logf("report 10: err=%v bytes=% x", err, buf)
}
```

- [ ] **Step 2: Check it compiles without hardware**

```bash
go vet -tags hardware ./...
make test
```

Expected: vet is clean with the tag. `make test` still passes, and does not build the hardware file without the tag.

- [ ] **Step 3: Confirm no EEPROM writes**

```bash
grep -n 'SetInfoBlock' hardware_test.go
```

Expected: only the comment line matches.

- [ ] **Step 4: Run on real hardware, three times in a row**

Ask the user to attach the Nano, the Square and the Focusrite, and to watch the LEDs. Then run:

```bash
for i in 1 2 3; do make test-hardware || break; done
```

Expected: three full passes. Ask the user to confirm that what they saw matched the logged descriptions (blinks, pulse, fade to blue, alternating red and blue on both sticks, then the Square green). Record the report 10 log line for Task 13.

- [ ] **Step 5: Commit (after the user approves)**

```bash
git add hardware_test.go
git commit -S -m "Added hardware tests for Nano and Square"
```

---

### Task 13: README, CAPABILITIES and CHANGELOG

**Files:**
- Modify: `README.md` (replace the interim one)
- Create: `CAPABILITIES.md`, `CHANGELOG.md`

**Interfaces:**
- Consumes: the report 10 result from Task 12
- Produces: user documentation

- [ ] **Step 1: Replace `README.md`**

````markdown
# go-blinkstick

[![Go Reference](https://pkg.go.dev/badge/github.com/bawdo/go-blinkstick.svg)](https://pkg.go.dev/github.com/bawdo/go-blinkstick)
[![Go Report Card](https://goreportcard.com/badge/github.com/bawdo/go-blinkstick)](https://goreportcard.com/report/github.com/bawdo/go-blinkstick)

A Go package for driving BlinkStick USB LED devices.

This package is unofficial. It is not made by, or affiliated with, Agile Innovative, the makers
of BlinkStick.

## Supported devices

| Device | macOS | Linux |
|---|---|---|
| BlinkStick Nano | yes | not yet |
| BlinkStick Square | yes | not yet |

See [CAPABILITIES.md](CAPABILITIES.md) for every feature these devices have and what this
package supports so far.

## Requirements

- Go 1.26 or later.
- cgo. On macOS, install the Xcode command line tools with `xcode-select --install`. Builds
  with `CGO_ENABLED=0` will not work.
- Nothing else. hidapi is bundled.

## Install

```sh
go get github.com/bawdo/go-blinkstick
```

## Quick start

```go
package main

import (
	"context"
	"log"
	"time"

	"github.com/bawdo/go-blinkstick"
)

func main() {
	d, err := blinkstick.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer d.Close()

	red, _ := blinkstick.ParseRGB("#ff0000")
	if err := d.SetAll(red); err != nil {
		log.Fatal(err)
	}

	// Pulse green twice, one second each.
	err = d.Pulse(context.Background(), blinkstick.RGB{G: 255}, time.Second, 2)
	if err != nil {
		log.Fatal(err)
	}
}
```

## More than one stick

`List` returns every attached stick. Open a particular one by its serial:

```go
infos, err := blinkstick.List()
if err != nil {
	log.Fatal(err)
}
for _, info := range infos {
	fmt.Println(info.Serial, info.Model.Name)
}

d, err := blinkstick.OpenSerial("BS072777-3.0")
```

Each open `Device` is independent, and its methods are safe to call from several goroutines.
macOS lets only one process open a stick at a time.

## Good to know

- `Blink`, `Pulse` and `Morph` block until they finish. Cancel them with the context.
- `SetBrightnessLimit` caps how bright the LEDs get. A Square at full white draws about 500 mA.
- `SetInfoBlock` writes to EEPROM on the device, which wears out with heavy use. Do not call
  it in a loop.

## Development

```sh
make help           # list targets
make test           # vet and unit tests, no hardware needed
make build          # compile
make test-hardware  # tests against a real Nano and Square
```

## Versioning

Releases follow [semantic versioning](https://semver.org/) and are tagged `vX.Y.Z`. Until
`v1.0.0` the API may change between minor releases. See [CHANGELOG.md](CHANGELOG.md).

## Licence

MIT. See [LICENSE](LICENSE).
````

- [ ] **Step 2: Write `CAPABILITIES.md`**

Before writing, set the Report 10 row's Notes from the Task 12 probe. If the probe returned data, write the bytes seen, for example `Square only, 2 bytes, read returns 00 00, purpose unknown`. If it errored, write `Square only, 2 bytes, read fails, purpose unknown`.

```markdown
# Capabilities

What the BlinkStick Nano and Square can do, and what this package supports so far. Back to the
[README](README.md).

This file is updated whenever a feature lands.

Key: **yes** supported, **no** not yet, **n/a** the device does not have it, **unverified** the
device may have it but it has not been confirmed on hardware.

## Device features

Features built into the BlinkStick firmware.

| Feature | Nano | Square | Supported since | Notes |
|---|---|---|---|---|
| List attached sticks | yes | yes | v0.1.0 | `List` |
| Open the first stick or one by serial | yes | yes | v0.1.0 | `Open`, `OpenSerial` |
| Several sticks open at once | yes | yes | v0.1.0 | |
| Serial and firmware version | yes | yes | v0.1.0 | `Info` |
| Manufacturer and product strings | yes | yes | v0.1.0 | `Info` |
| Set all LEDs | yes | yes | v0.1.0 | `SetAll`, `Off` |
| Set one LED | yes | yes | v0.1.0 | `SetLED` |
| Set every LED at once | yes | yes | v0.1.0 | `SetFrame` |
| Read LED values back | yes | yes | v0.1.0 | `Frame`, `LED` |
| Info blocks 1 and 2 | yes | yes | v0.1.0 | `InfoBlock`, `SetInfoBlock`. 32 bytes each, stored in EEPROM |
| Set one LED directly (report 5) | unverified | unverified | no | Needs firmware mode 2 |
| Device mode: normal, inverse, WS2812 (report 4) | unverified | unverified | no | Stored in EEPROM |
| Report 10 | n/a | unknown | no | Square only, 2 bytes, purpose unknown |
| LED count setting (report 0x81) | n/a | n/a | n/a | BlinkStick Flex only |
| 16, 32 and 64 LED frames (reports 7 to 9) | n/a | n/a | n/a | For longer strips |

## Host features

Features this package adds in software. They work the same on every supported device.

| Feature | Supported since | Notes |
|---|---|---|
| Model detection | v0.1.0 | From the serial and USB release number |
| Colour from hex or `r,g,b` | v0.1.0 | `ParseRGB` |
| Brightness limit | v0.1.0 | `SetBrightnessLimit` |
| Blink, pulse and morph | v0.1.0 | Cancellable with a context |
| CSS colour names | no | |
| Random colour | no | |
| Inverse colours | no | |
| Reconnect after unplug | no | |

## Platforms

| Platform | Supported since |
|---|---|
| macOS | v0.1.0 |
| Linux | no |
| Windows | no |
```

- [ ] **Step 3: Write `CHANGELOG.md`**

At release time the maintainer renames `Unreleased` to `[0.1.0] - <release date>`.

```markdown
# Changelog

All notable changes to this project are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[semantic versioning](https://semver.org/).

## [Unreleased]

### Added

- Support for the BlinkStick Nano and Square on macOS.
- `List`, `Open` and `OpenSerial`, with several sticks open at once.
- Model detection from the serial and USB release number.
- `SetAll`, `SetLED`, `SetFrame`, `Off`, `Frame` and `LED`.
- Info blocks 1 and 2: `InfoBlock` and `SetInfoBlock`.
- `ParseRGB` for hex and `r,g,b` values.
- `SetBrightnessLimit`.
- `Blink`, `Pulse` and `Morph` effects, cancellable with a context.
```

- [ ] **Step 4: Check the house rules**

```bash
wc -w README.md
grep -nP '[\x{2013}\x{2014}]' README.md CAPABILITIES.md CHANGELOG.md doc.go *.go
grep -nP '[\x{1F300}-\x{1FAFF}\x{2600}-\x{27BF}]' README.md CAPABILITIES.md CHANGELOG.md
grep -n 'CAPABILITIES.md' README.md
grep -n 'README.md' CAPABILITIES.md
```

Expected: the word count is under 600. The dash and emoji greps print nothing. Each link grep prints at least one line.

- [ ] **Step 5: Commit (after the user approves)**

```bash
git add README.md CAPABILITIES.md CHANGELOG.md
git commit -S -m "Added README, capabilities table and changelog"
```

---

### Task 14: CI and final verification

**Files:**
- Create: `.github/workflows/ci.yml`

**Interfaces:**
- Consumes: everything
- Produces: CI on every push and pull request

- [ ] **Step 1: Check the current action versions**

```bash
gh api repos/actions/checkout/releases/latest --jq .tag_name
gh api repos/actions/setup-go/releases/latest --jq .tag_name
```

Use each result's major version (for example `v5`) in place of the versions below if they differ.

- [ ] **Step 2: Write `.github/workflows/ci.yml`**

```yaml
name: CI

on:
  push:
  pull_request:

permissions:
  contents: read

jobs:
  test:
    runs-on: macos-latest
    strategy:
      matrix:
        go: ["1.26", "1.27"]
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v6
        with:
          go-version: ${{ matrix.go }}
      - run: go vet ./...
      - run: go test -race ./...
      - run: go run honnef.co/go/tools/cmd/staticcheck@latest ./...
      - run: go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

- [ ] **Step 3: Run the same checks locally**

```bash
make test
go run honnef.co/go/tools/cmd/staticcheck@latest ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

Expected: all clean. Fix any staticcheck finding in the file it names, then rerun `make test`.

- [ ] **Step 4: Final checks on the tree**

```bash
for i in 1 2 3; do go test -race -count=1 ./... || break; done
git status --short
git log --format='%h %G? %s' main..HEAD
```

Expected: three passes. Only `ci.yml` is uncommitted. Every commit on the branch shows `G`.

- [ ] **Step 5: Commit (after the user approves)**

```bash
git add .github/workflows/ci.yml
git commit -S -m "Added GitHub Actions CI for macOS"
```

- [ ] **Step 6: Hand the release over to the user**

Do not run these. Give them to the user once the branch is merged to `main`:

```bash
git push origin feature/go-package
# after merging to main, and after renaming Unreleased to [0.1.0] - <date> in CHANGELOG.md:
git tag -s v0.1.0 -m "v0.1.0"
git push origin v0.1.0
GOPROXY=proxy.golang.org go list -m github.com/bawdo/go-blinkstick@v0.1.0
```

The last command asks the Go module proxy to fetch the release so pkg.go.dev indexes it.

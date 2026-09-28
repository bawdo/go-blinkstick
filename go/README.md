# BlinkStick Go

Minimal Go driver and CLI for BlinkStick Nano and Square.

## Requirements

- Go 1.27+
- cgo toolchain: Xcode command line tools on macOS (`xcode-select --install`)
- hidapi is bundled by `github.com/sstallion/go-hid`, no Homebrew needed

## Build

    cd go
    go build -o bin/blink ./cmd/blink

## Use

    ./bin/blink on    # all LEDs white
    ./bin/blink off   # all LEDs off

Exit codes: 0 success, 1 device error, 2 usage error.

With more than one BlinkStick plugged in, the first one found is used.

## Test

    go test ./...

Hardware is not needed for the tests; the HID device is faked.

## Protocol notes

- USB 20a0:41e5, HID feature reports on vendor usage page 0xFF00.
- Report 6: `[0x06, channel, 8 x (G, R, B)]`. The Nano only uses the first 2 LEDs.
- GRB order and per-LED addressing verified on hardware: Nano LED 0 is the top LED, LED 1 the bottom; all 8 Square LEDs addressable.

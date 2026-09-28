# Changelog

All notable changes to this project are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[semantic versioning](https://semver.org/).

## [0.3.0] - 2026-09-28

### Added

- `ColourNames` lists every CSS colour name `ParseRGB` accepts, sorted, for shell completion.
- `RGB.Hex` formats a colour as lower case `#rrggbb`.

## [0.2.0] - 2026-09-28

### Added

- `ParseRGB` accepts the 148 CSS colour names, in any case.
- `ParseRGB` accepts hex without the leading `#`, for example `ff8800` or `f80`.
- `RandomRGB` for any colour and `RandomVivid` for a bright, fully saturated one.
- `RGB.Inverse`, and `SetInverse` to flip every colour written to a stick.
- Stick names, stored in info block 1: `SetName`, `Name`, `OpenName` and `ListNamed`.
- `ErrInvalidName` and `ErrDuplicateName`.
- Reconnect after unplug: a `Device` reopens its stick by serial and repaints the last frame.
  `ErrDisconnected` when the stick is still missing.

## [0.1.0] - 2026-09-28

### Added

- Support for the BlinkStick Nano and Square on macOS.
- `List`, `Open` and `OpenSerial`, with several sticks open at once.
- Model detection from the serial and USB release number.
- `SetAll`, `SetLED`, `SetFrame`, `Off`, `Frame` and `LED`.
- Info blocks 1 and 2: `InfoBlock` and `SetInfoBlock`.
- `ParseRGB` for hex and `r,g,b` values.
- `SetBrightnessLimit`.
- `Blink`, `Pulse` and `Morph` effects, cancellable with a context.
- Safe to list and open sticks from several goroutines.

[Unreleased]: https://github.com/bawdo/go-blinkstick/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/bawdo/go-blinkstick/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/bawdo/go-blinkstick/releases/tag/v0.1.0

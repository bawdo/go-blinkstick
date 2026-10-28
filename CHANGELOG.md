# Changelog

All notable changes to this project are recorded here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project uses
[semantic versioning](https://semver.org/).

## [Unreleased]

## [0.6.1] - 2026-10-29

### Fixed

- `SetMode` now waits until the stick answers again before it returns. A stick refuses every
  transfer for about 12 to 15 ms after a mode write, and the retries are too quick to outlast that,
  so the next call (reading the mode back, writing an info block, a name or LEDs) failed as if the
  stick had been unplugged. `SetMode` reads report 4 every 5 ms until it shows the new mode, and
  returns `ErrDisconnected` naming the stick if it has not answered after 250 ms. The old doc
  comment said the write blocks for about 50 ms; it never waited at all.

## [0.6.0] - 2026-10-26

### Changed

- **Linux udev rule:** `udev/60-blinkstick.rules` now grants only the `hidraw` node, not the raw
  USB device. This package needs nothing more, so it keeps working. Tools that use libusb directly,
  such as the official Python `blinkstick` package, lose access until you add the line given under
  "Linux" in the README. An installed copy of the old rule is not changed until you replace it.
- `ListNamed` lists a stick whose name cannot be read with `Busy` set, instead of failing the whole
  listing.
- When `OpenName` finds no match, its error now says why each skipped stick was skipped, including
  sticks that could not be opened. It still matches `ErrNotFound`.
- Errors from opening a stick now wrap hidapi's reason as well as `ErrNotFound`, so `errors.Is`
  and `errors.As` can reach it.

### Fixed

- A feature report read that returns fewer bytes than asked for is now retried and then reported
  as an error. Before, the missing bytes read as zero, so `Frame`, `LED`, `Mode`, `InfoBlock` and
  `Name` could quietly return wrong values.

## [0.5.0] - 2026-09-30

### Added

- `Mode` type with the constants `ModeRGB`, `ModeRGBInverse`, `ModeWS2812` and `ModeWS2812Mirror`, and `Mode.String`.
- `Device.Mode` reads the device mode, and `Device.SetMode` changes it (writes EEPROM; skipped when unchanged).
- `Model.SupportsMode` reports which modes a model allows.
- `ErrUnsupportedMode`, returned by `SetMode` for a mode the model does not support.

### Changed

- `SetLED` sends one report 5 instead of reading and rewriting the whole frame.
- `Model` gained an unexported field; caller-built models no longer equal predefined ones, support no modes, and `%+v` output includes a `modes` field.

## [0.4.0] - 2026-09-29

### Added

- Support for the BlinkStick Nano and Square on Linux, using hidapi's hidraw backend.
- `udev/60-blinkstick.rules` grants a logged-in desktop user access to a stick without root.

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

[Unreleased]: https://github.com/bawdo/go-blinkstick/compare/v0.6.1...HEAD
[0.6.1]: https://github.com/bawdo/go-blinkstick/compare/v0.6.0...v0.6.1
[0.6.0]: https://github.com/bawdo/go-blinkstick/compare/v0.5.0...v0.6.0
[0.5.0]: https://github.com/bawdo/go-blinkstick/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/bawdo/go-blinkstick/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/bawdo/go-blinkstick/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/bawdo/go-blinkstick/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/bawdo/go-blinkstick/releases/tag/v0.1.0

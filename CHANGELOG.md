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

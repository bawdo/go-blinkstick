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
| Name a stick and open it by name | yes | yes | v0.2.0 | `SetName`, `Name`, `OpenName`, `ListNamed`. Stored in info block 1 |
| Several sticks open at once | yes | yes | v0.1.0 | |
| Serial and firmware version | yes | yes | v0.1.0 | `Info` |
| Manufacturer and product strings | yes | yes | v0.1.0 | `Info` |
| Set all LEDs | yes | yes | v0.1.0 | `SetAll`, `Off` |
| Set one LED | yes | yes | v0.1.0 | `SetLED`. Since `v0.4.0` it uses report 5 |
| Set every LED at once | yes | yes | v0.1.0 | `SetFrame` |
| Read LED values back | yes | yes | v0.1.0 | `Frame`, `LED` |
| Info blocks 1 and 2 | yes | yes | v0.1.0 | `InfoBlock`, `SetInfoBlock`. 32 bytes each, stored in EEPROM |
| Set one LED directly (report 5) | yes | yes | v0.4.0 | `SetLED`. `[5, channel, index, R, G, B]`, RGB order (the frame is GRB). Works in modes 2 and 3. The channel byte is ignored and the index is not range checked |
| Read device mode (report 4) | yes | yes | v0.4.0 | `Mode`. `[4, mode]`. Both read mode 2 |
| Set device mode (report 4) | yes | yes | v0.4.0 | `SetMode`. Stored in EEPROM, blocks about 50 ms, skipped when the mode is unchanged, takes effect without a replug. 2 to 3 to 2 verified on both |
| Mode 2, WS2812 | yes | yes | n/a | The mode both are in, and the one every LED feature here needs. Probably the factory default |
| Mode 3, WS2812 mirror | yes | yes | v0.4.0 | Changes only report 1, whose colour goes to every LED. Reports 5 and 6 behave as in mode 2 |
| Modes 0 and 1, normal and inverse | n/a | n/a | n/a | For the BlinkStick Pro's RGB outputs. Upstream reports a Nano in mode 1 going dark until replugged. `SetMode` refuses them with `ErrUnsupportedMode` |
| Set LED 0, or every LED in mode 3 (report 1) | yes | yes | n/a | `[1, R, G, B]`, RGB order. Used by other BlinkStick software. Here `SetLED(0, c)` and `SetAll` do the same job in either mode |
| Report 10 | n/a | unknown | no | Square only, 2 bytes, reads 00 00. Undocumented; waiting on the manufacturer |
| LED count setting (report 0x81) | n/a | n/a | n/a | BlinkStick Flex only |
| 16, 32 and 64 LED frames (reports 7 to 9) | n/a | n/a | n/a | For longer strips |

## Host features

Features this package adds in software. They work the same on every supported device.

| Feature | Supported since | Notes |
|---|---|---|
| Model detection | v0.1.0 | From the serial and USB release number |
| Colour from hex or `r,g,b` | v0.1.0 | `ParseRGB`. Hex without the `#` since v0.2.0 |
| Brightness limit | v0.1.0 | `SetBrightnessLimit` |
| Blink, pulse and morph | v0.1.0 | Cancellable with a context |
| CSS colour names | v0.2.0 | `ParseRGB`, all 148 from CSS Color Level 4 |
| List CSS colour names | v0.3.0 | `ColourNames`, sorted, for shell completion |
| Colour as hex | v0.3.0 | `RGB.Hex`, lower case `#rrggbb` |
| Random colour | v0.2.0 | `RandomRGB`, `RandomVivid` |
| Inverse colours | v0.2.0 | `RGB.Inverse`, `SetInverse`. Not the firmware inverse mode |
| Reconnect after unplug | v0.2.0 | Automatic, repaints the last frame. `ErrDisconnected` |

## Platforms

| Platform | Supported since |
|---|---|
| macOS | v0.1.0 |
| Linux | v0.4.0 |
| Windows | no |

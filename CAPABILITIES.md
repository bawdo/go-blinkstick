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
| Set one LED | yes | yes | v0.1.0 | `SetLED`. Since `v0.9.0` it uses report 5 |
| Set every LED at once | yes | yes | v0.1.0 | `SetFrame` |
| Read LED values back | yes | yes | v0.1.0 | `Frame`, `LED` |
| Info blocks 1 and 2 | yes | yes | v0.1.0 | `InfoBlock`, `SetInfoBlock`. 32 bytes each, stored in EEPROM |
| Set one LED directly (report 5) | yes | yes | v0.9.0 | `SetLED`. `[5, channel, index, R, G, B]`, RGB order (the frame is GRB). Works in mode 2. The channel byte is ignored and the index is not range checked |
| Read device mode (report 4) | yes | yes | v0.9.0 | `Mode`. `[4, mode]`. Both read mode 2 |
| Set device mode (report 4) | unverified | unverified | no | Stored in EEPROM. Writing the current mode back is accepted, but a change to another mode has not been tried |
| Mode 2, WS2812 | yes | yes | n/a | The mode both are in, and the one every LED feature here needs. Probably the factory default |
| Mode 3, WS2812 mirror | unverified | unverified | no | Report 1 colour goes to every LED. Documented upstream, not tried |
| Modes 0 and 1, normal and inverse | n/a | n/a | n/a | For the BlinkStick Pro's RGB outputs. Upstream reports a Nano in mode 1 going dark until replugged |
| Report 10 | n/a | unknown | no | Square only, 2 bytes, read returns 00 00, purpose unknown |
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
| Linux | no |
| Windows | no |

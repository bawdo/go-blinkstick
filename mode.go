package blinkstick

import "fmt"

// Mode is the device's LED output mode, stored in EEPROM and read with
// report 4.
type Mode uint8

const (
	// ModeRGB is normal mode for the BlinkStick Pro RGB outputs.
	ModeRGB Mode = 0
	// ModeRGBInverse is inverse mode for the BlinkStick Pro RGB outputs.
	ModeRGBInverse Mode = 1
	// ModeWS2812 drives the WS2812 LEDs, which is what every LED method here needs.
	ModeWS2812 Mode = 2
	// ModeWS2812Mirror is ModeWS2812 where report 1 colours every LED, as sent by other BlinkStick software.
	ModeWS2812Mirror Mode = 3
)

// String returns the mode's name, or "mode(n)" for a value with no name.
func (m Mode) String() string {
	switch m {
	case ModeRGB:
		return "rgb"
	case ModeRGBInverse:
		return "rgb-inverse"
	case ModeWS2812:
		return "ws2812"
	case ModeWS2812Mirror:
		return "ws2812-mirror"
	}
	return fmt.Sprintf("mode(%d)", uint8(m))
}

// Mode reads the device's current mode. It returns ModeRGB and an error if
// the read fails.
func (d *Device) Mode() (Mode, error) {
	buf := make([]byte, modeReportSize)
	buf[0] = reportMode
	d.mu.Lock()
	defer d.mu.Unlock()
	if err := d.getLocked(buf); err != nil {
		return ModeRGB, err
	}
	return Mode(buf[1]), nil
}

package blinkstick

import (
	"context"
	"fmt"
	"time"
)

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

// After a mode write the stick refuses every transfer for about 12 to 15 ms,
// most likely while it writes EEPROM. SetMode reads report 4 every modePoll
// until the stick answers, for up to modeSettle.
const (
	modePoll   = 5 * time.Millisecond
	modeSettle = 250 * time.Millisecond
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

// SetMode switches the device to mode m. It returns ErrUnsupportedMode if the
// model does not allow m, which covers modes 0 and 1 on every model here.
//
// SetMode writes EEPROM, which wears with use, so call it once and not in a
// loop. It skips the write when the stick is already in m. The change takes
// effect at once, with no replug. The stick stops answering for a short time
// after the write, so SetMode waits until it reads back m, typically 12 to
// 15 ms. If the stick has not answered after 250 ms it returns
// ErrDisconnected.
//
// Mode 3 makes report 1 from other software colour every LED. This package's
// LED methods work the same in modes 2 and 3.
func (d *Device) SetMode(m Mode) error {
	if !d.info.Model.SupportsMode(m) {
		return fmt.Errorf("%w: %v on %s", ErrUnsupportedMode, m, d.info.Model.Name)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	buf := make([]byte, modeReportSize)
	buf[0] = reportMode
	if err := d.getLocked(buf); err != nil {
		return err
	}
	if Mode(buf[1]) == m {
		return nil
	}
	if err := d.sendLocked([]byte{reportMode, byte(m)}); err != nil {
		return err
	}
	return d.waitModeLocked(m)
}

// waitModeLocked reads report 4 every modePoll until it shows m. It calls the
// transport directly: the back to back retries in transferLocked would all
// land inside the busy window and reconnect a stick that never went away.
// d.mu must be held and d.t must not be nil.
func (d *Device) waitModeLocked(m Mode) error {
	buf := make([]byte, modeReportSize)
	buf[0] = reportMode
	var err error
	for waited := time.Duration(0); ; waited += modePoll {
		if _, err = get(buf)(d.t); err == nil {
			if Mode(buf[1]) == m {
				return nil
			}
			err = fmt.Errorf("mode reads %v", Mode(buf[1]))
		}
		if waited >= modeSettle {
			break
		}
		if err := sleep(context.Background(), modePoll); err != nil {
			return err
		}
	}
	return &disconnectedError{serial: d.info.Serial,
		cause: fmt.Errorf("no answer %v after setting mode %v: %w", modeSettle, m, err)}
}

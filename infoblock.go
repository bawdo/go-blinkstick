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

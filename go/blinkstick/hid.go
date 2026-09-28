package blinkstick

import (
	"errors"
	"fmt"

	"github.com/sstallion/go-hid"
)

// hidDevice closes the hidapi library along with the device.
type hidDevice struct {
	*hid.Device
}

func (d hidDevice) Close() error {
	return errors.Join(d.Device.Close(), hid.Exit())
}

// OpenFirst opens the first BlinkStick found. It deliberately avoids
// hid.Enumerate, which can panic on macOS (sstallion/go-hid issue #18).
func OpenFirst() (*Stick, error) {
	if err := hid.Init(); err != nil {
		return nil, fmt.Errorf("blinkstick: hid init: %w", err)
	}
	dev, err := hid.OpenFirst(VendorID, ProductID)
	if err != nil {
		hid.Exit()
		return nil, fmt.Errorf("blinkstick: no device found (%04x:%04x): %w", VendorID, ProductID, err)
	}
	return New(hidDevice{dev}), nil
}

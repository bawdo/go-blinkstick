package blinkstick

import (
	"errors"
	"fmt"
	"sync"
)

// Errors returned by this package. Check them with errors.Is.
var (
	ErrNotFound    = errors.New("blinkstick: device not found")
	ErrUnsupported = errors.New("blinkstick: unsupported model")
	ErrOutOfRange  = errors.New("blinkstick: out of range")
	ErrClosed      = errors.New("blinkstick: device closed")
)

// transferAttempts covers the firmware being busy straight after a write and
// intermittent IOHIDDeviceSetReport failures on macOS (go-hid issue #15).
const transferAttempts = 3

// transport is the slice of a HID device that Device needs.
type transport interface {
	SendFeatureReport(p []byte) (int, error)
	GetFeatureReport(p []byte) (int, error)
	Close() error
}

// Device is an open BlinkStick. Its methods are safe for concurrent use,
// but two effects running at once on one Device interleave their frames.
type Device struct {
	mu    sync.Mutex
	t     transport // nil once closed
	info  Info
	limit uint8
}

func newDevice(t transport, info Info) *Device {
	return &Device{t: t, info: info, limit: 255}
}

// Info returns the identity of the device.
func (d *Device) Info() Info {
	return d.info
}

// Close releases the device. Using a Device after Close returns ErrClosed.
func (d *Device) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.t == nil {
		return ErrClosed
	}
	err := d.t.Close()
	d.t = nil
	return err
}

// SetBrightnessLimit caps every channel written from now on, scaling each
// as v * limit / 255. 255, the default, means no limit. Frame and LED return
// the scaled values stored on the device.
func (d *Device) SetBrightnessLimit(limit uint8) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.limit = limit
}

// SetAll sets every LED to c.
func (d *Device) SetAll(c RGB) error {
	return d.SetFrame(fill(d.info.Model.LEDs, c))
}

// Off turns every LED off.
func (d *Device) Off() error {
	return d.SetAll(Off)
}

// SetFrame sets every LED at once. len(leds) must equal Info().Model.LEDs.
func (d *Device) SetFrame(leds []RGB) error {
	if n := d.info.Model.LEDs; len(leds) != n {
		return fmt.Errorf("%w: frame has %d LEDs, %s has %d",
			ErrOutOfRange, len(leds), d.info.Model.Name, n)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	scaled := make([]RGB, len(leds))
	for i, c := range leds {
		scaled[i] = c.scale(d.limit)
	}
	return d.writeFrameLocked(scaled)
}

// writeFrameLocked sends LED values as given, without scaling. d.mu must be
// held.
func (d *Device) writeFrameLocked(leds []RGB) error {
	return d.sendLocked(encodeFrame(leds))
}

func (d *Device) sendLocked(p []byte) error {
	return d.transferLocked("send", p[0], func(t transport) (int, error) {
		return t.SendFeatureReport(p)
	})
}

func (d *Device) getLocked(p []byte) error {
	return d.transferLocked("get", p[0], func(t transport) (int, error) {
		return t.GetFeatureReport(p)
	})
}

// transferLocked runs f up to transferAttempts times. d.mu must be held.
func (d *Device) transferLocked(op string, id byte, f func(transport) (int, error)) error {
	if d.t == nil {
		return ErrClosed
	}
	var err error
	for range transferAttempts {
		if _, err = f(d.t); err == nil {
			return nil
		}
	}
	return fmt.Errorf("blinkstick: %s report %d failed after %d attempts: %w",
		op, id, transferAttempts, err)
}

// fill returns n copies of c.
func fill(n int, c RGB) []RGB {
	leds := make([]RGB, n)
	for i := range leds {
		leds[i] = c
	}
	return leds
}

// Frame reads every LED back from the device.
func (d *Device) Frame() ([]RGB, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.readFrameLocked()
}

// LED reads LED i back from the device.
func (d *Device) LED(i int) (RGB, error) {
	if err := d.checkIndex(i); err != nil {
		return RGB{}, err
	}
	leds, err := d.Frame()
	if err != nil {
		return RGB{}, err
	}
	return leds[i], nil
}

// SetLED sets LED i to c and leaves the others as they are. It reads the
// frame back and rewrites it, because setting a single LED directly (report
// 5) needs a firmware mode that is unverified on v3 hardware.
func (d *Device) SetLED(i int, c RGB) error {
	if err := d.checkIndex(i); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	leds, err := d.readFrameLocked()
	if err != nil {
		return err
	}
	leds[i] = c.scale(d.limit)
	return d.writeFrameLocked(leds)
}

// readFrameLocked reads report 6. d.mu must be held.
func (d *Device) readFrameLocked() ([]RGB, error) {
	buf := make([]byte, frameReportSize)
	buf[0] = reportFrame
	if err := d.getLocked(buf); err != nil {
		return nil, err
	}
	return decodeFrame(buf, d.info.Model.LEDs), nil
}

func (d *Device) checkIndex(i int) error {
	if n := d.info.Model.LEDs; i < 0 || i >= n {
		return fmt.Errorf("%w: LED %d, %s has %d", ErrOutOfRange, i, d.info.Model.Name, n)
	}
	return nil
}

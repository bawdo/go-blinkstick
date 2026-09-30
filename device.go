package blinkstick

import (
	"errors"
	"fmt"
	"slices"
	"sync"
)

// Errors returned by this package. Check them with errors.Is. ErrNotFound is
// also returned when a stick is attached but cannot be opened, for example
// because another process holds it (macOS opens devices exclusively).
// ErrDisconnected means an open stick was unplugged and could not be
// reopened; errors.Is also matches it to ErrNotFound.
var (
	ErrNotFound    = errors.New("blinkstick: device not found")
	ErrUnsupported = errors.New("blinkstick: unsupported model")
	ErrOutOfRange  = errors.New("blinkstick: out of range")
	ErrClosed      = errors.New("blinkstick: device closed")

	ErrInvalidName   = errors.New("blinkstick: invalid name")
	ErrDuplicateName = errors.New("blinkstick: name used by more than one stick")
	ErrDisconnected  = error(&disconnectedError{})
)

// disconnectedError is returned when a stick cannot be reopened. It matches
// both ErrDisconnected and ErrNotFound, and unwraps to the reason.
type disconnectedError struct {
	serial string
	cause  error
}

func (e *disconnectedError) Error() string {
	msg := "blinkstick: device disconnected"
	if e.serial != "" {
		msg += ": " + e.serial
	}
	// "Not found" is what disconnected means, so only other causes add
	// anything worth reading.
	if e.cause != nil && !errors.Is(e.cause, ErrNotFound) {
		msg += ": " + e.cause.Error()
	}
	return msg
}

func (e *disconnectedError) Is(target error) bool {
	return target == ErrDisconnected || target == ErrNotFound
}

func (e *disconnectedError) Unwrap() error { return e.cause }

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
//
// If the stick is unplugged, the next call that fails reopens it by serial,
// repaints the last frame written (a replugged stick comes back dark) and
// carries on. If the stick is still missing it returns ErrDisconnected, and
// the call after that tries again.
type Device struct {
	mu      sync.Mutex
	t       transport // nil once closed or while disconnected
	closed  bool
	reopen  func() (transport, error) // nil if the device cannot reconnect
	last    []RGB                     // last frame written, before flipping
	info    Info
	limit   uint8
	inverse bool
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
	if d.closed {
		return ErrClosed
	}
	d.closed = true
	if d.t == nil { // disconnected, nothing left to release
		return nil
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

// SetInverse flips every channel written from now on to 255 - v, after any
// brightness limit, for LEDs wired so that 255 means off. Frame and LED flip
// values back, so they return what was written. It does not repaint the
// LEDs, so call it before setting them.
func (d *Device) SetInverse(on bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.inverse = on
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

// writeFrameLocked sends LED values without scaling, flipped if inverse is
// on. d.mu must be held.
func (d *Device) writeFrameLocked(leds []RGB) error {
	if err := d.sendLocked(encodeFrame(d.flipLocked(leds))); err != nil {
		return err
	}
	d.last = slices.Clone(leds)
	return nil
}

// flipLocked returns leds inverted if inverse is on, and leds itself if not.
// Inverting is its own undo, so reads use it too. d.mu must be held.
func (d *Device) flipLocked(leds []RGB) []RGB {
	if !d.inverse {
		return leds
	}
	out := make([]RGB, len(leds))
	for i, c := range leds {
		out[i] = c.Inverse()
	}
	return out
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

// transferLocked runs f up to transferAttempts times. If they all fail and
// the device can reconnect, it reopens the stick and runs f again. d.mu must
// be held.
func (d *Device) transferLocked(op string, id byte, f func(transport) (int, error)) error {
	if d.closed {
		return ErrClosed
	}
	if d.t != nil {
		err := d.tryLocked(f)
		if err == nil {
			return nil
		}
		if d.reopen == nil {
			return transferError(op, id, err)
		}
		d.t.Close()
		d.t = nil
	}
	// A failed frame send is about to be replaced, so repainting the old
	// frame first would only flash it.
	if err := d.reconnectLocked(!(op == "send" && id == reportFrame)); err != nil {
		return err
	}
	if err := d.tryLocked(f); err != nil {
		return transferError(op, id, err)
	}
	return nil
}

// tryLocked runs f up to transferAttempts times. d.mu must be held and d.t
// must not be nil.
func (d *Device) tryLocked(f func(transport) (int, error)) error {
	var err error
	for range transferAttempts {
		if _, err = f(d.t); err == nil {
			return nil
		}
	}
	return err
}

// reconnectLocked reopens the stick by serial and, if repaint is set,
// rewrites the last frame. d.mu must be held and d.t must be nil.
func (d *Device) reconnectLocked(repaint bool) error {
	t, err := d.reopen()
	if err != nil {
		return &disconnectedError{serial: d.info.Serial, cause: err}
	}
	d.t = t
	if !repaint || d.last == nil {
		return nil
	}
	p := encodeFrame(d.flipLocked(d.last))
	if err := d.tryLocked(func(t transport) (int, error) { return t.SendFeatureReport(p) }); err != nil {
		d.t.Close()
		d.t = nil
		return &disconnectedError{serial: d.info.Serial,
			cause: fmt.Errorf("repaint after reopening: %w", transferError("send", reportFrame, err))}
	}
	return nil
}

func transferError(op string, id byte, err error) error {
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

// Frame reads every LED back from the device. Values are those stored on the
// device, so they reflect any brightness limit in force when they were
// written. With SetInverse on they are flipped back.
func (d *Device) Frame() ([]RGB, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.readFrameLocked()
}

// LED reads LED i back from the device, as Frame does.
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

// SetLED sets LED i to c and leaves the others as they are. It sends one
// report (report 5), which takes about a millisecond. The first call on a
// Device that has written nothing also reads the frame back once, so a
// reconnect can repaint the other LEDs.
func (d *Device) SetLED(i int, c RGB) error {
	if err := d.checkIndex(i); err != nil {
		return err
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.last == nil {
		leds, err := d.readFrameLocked()
		if err != nil {
			return err
		}
		d.last = leds
	}
	scaled := c.scale(d.limit)
	if err := d.sendLocked(encodeLED(i, d.flipLocked([]RGB{scaled})[0])); err != nil {
		return err
	}
	d.last[i] = scaled
	return nil
}

// readFrameLocked reads report 6. d.mu must be held.
func (d *Device) readFrameLocked() ([]RGB, error) {
	buf := make([]byte, frameReportSize)
	buf[0] = reportFrame
	if err := d.getLocked(buf); err != nil {
		return nil, err
	}
	return d.flipLocked(decodeFrame(buf, d.info.Model.LEDs)), nil
}

func (d *Device) checkIndex(i int) error {
	if n := d.info.Model.LEDs; i < 0 || i >= n {
		return fmt.Errorf("%w: LED %d, %s has %d", ErrOutOfRange, i, d.info.Model.Name, n)
	}
	return nil
}

// backend finds and opens BlinkSticks. hidBackend is the real one; tests
// swap in a fake.
type backend interface {
	list() ([]deviceInfo, error)
	open(serial string) (transport, deviceInfo, error) // "" opens the first found
}

var sys backend = hidBackend{}

// List returns every attached BlinkStick, including models this package does
// not support yet (their Model.Name is "unknown").
func List() ([]Info, error) {
	dis, err := sys.list()
	if err != nil {
		return nil, err
	}
	infos := make([]Info, len(dis))
	for i, di := range dis {
		infos[i] = newInfo(di)
	}
	return infos, nil
}

// Open opens the first BlinkStick found. If the first stick found is a model
// this package does not support, Open returns ErrUnsupported even when a
// supported stick is also attached; use List and OpenSerial to choose one.
func Open() (*Device, error) {
	return open("")
}

// OpenSerial opens the BlinkStick with the given serial, as reported by
// List.
func OpenSerial(serial string) (*Device, error) {
	if serial == "" {
		return nil, fmt.Errorf("%w: empty serial", ErrNotFound)
	}
	return open(serial)
}

func open(serial string) (*Device, error) {
	t, di, err := sys.open(serial)
	if err != nil {
		return nil, err
	}
	info := newInfo(di)
	if info.Model == unknownModel {
		t.Close()
		return nil, fmt.Errorf("%w: %s (release %#04x)", ErrUnsupported, info.Serial, di.release)
	}
	return newReconnectingDevice(t, info), nil
}

// newReconnectingDevice returns a Device that reopens its stick by serial
// through the current backend if it is unplugged.
func newReconnectingDevice(t transport, info Info) *Device {
	d := newDevice(t, info)
	b := sys
	d.reopen = func() (transport, error) {
		t, _, err := b.open(info.Serial)
		return t, err
	}
	return d
}

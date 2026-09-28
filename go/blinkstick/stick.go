package blinkstick

import "fmt"

// sendAttempts covers intermittent IOHIDDeviceSetReport failures on macOS
// (sstallion/go-hid issue #15).
const sendAttempts = 3

// FeatureWriter is the slice of a HID device that Stick needs.
type FeatureWriter interface {
	SendFeatureReport(p []byte) (int, error)
	Close() error
}

// Stick is an open BlinkStick.
type Stick struct {
	w FeatureWriter
}

// New wraps an open HID device.
func New(w FeatureWriter) *Stick {
	return &Stick{w: w}
}

// SetAll sets every LED on channel 0 to c.
func (s *Stick) SetAll(c RGB) error {
	report := FrameReport(0, c)
	var err error
	for i := 0; i < sendAttempts; i++ {
		if _, err = s.w.SendFeatureReport(report); err == nil {
			return nil
		}
	}
	return fmt.Errorf("blinkstick: set colour failed after %d attempts: %w", sendAttempts, err)
}

// Close releases the device.
func (s *Stick) Close() error {
	return s.w.Close()
}

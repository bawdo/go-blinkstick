// Package blinkstick drives BlinkStick USB LED devices (Nano, Square) over
// HID feature reports.
package blinkstick

// USB identifiers shared by every BlinkStick model.
const (
	VendorID  uint16 = 0x20A0
	ProductID uint16 = 0x41E5
)

// reportFrame8 is the feature report that sets 8 LEDs in one transfer.
const reportFrame8 = 0x06

// FrameLEDs is the number of LEDs carried by a report 6 frame.
const FrameLEDs = 8

// RGB is a colour with 8 bits per channel.
type RGB struct{ R, G, B uint8 }

var (
	White = RGB{R: 255, G: 255, B: 255}
	Off   = RGB{}
)

// FrameReport builds report 6, setting all FrameLEDs LEDs on channel to c.
// The device expects each LED as G, R, B. The first byte is the report ID.
func FrameReport(channel uint8, c RGB) []byte {
	buf := make([]byte, 2+FrameLEDs*3)
	buf[0] = reportFrame8
	buf[1] = channel
	for i := 0; i < FrameLEDs; i++ {
		buf[2+i*3] = c.G
		buf[3+i*3] = c.R
		buf[4+i*3] = c.B
	}
	return buf
}

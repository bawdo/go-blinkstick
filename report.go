package blinkstick

// USB identifiers shared by every BlinkStick model.
const (
	vendorID  uint16 = 0x20A0
	productID uint16 = 0x41E5
)

// Feature report IDs and sizes. Sizes include the leading report ID byte
// and match the device's HID report descriptor exactly.
const (
	reportInfo1 = 2
	reportInfo2 = 3
	reportLED   = 5
	reportFrame = 6

	ledReportSize   = 6 // ID, channel, index, R, G, B
	frameLEDs       = 8
	frameReportSize = 2 + frameLEDs*3 // ID, channel, 8 x (G, R, B)
	infoBlockSize   = 32
	infoReportSize  = 1 + infoBlockSize
)

// encodeFrame builds report 6 for channel 0. The device expects each LED as
// G, R, B. Slots beyond len(leds) are zero. len(leds) must not exceed
// frameLEDs.
func encodeFrame(leds []RGB) []byte {
	buf := make([]byte, frameReportSize)
	buf[0] = reportFrame
	for i, c := range leds {
		buf[2+i*3], buf[3+i*3], buf[4+i*3] = c.G, c.R, c.B
	}
	return buf
}

// encodeLED builds report 5, which sets one LED. Unlike the frame, it takes
// R, G, B in that order. The firmware ignores the channel byte, so it is
// always 0, and does not range check i, so callers must.
func encodeLED(i int, c RGB) []byte {
	return []byte{reportLED, 0, byte(i), c.R, c.G, c.B}
}

// decodeFrame reads the first n LEDs out of a report 6 buffer.
func decodeFrame(buf []byte, n int) []RGB {
	leds := make([]RGB, n)
	for i := range leds {
		leds[i] = RGB{G: buf[2+i*3], R: buf[3+i*3], B: buf[4+i*3]}
	}
	return leds
}

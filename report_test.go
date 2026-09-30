package blinkstick

import (
	"slices"
	"testing"
)

func TestEncodeFrameLayout(t *testing.T) {
	got := encodeFrame([]RGB{{R: 0x11, G: 0x22, B: 0x33}, {R: 0x44, G: 0x55, B: 0x66}})
	if len(got) != frameReportSize || frameReportSize != 26 {
		t.Fatalf("len = %d, want 26", len(got))
	}
	want := []byte{0x06, 0x00, 0x22, 0x11, 0x33, 0x55, 0x44, 0x66}
	if !slices.Equal(got[:8], want) {
		t.Errorf("head = % x, want % x", got[:8], want)
	}
	for i, v := range got[8:] {
		if v != 0 {
			t.Fatalf("byte %d = %#x, want 0 for unused slots", i+8, v)
		}
	}
}

func TestDecodeFrameRoundTrip(t *testing.T) {
	leds := []RGB{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}, {10, 11, 12}, {13, 14, 15}, {16, 17, 18}, {19, 20, 21}, {22, 23, 24}}
	if got := decodeFrame(encodeFrame(leds), 8); !slices.Equal(got, leds) {
		t.Errorf("round trip = %v, want %v", got, leds)
	}
	if got := decodeFrame(encodeFrame(leds), 2); !slices.Equal(got, leds[:2]) {
		t.Errorf("first two = %v, want %v", got, leds[:2])
	}
}

func TestReportSizes(t *testing.T) {
	if infoReportSize != 33 || infoBlockSize != 32 {
		t.Errorf("info report %d, block %d", infoReportSize, infoBlockSize)
	}
	if vendorID != 0x20A0 || productID != 0x41E5 {
		t.Errorf("USB ID %04x:%04x", vendorID, productID)
	}
}

func TestEncodeLEDLayout(t *testing.T) {
	got := encodeLED(3, RGB{R: 0x11, G: 0x22, B: 0x33})
	want := []byte{0x05, 0x00, 0x03, 0x11, 0x22, 0x33} // RGB on the wire, channel 0
	if len(got) != ledReportSize || ledReportSize != 6 {
		t.Fatalf("len = %d, want 6", len(got))
	}
	if !slices.Equal(got, want) {
		t.Errorf("encodeLED = % x, want % x", got, want)
	}
}

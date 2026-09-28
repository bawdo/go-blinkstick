package blinkstick

import "testing"

func TestFrameReportLayout(t *testing.T) {
	got := FrameReport(0, RGB{R: 0x11, G: 0x22, B: 0x33})

	if len(got) != 26 {
		t.Fatalf("len = %d, want 26", len(got))
	}
	if got[0] != 0x06 {
		t.Errorf("report ID = %#x, want 0x06", got[0])
	}
	if got[1] != 0x00 {
		t.Errorf("channel = %#x, want 0x00", got[1])
	}
	for i := 0; i < FrameLEDs; i++ {
		g, r, b := got[2+i*3], got[3+i*3], got[4+i*3]
		if g != 0x22 || r != 0x11 || b != 0x33 {
			t.Errorf("LED %d = G%#x R%#x B%#x, want G0x22 R0x11 B0x33", i, g, r, b)
		}
	}
}

func TestFrameReportChannel(t *testing.T) {
	got := FrameReport(2, White)
	if got[1] != 2 {
		t.Errorf("channel = %d, want 2", got[1])
	}
}

func TestFrameReportOffIsAllZero(t *testing.T) {
	got := FrameReport(0, Off)
	for i, v := range got[1:] {
		if v != 0 {
			t.Fatalf("byte %d = %#x, want 0", i+1, v)
		}
	}
}

package blinkstick

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestModeString(t *testing.T) {
	tests := []struct {
		m    Mode
		want string
	}{
		{ModeRGB, "rgb"},
		{ModeRGBInverse, "rgb-inverse"},
		{ModeWS2812, "ws2812"},
		{ModeWS2812Mirror, "ws2812-mirror"},
		{Mode(9), "mode(9)"},
	}
	for _, tt := range tests {
		if got := tt.m.String(); got != tt.want {
			t.Errorf("Mode(%d).String() = %q, want %q", uint8(tt.m), got, tt.want)
		}
	}
}

func TestModeReadsReport4(t *testing.T) {
	d, ft := openFake(Nano)
	got, err := d.Mode()
	if err != nil || got != ModeWS2812 {
		t.Fatalf("Mode() = %v, %v, want %v, nil", got, err, ModeWS2812)
	}
	if _, err := ft.SendFeatureReport([]byte{4, 3}); err != nil {
		t.Fatal(err)
	}
	got, err = d.Mode()
	if err != nil || got != ModeWS2812Mirror {
		t.Fatalf("Mode() = %v, %v, want %v, nil", got, err, ModeWS2812Mirror)
	}
}

func TestModeReadError(t *testing.T) {
	d, ft := openFake(Square)
	ft.fail = transferAttempts
	got, err := d.Mode()
	if err == nil || !strings.Contains(err.Error(), "get report 4") {
		t.Fatalf("err = %v, want one mentioning \"get report 4\"", err)
	}
	if got != ModeRGB {
		t.Errorf("Mode() = %v on error, want %v", got, ModeRGB)
	}
}

func TestSetModeWritesReport4(t *testing.T) {
	d, ft := openFake(Square)
	if err := d.SetMode(ModeWS2812Mirror); err != nil {
		t.Fatalf("SetMode(mirror): %v", err)
	}
	if got := ft.lastSend(); !slices.Equal(got, []byte{4, 3}) {
		t.Errorf("last send = %v, want [4 3]", got)
	}
	if got, err := d.Mode(); err != nil || got != ModeWS2812Mirror {
		t.Errorf("Mode() = %v, %v, want %v, nil", got, err, ModeWS2812Mirror)
	}
	if err := d.SetMode(ModeWS2812); err != nil {
		t.Fatalf("SetMode(ws2812): %v", err)
	}
	if got := ft.lastSend(); !slices.Equal(got, []byte{4, 2}) {
		t.Errorf("last send = %v, want [4 2]", got)
	}
}

func TestSetModeSameModeSkipsWrite(t *testing.T) {
	d, ft := openFake(Nano)
	if err := d.SetMode(ModeWS2812); err != nil {
		t.Fatalf("SetMode: %v", err)
	}
	if n := ft.sendCount(); n != 0 {
		t.Errorf("sends = %d, want 0", n)
	}
}

func TestSetModeRejectsUnsupported(t *testing.T) {
	for _, m := range []Model{Nano, Square} {
		for _, x := range []Mode{ModeRGB, ModeRGBInverse, Mode(4)} {
			d, ft := openFake(m)
			err := d.SetMode(x)
			if !errors.Is(err, ErrUnsupportedMode) {
				t.Errorf("%s SetMode(%v) = %v, want ErrUnsupportedMode", m.Name, x, err)
				continue
			}
			if !strings.Contains(err.Error(), x.String()) || !strings.Contains(err.Error(), m.Name) {
				t.Errorf("error %q should name %v and %s", err, x, m.Name)
			}
			if ft.sendCount() != 0 || ft.getCount() != 0 {
				t.Errorf("%s SetMode(%v): sends = %d, gets = %d, want 0, 0", m.Name, x, ft.sendCount(), ft.getCount())
			}
		}
	}
}

// Mode 3 changes only report 1, so the LED methods behave as in mode 2.
func TestLEDMethodsInMirrorMode(t *testing.T) {
	d, _ := openFake(Square)
	if err := d.SetMode(ModeWS2812Mirror); err != nil {
		t.Fatalf("SetMode: %v", err)
	}
	palette := make([]RGB, Square.LEDs)
	for i := range palette {
		palette[i] = RGB{R: uint8(10 * (i + 1)), G: uint8(i), B: uint8(200 - i)}
	}
	if err := d.SetFrame(palette); err != nil {
		t.Fatalf("SetFrame: %v", err)
	}
	if err := d.SetLED(1, RGB{R: 1}); err != nil {
		t.Fatalf("SetLED: %v", err)
	}
	want := slices.Clone(palette)
	want[1] = RGB{R: 1}
	got, err := d.Frame()
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("Frame() = %v, %v, want %v", got, err, want)
	}
}

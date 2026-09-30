package blinkstick

import (
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

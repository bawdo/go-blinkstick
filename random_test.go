package blinkstick

import "testing"

func TestRandomRGBVaries(t *testing.T) {
	seen := map[RGB]bool{}
	for range 100 {
		seen[RandomRGB()] = true
	}
	if len(seen) < 90 {
		t.Errorf("100 calls gave %d distinct colours, want at least 90", len(seen))
	}
}

func TestRandomVividIsFullySaturated(t *testing.T) {
	seen := map[RGB]bool{}
	for range 1000 {
		c := RandomVivid()
		seen[c] = true
		hi := max(c.R, c.G, c.B)
		lo := min(c.R, c.G, c.B)
		if hi != 255 || lo != 0 {
			t.Fatalf("RandomVivid() = %v, want one channel 255 and one 0", c)
		}
	}
	if len(seen) < 500 {
		t.Errorf("1000 calls gave %d distinct colours, want at least 500", len(seen))
	}
}

func TestHue(t *testing.T) {
	tests := []struct {
		h    int
		want RGB
	}{
		{0, RGB{255, 0, 0}},
		{128, RGB{255, 128, 0}},
		{255, RGB{255, 255, 0}},
		{510, RGB{0, 255, 0}},
		{765, RGB{0, 255, 255}},
		{1020, RGB{0, 0, 255}},
		{1275, RGB{255, 0, 255}},
		{1529, RGB{255, 0, 1}},
	}
	for _, tt := range tests {
		if got := hue(tt.h); got != tt.want {
			t.Errorf("hue(%d) = %v, want %v", tt.h, got, tt.want)
		}
	}
}

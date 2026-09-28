package blinkstick

import "testing"

func TestParseRGB(t *testing.T) {
	tests := []struct {
		in   string
		want RGB
	}{
		{"#ff8800", RGB{255, 136, 0}},
		{"#FF8800", RGB{255, 136, 0}},
		{"#f80", RGB{255, 136, 0}},
		{"#000", Off},
		{"#ffffff", White},
		{"255,136,0", RGB{255, 136, 0}},
		{" 1, 2 ,3 ", RGB{1, 2, 3}},
	}
	for _, tt := range tests {
		got, err := ParseRGB(tt.in)
		if err != nil {
			t.Errorf("ParseRGB(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("ParseRGB(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestParseRGBRejects(t *testing.T) {
	for _, in := range []string{
		"", "#", "#ff88", "#ff88000", "#gg0000", "ff8800",
		"1,2", "1,2,3,4", "256,0,0", "-1,0,0", "a,b,c",
	} {
		if got, err := ParseRGB(in); err == nil {
			t.Errorf("ParseRGB(%q) = %v, want error", in, got)
		}
	}
}

func TestScale(t *testing.T) {
	tests := []struct {
		c     RGB
		limit uint8
		want  RGB
	}{
		{White, 255, White},
		{White, 0, Off},
		{RGB{255, 128, 0}, 128, RGB{128, 64, 0}},
		{RGB{10, 20, 30}, 255, RGB{10, 20, 30}},
	}
	for _, tt := range tests {
		if got := tt.c.scale(tt.limit); got != tt.want {
			t.Errorf("%v.scale(%d) = %v, want %v", tt.c, tt.limit, got, tt.want)
		}
	}
}

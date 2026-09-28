package blinkstick

import (
	"strings"
	"testing"
)

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
		{"ff8800", RGB{255, 136, 0}},
		{"f80", RGB{255, 136, 0}},
		{" FFF ", White},
		{"red", RGB{255, 0, 0}},
		{"CornflowerBlue", RGB{100, 149, 237}},
		{" rebeccapurple ", RGB{102, 51, 153}},
		{"grey", RGB{128, 128, 128}},
		{"gray", RGB{128, 128, 128}},
		{"lightslategrey", RGB{119, 136, 153}},
		{"black", Off},
		{"white", White},
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
		"", "#", "#ff88", "#ff88000", "#gg0000", "ff88", "ff88000", "gg0000",
		"transparent", "currentcolor", "notacolour", "red ish", "#red",
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

func TestColourNamesTable(t *testing.T) {
	if n := len(colourNames); n != 148 {
		t.Errorf("colourNames has %d entries, want 148 (CSS Color Level 4)", n)
	}
	for name := range colourNames {
		if name != strings.ToLower(name) {
			t.Errorf("colourNames key %q is not lower case", name)
		}
		if _, err := parseHex(name, name); err == nil {
			t.Errorf("colourNames key %q also parses as bare hex", name)
		}
	}
}

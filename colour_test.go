package blinkstick

import (
	"slices"
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

func TestColourNamesCoversTable(t *testing.T) {
	names := ColourNames()
	if len(names) != len(colourNames) {
		t.Fatalf("ColourNames returned %d names, want %d", len(names), len(colourNames))
	}
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if seen[name] {
			t.Errorf("ColourNames returned %q twice", name)
		}
		seen[name] = true
		if _, ok := colourNames[name]; !ok {
			t.Errorf("ColourNames returned %q, which is not in colourNames", name)
		}
	}
}

func TestColourNamesSorted(t *testing.T) {
	names := ColourNames()
	if !slices.IsSorted(names) {
		t.Errorf("ColourNames is not sorted: %v", names)
	}
}

func TestColourNamesRoundTrip(t *testing.T) {
	for _, name := range ColourNames() {
		got, err := ParseRGB(name)
		if err != nil {
			t.Errorf("ParseRGB(%q): %v", name, err)
			continue
		}
		if want := colourNames[name]; got != want {
			t.Errorf("ParseRGB(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestColourNamesFreshSlice(t *testing.T) {
	first := ColourNames()
	if len(first) == 0 {
		t.Fatal("ColourNames returned no names")
	}
	want := first[0]
	first[0] = "mutated"
	if got := ColourNames()[0]; got != want {
		t.Errorf("after mutating the returned slice, ColourNames()[0] = %q, want %q", got, want)
	}
}

func TestHex(t *testing.T) {
	tests := []struct {
		c    RGB
		want string
	}{
		{Off, "#000000"},
		{White, "#ffffff"},
		{RGB{R: 1}, "#010000"},
		{RGB{G: 1}, "#000100"},
		{RGB{B: 1}, "#000001"},
		{RGB{255, 136, 0}, "#ff8800"},
		{RGB{100, 149, 237}, "#6495ed"},
	}
	for _, tt := range tests {
		if got := tt.c.Hex(); got != tt.want {
			t.Errorf("%v.Hex() = %q, want %q", tt.c, got, tt.want)
		}
	}
}

func TestHexRoundTrip(t *testing.T) {
	for _, c := range []RGB{Off, White, {R: 1}, {12, 34, 56}, {255, 136, 0}} {
		got, err := ParseRGB(c.Hex())
		if err != nil {
			t.Errorf("ParseRGB(%q): %v", c.Hex(), err)
			continue
		}
		if got != c {
			t.Errorf("ParseRGB(%v.Hex()) = %v", c, got)
		}
	}
}

package blinkstick

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"
)

// RGB is a colour with 8 bits per channel.
type RGB struct{ R, G, B uint8 }

// Common values.
var (
	Off   = RGB{}
	White = RGB{R: 255, G: 255, B: 255}
)

// ParseRGB parses a colour in any of these forms:
//
//   - hex "#rgb" or "#rrggbb", with or without the "#"
//   - "r,g,b" with decimal channels from 0 to 255
//   - a CSS colour name such as "cornflowerblue", in any case
func ParseRGB(s string) (RGB, error) {
	s = strings.TrimSpace(s)
	if hex, ok := strings.CutPrefix(s, "#"); ok {
		return parseHex(s, hex)
	}
	if c, ok := colourNames[strings.ToLower(s)]; ok {
		return c, nil
	}
	parts := strings.Split(s, ",")
	if len(parts) == 1 {
		// No CSS name is made only of hex digits, so a bare hex code cannot
		// shadow a name.
		return parseHex(s, s)
	}
	if len(parts) != 3 {
		return RGB{}, invalid(s)
	}
	var v [3]uint8
	for i, p := range parts {
		n, err := strconv.ParseUint(strings.TrimSpace(p), 10, 8)
		if err != nil {
			return RGB{}, invalid(s)
		}
		v[i] = uint8(n)
	}
	return RGB{R: v[0], G: v[1], B: v[2]}, nil
}

func parseHex(s, hex string) (RGB, error) {
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return RGB{}, invalid(s)
	}
	n, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return RGB{}, invalid(s)
	}
	return RGB{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n)}, nil
}

func invalid(s string) error {
	return fmt.Errorf("blinkstick: invalid colour %q", s)
}

// scale caps each channel at limit, keeping the hue: v * limit / 255.
func (c RGB) scale(limit uint8) RGB {
	f := func(v uint8) uint8 { return uint8(uint16(v) * uint16(limit) / 255) }
	return RGB{R: f(c.R), G: f(c.G), B: f(c.B)}
}

// Inverse returns c with each channel flipped: 255 - v.
func (c RGB) Inverse() RGB {
	return RGB{R: 255 - c.R, G: 255 - c.G, B: 255 - c.B}
}

// ColourNames returns every CSS colour name that ParseRGB accepts, lower case
// and sorted. Each call returns a new slice, so callers may modify it.
// ParseRGB gives the value of a name.
func ColourNames() []string {
	return slices.Sorted(maps.Keys(colourNames))
}

// Hex returns c as lower case "#rrggbb".
func (c RGB) Hex() string {
	return fmt.Sprintf("#%02x%02x%02x", c.R, c.G, c.B)
}

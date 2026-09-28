package blinkstick

import "math/rand/v2"

// hues is the number of distinct fully saturated colours hue produces: six
// segments of 255 steps around the colour wheel.
const hues = 6 * 255

// RandomRGB returns a colour with each channel picked at random from 0 to
// 255. Many results are muted or greyish; use RandomVivid for bright ones.
// It is not suitable for anything security related.
func RandomRGB() RGB {
	n := rand.Uint32()
	return RGB{R: uint8(n >> 16), G: uint8(n >> 8), B: uint8(n)}
}

// RandomVivid returns a fully saturated colour at full brightness with a
// random hue: one channel is always 255 and another is always 0.
func RandomVivid() RGB {
	return hue(rand.IntN(hues))
}

// hue returns the fully saturated colour at step h of the wheel, running
// red, yellow, green, cyan, blue, magenta and back towards red. h must be in
// [0, hues).
func hue(h int) RGB {
	x := uint8(h % 255)
	switch h / 255 {
	case 0:
		return RGB{R: 255, G: x}
	case 1:
		return RGB{R: 255 - x, G: 255}
	case 2:
		return RGB{G: 255, B: x}
	case 3:
		return RGB{G: 255 - x, B: 255}
	case 4:
		return RGB{R: x, B: 255}
	default:
		return RGB{R: 255, B: 255 - x}
	}
}

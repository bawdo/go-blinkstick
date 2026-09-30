package blinkstick

import "strings"

// Model describes a BlinkStick product. Models differ only in data,
// including which modes they allow, so supporting another product means
// adding a row to the models table.
type Model struct {
	Name  string // "Nano", "Square" or "unknown"
	LEDs  int    // addressable LEDs on channel 0
	modes uint8  // allowed modes, bit 1<<m set for each Mode m
}

// wsModes allows ModeWS2812 and ModeWS2812Mirror. Modes 0 and 1 are for the
// BlinkStick Pro's RGB outputs, and a Nano set to mode 1 goes dark.
const wsModes = 1<<ModeWS2812 | 1<<ModeWS2812Mirror

// SupportsMode reports whether SetMode accepts x on this model.
func (m Model) SupportsMode(x Mode) bool {
	return x < 8 && m.modes&(1<<x) != 0
}

// Supported models.
var (
	Nano   = Model{Name: "Nano", LEDs: 2, modes: wsModes}
	Square = Model{Name: "Square", LEDs: 8, modes: wsModes}

	unknownModel = Model{Name: "unknown"}
)

type modelKey struct {
	major   string // serial major version, the "3" in "BS072777-3.0"
	release uint16 // USB bcdDevice
}

// models maps hardware identity to a Model. Upstream libraries map
// bcdDevice 0x0201 to the Strip, but a Square (BS073788-3.1) reports 0x0201
// too, so the Strip may share this value.
var models = map[modelKey]Model{
	{major: "3", release: 0x0202}: Nano,
	{major: "3", release: 0x0201}: Square,
}

// Info identifies an attached BlinkStick.
type Info struct {
	Serial       string // for example "BS072777-3.0"
	Version      string // firmware version from the serial, for example "3.0"
	Manufacturer string
	Product      string
	Model        Model
}

// deviceInfo is what the HID layer reports about a device.
type deviceInfo struct {
	serial, manufacturer, product string
	release                       uint16
}

func newInfo(di deviceInfo) Info {
	_, version, _ := strings.Cut(di.serial, "-")
	major, _, _ := strings.Cut(version, ".")
	m, ok := models[modelKey{major: major, release: di.release}]
	if !ok {
		m = unknownModel
	}
	return Info{
		Serial:       di.serial,
		Version:      version,
		Manufacturer: di.manufacturer,
		Product:      di.product,
		Model:        m,
	}
}

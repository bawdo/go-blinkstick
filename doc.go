// Package blinkstick drives BlinkStick USB LED devices. The Nano and Square
// are supported on macOS.
//
// Open a stick, set its LEDs and close it:
//
//	d, err := blinkstick.Open()
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer d.Close()
//	err = d.SetAll(blinkstick.RGB{R: 255})
//
// Use [List] to find every attached stick and [OpenSerial] to open a
// particular one. A stick can also be given a name with [Device.SetName] and
// opened by it with [OpenName]. Several sticks can be open at once, and a [Device] is safe
// for concurrent use.
//
// # Requirements
//
// The package uses cgo and a bundled copy of hidapi. On macOS, install the
// Xcode command line tools. macOS lets only one process open a stick at a
// time.
//
// # EEPROM
//
// [Device.SetInfoBlock], [Device.SetName] and [Device.SetMode] write to EEPROM on the device,
// which wears with use. Setting LEDs does not.
//
// This package is unofficial and not affiliated with Agile Innovative.
package blinkstick

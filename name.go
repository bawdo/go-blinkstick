package blinkstick

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Names live in info block 1, following the convention of the official
// BlinkStick tools. Info block 2 is left free for other data.
const nameBlock = 1

// Name returns the name stored on the stick, or "" if none is set. Another
// tool may have stored bytes that are not valid UTF-8 or that include
// control characters, so sanitise the name before printing it to a terminal.
func (d *Device) Name() (string, error) {
	b, err := d.InfoBlock(nameBlock)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// SetName stores name on the stick, replacing any name already there. name is
// UTF-8 of at most 32 bytes with no NUL characters; "" clears the name.
// Names are not checked for uniqueness, but OpenName fails if two attached
// sticks share one. It writes EEPROM, which wears with use, so do not call it
// in a loop.
func (d *Device) SetName(name string) error {
	if strings.ContainsRune(name, 0) || !utf8.ValidString(name) {
		return fmt.Errorf("%w: %q must be UTF-8 without NUL characters", ErrInvalidName, name)
	}
	if len(name) > infoBlockSize {
		return fmt.Errorf("%w: name is %d bytes, max %d", ErrOutOfRange, len(name), infoBlockSize)
	}
	return d.SetInfoBlock(nameBlock, []byte(name))
}

// NamedInfo is an attached stick and the name stored on it.
type NamedInfo struct {
	Info
	Name string // "" if none is set, or if Busy
	Busy bool   // the stick could not be opened or its name read, for example because another process holds it
}

// ListNamed returns every attached BlinkStick with its name. It opens each
// stick in turn to read the name and closes it again, so it is slower than
// List. Sticks it cannot open or read, including any this process already
// has open, are listed with Busy set.
func ListNamed() ([]NamedInfo, error) {
	dis, err := sys.list()
	if err != nil {
		return nil, err
	}
	named := make([]NamedInfo, len(dis))
	for i, di := range dis {
		named[i].Info = newInfo(di)
		d, name, err := openNamed(di.serial)
		if err != nil {
			named[i].Busy = true
			continue
		}
		d.Close()
		named[i].Name = name
	}
	return named, nil
}

// OpenName opens the BlinkStick whose stored name is exactly name. It opens
// each attached stick in turn to read its name, and skips sticks it cannot
// open or read, including any this process already has open; if none
// matches, the error says why each was skipped. It returns
// ErrDuplicateName if more than one stick has the name.
func OpenName(name string) (*Device, error) {
	if name == "" {
		return nil, fmt.Errorf("%w: empty name", ErrNotFound)
	}
	dis, err := sys.list()
	if err != nil {
		return nil, err
	}
	var matches []*Device
	var skipped []error
	for _, di := range dis {
		d, got, err := openNamed(di.serial)
		if err != nil {
			skipped = append(skipped, err)
			continue
		}
		if got != name {
			d.Close()
			continue
		}
		matches = append(matches, d)
	}
	switch len(matches) {
	case 0:
		return nil, errors.Join(append([]error{fmt.Errorf("%w: name %q", ErrNotFound, name)}, skipped...)...)
	case 1:
		d := matches[0]
		if d.info.Model == unknownModel {
			d.Close()
			return nil, fmt.Errorf("%w: %s is named %q", ErrUnsupported, d.info.Serial, name)
		}
		return d, nil
	}
	serials := make([]string, len(matches))
	for i, d := range matches {
		serials[i] = d.info.Serial
		d.Close()
	}
	return nil, fmt.Errorf("%w: %q is on %s", ErrDuplicateName, name, strings.Join(serials, ", "))
}

// openNamed opens the stick with the given serial and reads its name. The
// caller closes the Device; on error there is nothing to close.
func openNamed(serial string) (*Device, string, error) {
	d, err := openAny(serial)
	if err != nil {
		return nil, "", err
	}
	name, err := d.Name()
	if err != nil {
		d.Close()
		return nil, "", fmt.Errorf("reading name of %s: %w", serial, err)
	}
	return d, name, nil
}

// openAny opens the stick with the given serial whatever its model. Info
// blocks work the same on every model, so names can be read from sticks
// this package cannot otherwise drive.
func openAny(serial string) (*Device, error) {
	t, di, err := sys.open(serial)
	if err != nil {
		return nil, err
	}
	return newReconnectingDevice(t, newInfo(di)), nil
}

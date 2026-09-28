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

// Name returns the name stored on the stick, or "" if none is set.
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
	Busy bool   // the stick could not be opened, for example because another process holds it
}

// ListNamed returns every attached BlinkStick with its name. It opens each
// stick in turn to read the name and closes it again, so it is slower than
// List. Sticks it cannot open, including any this process already has open,
// are listed with Busy set.
func ListNamed() ([]NamedInfo, error) {
	dis, err := sys.list()
	if err != nil {
		return nil, err
	}
	named := make([]NamedInfo, len(dis))
	for i, di := range dis {
		named[i].Info = newInfo(di)
		d, err := openAny(di.serial)
		if err != nil {
			named[i].Busy = true
			continue
		}
		named[i].Name, err = d.Name()
		d.Close()
		if err != nil {
			return nil, fmt.Errorf("blinkstick: reading name of %s: %w", di.serial, err)
		}
	}
	return named, nil
}

// OpenName opens the BlinkStick whose stored name is exactly name. It opens
// each attached stick in turn to read its name, and skips sticks it cannot
// open, including any this process already has open. It returns
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
	var readErrs []error
	for _, di := range dis {
		d, err := openAny(di.serial)
		if err != nil {
			continue
		}
		got, err := d.Name()
		if err != nil {
			readErrs = append(readErrs, fmt.Errorf("reading name of %s: %w", di.serial, err))
		}
		if err != nil || got != name {
			d.Close()
			continue
		}
		matches = append(matches, d)
	}
	switch len(matches) {
	case 0:
		return nil, errors.Join(append([]error{fmt.Errorf("%w: name %q", ErrNotFound, name)}, readErrs...)...)
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

// openAny opens the stick with the given serial whatever its model. Info
// blocks work the same on every model, so names can be read from sticks
// this package cannot otherwise drive.
func openAny(serial string) (*Device, error) {
	t, di, err := sys.open(serial)
	if err != nil {
		return nil, err
	}
	return newDevice(t, newInfo(di)), nil
}

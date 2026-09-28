package blinkstick

import (
	"errors"
	"fmt"
	"sync"

	"github.com/sstallion/go-hid"
)

// hidapi must be initialised before use and finalised after the last device
// closes. Several Devices can be open at once, so Init and Exit are
// reference counted. hidInit and hidExit are swapped in tests.
var (
	hidMu   sync.Mutex
	hidRefs int
	hidInit = hid.Init
	hidExit = hid.Exit
)

// enumMu serialises hidapi calls that walk the device list. On macOS
// hid_enumerate reconfigures a single global IOHIDManager and hid_open calls
// hid_enumerate, so they must not run concurrently.
var enumMu sync.Mutex

func acquireHID() error {
	hidMu.Lock()
	defer hidMu.Unlock()
	if hidRefs == 0 {
		if err := hidInit(); err != nil {
			return fmt.Errorf("blinkstick: hid init: %w", err)
		}
	}
	hidRefs++
	return nil
}

func releaseHID() error {
	hidMu.Lock()
	defer hidMu.Unlock()
	if hidRefs == 0 {
		return nil
	}
	hidRefs--
	if hidRefs == 0 {
		return hidExit()
	}
	return nil
}

type hidBackend struct{}

// list enumerates BlinkSticks only. Filtering by vendor and product ID inside
// hidapi keeps other devices' strings out of Go, which avoids a panic on
// macOS when an unrelated device has a malformed string (go-hid issue #18).
func (hidBackend) list() ([]deviceInfo, error) {
	if err := acquireHID(); err != nil {
		return nil, err
	}
	defer releaseHID()
	enumMu.Lock()
	defer enumMu.Unlock()
	var dis []deviceInfo
	err := hid.Enumerate(vendorID, productID, func(hi *hid.DeviceInfo) error {
		dis = append(dis, fromHID(hi))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("blinkstick: enumerate: %w", err)
	}
	return dis, nil
}

func (hidBackend) open(serial string) (transport, deviceInfo, error) {
	if err := acquireHID(); err != nil {
		return nil, deviceInfo{}, err
	}
	enumMu.Lock()
	var dev *hid.Device
	var err error
	if serial == "" {
		dev, err = hid.OpenFirst(vendorID, productID)
	} else {
		dev, err = hid.Open(vendorID, productID, serial)
	}
	if err != nil {
		enumMu.Unlock()
		releaseHID()
		return nil, deviceInfo{}, fmt.Errorf("%w: %q: %v", ErrNotFound, serial, err)
	}
	hi, err := dev.GetDeviceInfo()
	enumMu.Unlock()
	if err != nil {
		dev.Close()
		releaseHID()
		return nil, deviceInfo{}, fmt.Errorf("blinkstick: device info: %w", err)
	}
	return hidTransport{dev}, fromHID(hi), nil
}

func fromHID(hi *hid.DeviceInfo) deviceInfo {
	return deviceInfo{
		serial:       hi.SerialNbr,
		manufacturer: hi.MfrStr,
		product:      hi.ProductStr,
		release:      hi.ReleaseNbr,
	}
}

// hidTransport releases hidapi along with the device.
type hidTransport struct {
	*hid.Device
}

func (t hidTransport) Close() error {
	return errors.Join(t.Device.Close(), releaseHID())
}

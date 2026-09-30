// Unit tests run against fakeTransport, which keeps reports in memory. They
// never touch a real stick, so writing info blocks here causes no EEPROM
// wear. Automated tests must never write EEPROM on real hardware: see the
// EEPROM RULE in hardware_test.go.

package blinkstick

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"testing"
)

var (
	errBusy = errors.New("general error")
	errGone = errors.New("device unplugged")
)

// fakeTransport behaves like BlinkStick firmware: a get returns the last
// report sent with the same ID, or zeros.
type fakeTransport struct {
	mu      sync.Mutex
	reports map[byte][]byte
	sends   [][]byte
	gets    int  // GetFeatureReport calls, failed ones included
	fail    int  // fail this many transfers before succeeding
	gone    bool // unplugged: every transfer fails
	closed  bool
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{reports: map[byte][]byte{}}
}

func (f *fakeTransport) SendFeatureReport(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sends = append(f.sends, bytes.Clone(p))
	if f.gone {
		return 0, errGone
	}
	if f.fail > 0 {
		f.fail--
		return 0, errBusy
	}
	f.reports[p[0]] = bytes.Clone(p)
	if p[0] == reportLED {
		f.applyLEDLocked(p)
	}
	return len(p), nil
}

// applyLEDLocked stores a report 5 in the frame as the firmware does: RGB on
// the wire becomes GRB in the slot. The channel byte is ignored and slots of
// 8 or more are dropped. f.mu must be held.
func (f *fakeTransport) applyLEDLocked(p []byte) {
	slot := int(p[2])
	if slot >= frameLEDs {
		return
	}
	buf := f.reports[reportFrame]
	if buf == nil {
		buf = make([]byte, frameReportSize)
		buf[0] = reportFrame
		f.reports[reportFrame] = buf
	}
	buf[2+slot*3], buf[3+slot*3], buf[4+slot*3] = p[4], p[3], p[5]
}

func (f *fakeTransport) GetFeatureReport(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.gets++
	if f.gone {
		return 0, errGone
	}
	if f.fail > 0 {
		f.fail--
		return 0, errBusy
	}
	clear(p[1:])
	if stored := f.reports[p[0]]; stored != nil {
		copy(p[1:], stored[1:])
	}
	return len(p), nil
}

func (f *fakeTransport) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed = true
	return nil
}

// frame decodes the last report 6 the fake accepted.
func (f *fakeTransport) frame(n int) []RGB {
	f.mu.Lock()
	defer f.mu.Unlock()
	buf := f.reports[reportFrame]
	if buf == nil {
		buf = make([]byte, frameReportSize)
	}
	return decodeFrame(buf, n)
}

// sentFrames decodes every report 6 send attempt, in order.
func (f *fakeTransport) sentFrames(n int) [][]RGB {
	f.mu.Lock()
	defer f.mu.Unlock()
	var frames [][]RGB
	for _, p := range f.sends {
		if p[0] == reportFrame {
			frames = append(frames, decodeFrame(p, n))
		}
	}
	return frames
}

func (f *fakeTransport) sendCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.sends)
}

func (f *fakeTransport) getCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.gets
}

func openFake(m Model) (*Device, *fakeTransport) {
	ft := newFakeTransport()
	return newDevice(ft, Info{Serial: "BS000001-3.0", Version: "3.0", Model: m}), ft
}

type fakeDevice struct {
	info deviceInfo
	t    *fakeTransport
	busy bool // held by another process, so open fails
}

type fakeBackend struct {
	mu      sync.Mutex
	devices []fakeDevice
}

// unplug makes device i vanish: its transport fails and it cannot be opened.
func (b *fakeBackend) unplug(i int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.devices[i].t.mu.Lock()
	b.devices[i].t.gone = true
	b.devices[i].t.mu.Unlock()
	b.devices[i].busy = true
}

// replug brings device i back as a fresh transport with its LEDs off, as a
// real stick is after losing power. Info blocks survive, as EEPROM does.
func (b *fakeBackend) replug(i int) *fakeTransport {
	b.mu.Lock()
	defer b.mu.Unlock()
	old := b.devices[i].t
	nt := newFakeTransport()
	old.mu.Lock()
	for _, id := range []byte{reportInfo1, reportInfo2} {
		if r := old.reports[id]; r != nil {
			nt.reports[id] = bytes.Clone(r)
		}
	}
	old.mu.Unlock()
	b.devices[i].t = nt
	b.devices[i].busy = false
	return nt
}

func (b *fakeBackend) list() ([]deviceInfo, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	var dis []deviceInfo
	for _, d := range b.devices {
		dis = append(dis, d.info)
	}
	return dis, nil
}

func (b *fakeBackend) open(serial string) (transport, deviceInfo, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, d := range b.devices {
		if serial == "" || d.info.serial == serial {
			if d.busy {
				return nil, deviceInfo{}, fmt.Errorf("%w: %s: busy", ErrNotFound, serial)
			}
			return d.t, d.info, nil
		}
	}
	return nil, deviceInfo{}, fmt.Errorf("%w: %s", ErrNotFound, serial)
}

// useBackend swaps the package backend for the length of the test.
func useBackend(t *testing.T, b backend) {
	orig := sys
	sys = b
	t.Cleanup(func() { sys = orig })
}

var (
	nanoInfo = deviceInfo{serial: "BS072777-3.0", manufacturer: "Agile Innovative Ltd",
		product: "BlinkStick Nano", release: 0x0202}
	squareInfo = deviceInfo{serial: "BS073788-3.1", manufacturer: "Agile Innovative Ltd",
		product: "BlinkStick", release: 0x0201}
	flexInfo = deviceInfo{serial: "BS000002-3.0", manufacturer: "Agile Innovative Ltd",
		product: "BlinkStick Flex", release: 0x0203}
)

// threeSticks returns a backend holding a Nano, a Square and a Flex.
func threeSticks() (*fakeBackend, map[string]*fakeTransport) {
	ts := map[string]*fakeTransport{
		"nano": newFakeTransport(), "square": newFakeTransport(), "flex": newFakeTransport(),
	}
	return &fakeBackend{devices: []fakeDevice{
		{info: nanoInfo, t: ts["nano"]}, {info: squareInfo, t: ts["square"]},
		{info: flexInfo, t: ts["flex"]},
	}}, ts
}

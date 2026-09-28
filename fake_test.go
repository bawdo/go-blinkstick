package blinkstick

import (
	"bytes"
	"errors"
	"fmt"
	"sync"
	"testing"
)

var errBusy = errors.New("general error")

// fakeTransport behaves like BlinkStick firmware: a get returns the last
// report sent with the same ID, or zeros.
type fakeTransport struct {
	mu      sync.Mutex
	reports map[byte][]byte
	sends   [][]byte
	fail    int // fail this many transfers before succeeding
	closed  bool
}

func newFakeTransport() *fakeTransport {
	return &fakeTransport{reports: map[byte][]byte{}}
}

func (f *fakeTransport) SendFeatureReport(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.sends = append(f.sends, bytes.Clone(p))
	if f.fail > 0 {
		f.fail--
		return 0, errBusy
	}
	f.reports[p[0]] = bytes.Clone(p)
	return len(p), nil
}

func (f *fakeTransport) GetFeatureReport(p []byte) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
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

func openFake(m Model) (*Device, *fakeTransport) {
	ft := newFakeTransport()
	return newDevice(ft, Info{Serial: "BS000001-3.0", Version: "3.0", Model: m}), ft
}

type fakeDevice struct {
	info deviceInfo
	t    *fakeTransport
}

type fakeBackend struct {
	devices []fakeDevice
}

func (b *fakeBackend) list() ([]deviceInfo, error) {
	var dis []deviceInfo
	for _, d := range b.devices {
		dis = append(dis, d.info)
	}
	return dis, nil
}

func (b *fakeBackend) open(serial string) (transport, deviceInfo, error) {
	for _, d := range b.devices {
		if serial == "" || d.info.serial == serial {
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
		{nanoInfo, ts["nano"]}, {squareInfo, ts["square"]}, {flexInfo, ts["flex"]},
	}}, ts
}

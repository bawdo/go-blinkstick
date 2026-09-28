package blinkstick

import (
	"bytes"
	"errors"
	"sync"
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

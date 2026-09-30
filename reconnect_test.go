package blinkstick

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// openNano opens the Nano in threeSticks through the backend, so it can
// reconnect.
func openNano(t *testing.T) (*Device, *fakeBackend, *fakeTransport) {
	t.Helper()
	b, ts := threeSticks()
	useBackend(t, b)
	d, err := OpenSerial(nanoInfo.serial)
	if err != nil {
		t.Fatalf("OpenSerial: %v", err)
	}
	return d, b, ts["nano"]
}

func TestReconnectAfterReplug(t *testing.T) {
	d, b, old := openNano(t)
	d.SetAll(RGB{R: 255})
	b.unplug(0)
	nt := b.replug(0)
	if err := d.SetAll(RGB{B: 255}); err != nil {
		t.Fatalf("SetAll after replug: %v", err)
	}
	if !old.closed {
		t.Error("old transport not closed")
	}
	if got := nt.frame(2); !slices.Equal(got, fill(2, RGB{B: 255})) {
		t.Errorf("frame after replug = %v, want blue", got)
	}
}

func TestReconnectRepaintsLastFrame(t *testing.T) {
	d, b, _ := openNano(t)
	d.SetFrame([]RGB{{R: 255}, {G: 255}})
	b.unplug(0)
	nt := b.replug(0)
	got, err := d.Frame()
	if err != nil {
		t.Fatalf("Frame after replug: %v", err)
	}
	if want := []RGB{{R: 255}, {G: 255}}; !slices.Equal(got, want) {
		t.Errorf("Frame = %v, want %v repainted", got, want)
	}
	if got := nt.frame(2); !slices.Equal(got, []RGB{{R: 255}, {G: 255}}) {
		t.Errorf("device frame = %v, want repainted", got)
	}
}

func TestReconnectRepaintsInverted(t *testing.T) {
	d, b, _ := openNano(t)
	d.SetInverse(true)
	d.SetAll(RGB{R: 255})
	b.unplug(0)
	nt := b.replug(0)
	if _, err := d.Frame(); err != nil {
		t.Fatalf("Frame after replug: %v", err)
	}
	if got := nt.frame(2); !slices.Equal(got, fill(2, RGB{G: 255, B: 255})) {
		t.Errorf("device frame = %v, want red inverted", got)
	}
}

func TestReconnectNothingToRepaint(t *testing.T) {
	d, b, _ := openNano(t)
	b.unplug(0)
	nt := b.replug(0)
	if _, err := d.Name(); err != nil {
		t.Fatalf("Name after replug: %v", err)
	}
	if nt.sendCount() != 0 {
		t.Errorf("sends = %d, want 0 (no frame was ever written)", nt.sendCount())
	}
}

func TestDisconnectedFailsFast(t *testing.T) {
	d, b, old := openNano(t)
	b.unplug(0)
	err := d.SetAll(White)
	if !errors.Is(err, ErrDisconnected) || !errors.Is(err, ErrNotFound) {
		t.Fatalf("SetAll while unplugged = %v, want ErrDisconnected wrapping ErrNotFound", err)
	}
	if n := old.sendCount(); n != transferAttempts {
		t.Errorf("sends = %d, want %d", n, transferAttempts)
	}
	if err := d.SetAll(White); !errors.Is(err, ErrDisconnected) {
		t.Errorf("second SetAll = %v, want ErrDisconnected", err)
	}
	nt := b.replug(0)
	if err := d.SetAll(RGB{G: 255}); err != nil {
		t.Fatalf("SetAll after replug: %v", err)
	}
	if got := nt.frame(2); !slices.Equal(got, fill(2, RGB{G: 255})) {
		t.Errorf("frame = %v, want green", got)
	}
}

func TestCloseWhileDisconnected(t *testing.T) {
	d, b, _ := openNano(t)
	b.unplug(0)
	d.SetAll(White)
	if err := d.Close(); err != nil {
		t.Errorf("Close while disconnected = %v, want nil", err)
	}
	b.replug(0)
	if err := d.SetAll(White); !errors.Is(err, ErrClosed) {
		t.Errorf("SetAll after Close = %v, want ErrClosed", err)
	}
	if err := d.Close(); !errors.Is(err, ErrClosed) {
		t.Errorf("second Close = %v, want ErrClosed", err)
	}
}

func TestReconnectKeepsSettings(t *testing.T) {
	d, b, _ := openNano(t)
	d.SetBrightnessLimit(128)
	b.unplug(0)
	nt := b.replug(0)
	if err := d.SetAll(White); err != nil {
		t.Fatalf("SetAll: %v", err)
	}
	if got := nt.frame(2); !slices.Equal(got, fill(2, RGB{128, 128, 128})) {
		t.Errorf("frame = %v, want brightness limit kept", got)
	}
}

func TestDisconnectedErrorText(t *testing.T) {
	d, b, _ := openNano(t)
	b.unplug(0)
	err := d.SetAll(White)
	if want := "blinkstick: device disconnected: BS072777-3.0"; err == nil || err.Error() != want {
		t.Errorf("error = %v, want %q", err, want)
	}
	if !errors.Is(err, ErrDisconnected) || !errors.Is(err, ErrNotFound) {
		t.Errorf("error %v is not ErrDisconnected and ErrNotFound", err)
	}
	if !errors.Is(ErrDisconnected, ErrNotFound) {
		t.Error("ErrDisconnected does not match ErrNotFound")
	}
	if errors.Unwrap(err) == nil {
		t.Error("reopen cause lost")
	}
}

func TestDisconnectedRepaintFailureKeepsCause(t *testing.T) {
	d, b, _ := openNano(t)
	d.SetAll(White)
	b.unplug(0)
	nt := b.replug(0)
	// The repaint fails, then the stick answers. Name is not a frame send,
	// so reconnect repaints before retrying it.
	nt.fail = transferAttempts
	_, err := d.Name()
	if !errors.Is(err, ErrDisconnected) || !strings.Contains(err.Error(), "repaint") {
		t.Errorf("error = %v, want ErrDisconnected mentioning the repaint", err)
	}
}

func TestReconnectRepaintsSetLED(t *testing.T) {
	d, b, _ := openNano(t)
	d.SetFrame([]RGB{{R: 255}, {G: 255}})
	if err := d.SetLED(1, RGB{B: 255}); err != nil {
		t.Fatalf("SetLED: %v", err)
	}
	b.unplug(0)
	nt := b.replug(0)
	if _, err := d.Frame(); err != nil {
		t.Fatalf("Frame after replug: %v", err)
	}
	if got, want := nt.frame(2), []RGB{{R: 255}, {B: 255}}; !slices.Equal(got, want) {
		t.Errorf("device frame = %v, want %v", got, want)
	}
}

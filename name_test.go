// These tests write names through fakeTransport only, never to a real stick.
// See the note at the top of fake_test.go.

package blinkstick

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// nameStick stores name in info block 1 of a fake stick, as SetName would.
func nameStick(ft *fakeTransport, name string) {
	buf := make([]byte, infoReportSize)
	buf[0] = reportInfo1
	copy(buf[1:], name)
	ft.reports[reportInfo1] = buf
}

func TestNameRoundTrip(t *testing.T) {
	d, ft := openFake(Nano)
	if err := d.SetName("office"); err != nil {
		t.Fatalf("SetName: %v", err)
	}
	if ft.sendCount() != 1 {
		t.Fatalf("sends = %d, want 1", ft.sendCount())
	}
	if got := ft.sends[0][0]; got != reportInfo1 {
		t.Errorf("SetName sent report %d, want info block 1 (%d)", got, reportInfo1)
	}
	got, err := d.Name()
	if err != nil {
		t.Fatalf("Name: %v", err)
	}
	if got != "office" {
		t.Errorf("Name = %q, want office", got)
	}
}

func TestNameUnset(t *testing.T) {
	d, _ := openFake(Nano)
	got, err := d.Name()
	if err != nil || got != "" {
		t.Errorf("Name = %q, %v; want empty", got, err)
	}
}

func TestSetNameEmptyClears(t *testing.T) {
	d, _ := openFake(Nano)
	d.SetName("office")
	if err := d.SetName(""); err != nil {
		t.Fatalf("SetName(\"\"): %v", err)
	}
	if got, _ := d.Name(); got != "" {
		t.Errorf("Name = %q, want empty", got)
	}
}

func TestSetNameUTF8(t *testing.T) {
	d, _ := openFake(Nano)
	name := "büro ☕"
	if err := d.SetName(name); err != nil {
		t.Fatalf("SetName: %v", err)
	}
	if got, _ := d.Name(); got != name {
		t.Errorf("Name = %q, want %q", got, name)
	}
}

func TestSetNameLimits(t *testing.T) {
	d, ft := openFake(Nano)
	if err := d.SetName(strings.Repeat("x", 32)); err != nil {
		t.Errorf("SetName(32 bytes) = %v, want nil", err)
	}
	sent := ft.sendCount()
	if err := d.SetName(strings.Repeat("x", 33)); !errors.Is(err, ErrOutOfRange) {
		t.Errorf("SetName(33 bytes) = %v, want ErrOutOfRange", err)
	}
	for _, name := range []string{"a\x00b", "bad \xff utf8"} {
		if err := d.SetName(name); !errors.Is(err, ErrInvalidName) {
			t.Errorf("SetName(%q) = %v, want ErrInvalidName", name, err)
		}
	}
	if ft.sendCount() != sent {
		t.Errorf("rejected names were sent to the device")
	}
}

func TestOpenName(t *testing.T) {
	b, ts := threeSticks()
	nameStick(ts["nano"], "desk")
	nameStick(ts["square"], "shelf")
	useBackend(t, b)
	d, err := OpenName("shelf")
	if err != nil {
		t.Fatalf("OpenName: %v", err)
	}
	if d.Info().Model != Square {
		t.Errorf("OpenName(shelf) opened %v, want the Square", d.Info().Model)
	}
	if ts["square"].closed {
		t.Error("matched stick was closed")
	}
	if !ts["nano"].closed || !ts["flex"].closed {
		t.Error("sticks that did not match were left open")
	}
}

func TestOpenNameIsExact(t *testing.T) {
	b, ts := threeSticks()
	nameStick(ts["square"], "Shelf")
	useBackend(t, b)
	for _, name := range []string{"shelf", "Shel", " Shelf"} {
		if _, err := OpenName(name); !errors.Is(err, ErrNotFound) {
			t.Errorf("OpenName(%q) = %v, want ErrNotFound", name, err)
		}
	}
}

func TestOpenNameEmpty(t *testing.T) {
	b, _ := threeSticks()
	useBackend(t, b)
	if _, err := OpenName(""); !errors.Is(err, ErrNotFound) {
		t.Errorf("OpenName(\"\") = %v, want ErrNotFound", err)
	}
}

func TestOpenNameSkipsBusy(t *testing.T) {
	b, ts := threeSticks()
	nameStick(ts["nano"], "desk")
	nameStick(ts["square"], "desk")
	b.devices[0].busy = true
	useBackend(t, b)
	d, err := OpenName("desk")
	if err != nil {
		t.Fatalf("OpenName: %v", err)
	}
	if d.Info().Model != Square {
		t.Errorf("OpenName opened %v, want the Square", d.Info().Model)
	}
}

func TestOpenNameDuplicate(t *testing.T) {
	b, ts := threeSticks()
	nameStick(ts["nano"], "desk")
	nameStick(ts["square"], "desk")
	useBackend(t, b)
	_, err := OpenName("desk")
	if !errors.Is(err, ErrDuplicateName) {
		t.Fatalf("OpenName = %v, want ErrDuplicateName", err)
	}
	for _, s := range []string{"BS072777-3.0", "BS073788-3.1"} {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("error %q does not name %s", err, s)
		}
	}
	if !ts["nano"].closed || !ts["square"].closed {
		t.Error("duplicate sticks were left open")
	}
}

func TestOpenNameUnsupported(t *testing.T) {
	b, ts := threeSticks()
	nameStick(ts["flex"], "strip")
	useBackend(t, b)
	if _, err := OpenName("strip"); !errors.Is(err, ErrUnsupported) {
		t.Errorf("OpenName(strip) = %v, want ErrUnsupported", err)
	}
	if !ts["flex"].closed {
		t.Error("unsupported stick left open")
	}
}

func TestListNamed(t *testing.T) {
	b, ts := threeSticks()
	nameStick(ts["nano"], "desk")
	nameStick(ts["flex"], "strip")
	b.devices[1].busy = true
	useBackend(t, b)
	got, err := ListNamed()
	if err != nil {
		t.Fatalf("ListNamed: %v", err)
	}
	var rows []string
	for _, n := range got {
		busy := ""
		if n.Busy {
			busy = " busy"
		}
		rows = append(rows, n.Serial+" "+n.Model.Name+" "+n.Name+busy)
	}
	want := []string{
		"BS072777-3.0 Nano desk",
		"BS073788-3.1 Square  busy",
		"BS000002-3.0 unknown strip",
	}
	if !slices.Equal(rows, want) {
		t.Errorf("ListNamed = %q, want %q", rows, want)
	}
	if !ts["nano"].closed || !ts["flex"].closed {
		t.Error("ListNamed left sticks open")
	}
}

package blinkstick

import (
	"errors"
	"slices"
	"testing"
)

func TestList(t *testing.T) {
	b, _ := threeSticks()
	useBackend(t, b)
	infos, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var got []string
	for _, i := range infos {
		got = append(got, i.Serial+" "+i.Model.Name)
	}
	want := []string{"BS072777-3.0 Nano", "BS073788-3.1 Square", "BS000002-3.0 unknown"}
	if !slices.Equal(got, want) {
		t.Errorf("List = %v, want %v", got, want)
	}
}

func TestOpenFirst(t *testing.T) {
	b, _ := threeSticks()
	useBackend(t, b)
	d, err := Open()
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if d.Info().Model != Nano {
		t.Errorf("Open model = %v, want Nano", d.Info().Model)
	}
}

func TestOpenSerial(t *testing.T) {
	b, _ := threeSticks()
	useBackend(t, b)
	d, err := OpenSerial("BS073788-3.1")
	if err != nil {
		t.Fatalf("OpenSerial: %v", err)
	}
	if d.Info().Model != Square || d.Info().Product != "BlinkStick" {
		t.Errorf("Info = %+v", d.Info())
	}
}

func TestOpenSerialNotFound(t *testing.T) {
	b, _ := threeSticks()
	useBackend(t, b)
	for _, s := range []string{"BS999999-3.0", ""} {
		if _, err := OpenSerial(s); !errors.Is(err, ErrNotFound) {
			t.Errorf("OpenSerial(%q) = %v, want ErrNotFound", s, err)
		}
	}
}

func TestOpenNoDevices(t *testing.T) {
	useBackend(t, &fakeBackend{})
	if _, err := Open(); !errors.Is(err, ErrNotFound) {
		t.Errorf("Open = %v, want ErrNotFound", err)
	}
}

func TestOpenUnsupportedClosesTransport(t *testing.T) {
	b, ts := threeSticks()
	useBackend(t, b)
	if _, err := OpenSerial("BS000002-3.0"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("OpenSerial(flex) = %v, want ErrUnsupported", err)
	}
	if !ts["flex"].closed {
		t.Error("flex transport left open")
	}
}

func TestTwoDevicesAreIndependent(t *testing.T) {
	b, ts := threeSticks()
	useBackend(t, b)
	nano, err := OpenSerial("BS072777-3.0")
	if err != nil {
		t.Fatal(err)
	}
	square, err := OpenSerial("BS073788-3.1")
	if err != nil {
		t.Fatal(err)
	}
	nano.SetAll(RGB{R: 255})
	square.SetAll(RGB{G: 255})
	if err := nano.Close(); err != nil {
		t.Fatal(err)
	}
	if err := square.SetAll(RGB{B: 255}); err != nil {
		t.Fatalf("square after closing nano: %v", err)
	}
	if got := ts["nano"].frame(2); !slices.Equal(got, fill(2, RGB{R: 255})) {
		t.Errorf("nano frame = %v", got)
	}
	if got := ts["square"].frame(8); !slices.Equal(got, fill(8, RGB{B: 255})) {
		t.Errorf("square frame = %v", got)
	}
}

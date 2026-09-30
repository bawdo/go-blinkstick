package blinkstick

import (
	"errors"
	"slices"
	"sync"
	"testing"
)

func TestSetAllSendsFullFrame(t *testing.T) {
	d, ft := openFake(Square)
	c := RGB{1, 2, 3}
	if err := d.SetAll(c); err != nil {
		t.Fatalf("SetAll: %v", err)
	}
	if ft.sendCount() != 1 {
		t.Fatalf("sends = %d, want 1", ft.sendCount())
	}
	if got := ft.frame(8); !slices.Equal(got, fill(8, c)) {
		t.Errorf("frame = %v, want all %v", got, c)
	}
}

func TestSetAllNanoLeavesUnusedSlotsZero(t *testing.T) {
	d, ft := openFake(Nano)
	if err := d.SetAll(White); err != nil {
		t.Fatalf("SetAll: %v", err)
	}
	want := append(fill(2, White), fill(6, Off)...)
	if got := ft.frame(8); !slices.Equal(got, want) {
		t.Errorf("frame = %v, want %v", got, want)
	}
}

func TestOff(t *testing.T) {
	d, ft := openFake(Square)
	d.SetAll(White)
	if err := d.Off(); err != nil {
		t.Fatalf("Off: %v", err)
	}
	if got := ft.frame(8); !slices.Equal(got, fill(8, Off)) {
		t.Errorf("frame = %v, want all off", got)
	}
}

func TestSetFrame(t *testing.T) {
	d, ft := openFake(Nano)
	leds := []RGB{{R: 255}, {B: 255}}
	if err := d.SetFrame(leds); err != nil {
		t.Fatalf("SetFrame: %v", err)
	}
	if got := ft.frame(2); !slices.Equal(got, leds) {
		t.Errorf("frame = %v, want %v", got, leds)
	}
}

func TestSetFrameWrongLength(t *testing.T) {
	d, ft := openFake(Nano)
	for _, leds := range [][]RGB{nil, fill(1, White), fill(3, White)} {
		if err := d.SetFrame(leds); !errors.Is(err, ErrOutOfRange) {
			t.Errorf("SetFrame(%d LEDs) = %v, want ErrOutOfRange", len(leds), err)
		}
	}
	if ft.sendCount() != 0 {
		t.Errorf("sends = %d, want 0", ft.sendCount())
	}
}

func TestRetriesTransientFailure(t *testing.T) {
	d, ft := openFake(Square)
	ft.fail = 2
	if err := d.SetAll(White); err != nil {
		t.Fatalf("SetAll: %v", err)
	}
	if ft.sendCount() != 3 {
		t.Errorf("sends = %d, want 3", ft.sendCount())
	}
}

func TestGivesUpAfterThreeAttempts(t *testing.T) {
	d, ft := openFake(Square)
	ft.fail = 5
	err := d.SetAll(White)
	if !errors.Is(err, errBusy) {
		t.Fatalf("err = %v, want wrapping %v", err, errBusy)
	}
	if ft.sendCount() != 3 {
		t.Errorf("sends = %d, want 3", ft.sendCount())
	}
}

func TestBrightnessLimitScalesWrites(t *testing.T) {
	d, ft := openFake(Square)
	d.SetBrightnessLimit(128)
	if err := d.SetAll(White); err != nil {
		t.Fatalf("SetAll: %v", err)
	}
	if got := ft.frame(8); !slices.Equal(got, fill(8, RGB{128, 128, 128})) {
		t.Errorf("frame = %v, want all 128", got)
	}
}

func TestClose(t *testing.T) {
	d, ft := openFake(Square)
	if err := d.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !ft.closed {
		t.Error("transport not closed")
	}
	if err := d.Close(); !errors.Is(err, ErrClosed) {
		t.Errorf("second Close = %v, want ErrClosed", err)
	}
	if err := d.SetAll(White); !errors.Is(err, ErrClosed) {
		t.Errorf("SetAll after Close = %v, want ErrClosed", err)
	}
}

func TestInfo(t *testing.T) {
	d, _ := openFake(Nano)
	if got := d.Info(); got.Model != Nano || got.Serial != "BS000001-3.0" {
		t.Errorf("Info = %+v", got)
	}
}

func TestFrameReadsBack(t *testing.T) {
	d, _ := openFake(Nano)
	leds := []RGB{{R: 10}, {G: 20}}
	d.SetFrame(leds)
	got, err := d.Frame()
	if err != nil {
		t.Fatalf("Frame: %v", err)
	}
	if !slices.Equal(got, leds) {
		t.Errorf("Frame = %v, want %v", got, leds)
	}
}

func TestLED(t *testing.T) {
	d, _ := openFake(Nano)
	d.SetFrame([]RGB{{R: 10}, {G: 20}})
	got, err := d.LED(1)
	if err != nil {
		t.Fatalf("LED: %v", err)
	}
	if got != (RGB{G: 20}) {
		t.Errorf("LED(1) = %v, want {0 20 0}", got)
	}
}

func TestSetLEDKeepsOthers(t *testing.T) {
	d, ft := openFake(Square)
	d.SetAll(RGB{B: 50})
	if err := d.SetLED(3, RGB{R: 255}); err != nil {
		t.Fatalf("SetLED: %v", err)
	}
	want := fill(8, RGB{B: 50})
	want[3] = RGB{R: 255}
	if got := ft.frame(8); !slices.Equal(got, want) {
		t.Errorf("frame = %v, want %v", got, want)
	}
}

func TestSetLEDDoesNotRescaleOthers(t *testing.T) {
	d, ft := openFake(Square)
	d.SetBrightnessLimit(128)
	d.SetAll(White)
	if err := d.SetLED(0, White); err != nil {
		t.Fatalf("SetLED: %v", err)
	}
	if got := ft.frame(8); !slices.Equal(got, fill(8, RGB{128, 128, 128})) {
		t.Errorf("frame = %v, want all 128", got)
	}
}

func TestIndexOutOfRange(t *testing.T) {
	d, ft := openFake(Nano)
	for _, i := range []int{-1, 2, 8} {
		if _, err := d.LED(i); !errors.Is(err, ErrOutOfRange) {
			t.Errorf("LED(%d) = %v, want ErrOutOfRange", i, err)
		}
		if err := d.SetLED(i, White); !errors.Is(err, ErrOutOfRange) {
			t.Errorf("SetLED(%d) = %v, want ErrOutOfRange", i, err)
		}
	}
	if ft.sendCount() != 0 {
		t.Errorf("sends = %d, want 0", ft.sendCount())
	}
}

func TestSetLEDConcurrent(t *testing.T) {
	d, ft := openFake(Square)
	var wg sync.WaitGroup
	for i := range Square.LEDs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := d.SetLED(i, White); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if got := ft.frame(8); !slices.Equal(got, fill(8, White)) {
		t.Errorf("frame = %v, want all white (a lost update means SetLED is not atomic)", got)
	}
}

func TestSetLEDSendsOneReport5(t *testing.T) {
	d, ft := openFake(Square)
	d.SetAll(RGB{B: 50})
	sends, gets := ft.sendCount(), ft.getCount()
	if err := d.SetLED(3, RGB{R: 255}); err != nil {
		t.Fatalf("SetLED: %v", err)
	}
	if got := ft.sendCount() - sends; got != 1 {
		t.Fatalf("sends = %d, want 1", got)
	}
	if got, want := ft.sends[sends], []byte{5, 0, 3, 255, 0, 0}; !slices.Equal(got, want) {
		t.Errorf("sent % x, want % x", got, want)
	}
	if got := ft.getCount() - gets; got != 0 {
		t.Errorf("gets = %d, want 0", got)
	}
	want := fill(8, RGB{B: 50})
	want[3] = RGB{R: 255}
	if got := ft.frame(8); !slices.Equal(got, want) {
		t.Errorf("frame = %v, want %v", got, want)
	}
}

func TestSetLEDSeedsLastFrameOnce(t *testing.T) {
	d, ft := openFake(Nano)
	ft.SendFeatureReport(encodeFrame([]RGB{{G: 9}, {B: 9}}))
	if err := d.SetLED(0, RGB{R: 1}); err != nil {
		t.Fatalf("SetLED: %v", err)
	}
	if got := ft.getCount(); got != 1 {
		t.Errorf("gets = %d, want 1", got)
	}
	if want := []RGB{{R: 1}, {B: 9}}; !slices.Equal(d.last, want) {
		t.Errorf("last = %v, want %v", d.last, want)
	}
	if err := d.SetLED(1, RGB{R: 2}); err != nil {
		t.Fatalf("SetLED: %v", err)
	}
	if got := ft.getCount(); got != 1 {
		t.Errorf("gets = %d, want still 1", got)
	}
	if want := []RGB{{R: 1}, {R: 2}}; !slices.Equal(d.last, want) {
		t.Errorf("last = %v, want %v", d.last, want)
	}
}

func TestSetLEDBrightnessAndInverse(t *testing.T) {
	d, ft := openFake(Nano)
	d.SetAll(Off)
	d.SetBrightnessLimit(128)
	d.SetInverse(true)
	if err := d.SetLED(0, White); err != nil {
		t.Fatalf("SetLED: %v", err)
	}
	if got, want := ft.sends[len(ft.sends)-1], []byte{5, 0, 0, 127, 127, 127}; !slices.Equal(got, want) {
		t.Errorf("sent % x, want % x", got, want)
	}
	got, err := d.LED(0)
	if err != nil {
		t.Fatalf("LED: %v", err)
	}
	if want := (RGB{128, 128, 128}); got != want {
		t.Errorf("LED(0) = %v, want %v", got, want)
	}
}

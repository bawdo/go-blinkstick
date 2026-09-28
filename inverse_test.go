package blinkstick

import (
	"context"
	"slices"
	"testing"
	"time"
)

func TestRGBInverse(t *testing.T) {
	tests := []struct{ c, want RGB }{
		{Off, White},
		{White, Off},
		{RGB{0, 128, 255}, RGB{255, 127, 0}},
	}
	for _, tt := range tests {
		if got := tt.c.Inverse(); got != tt.want {
			t.Errorf("%v.Inverse() = %v, want %v", tt.c, got, tt.want)
		}
	}
}

func TestInverseOffByDefault(t *testing.T) {
	d, ft := openFake(Nano)
	d.SetAll(RGB{R: 255})
	if got := ft.frame(2); !slices.Equal(got, fill(2, RGB{R: 255})) {
		t.Errorf("device frame = %v, want red", got)
	}
}

func TestInverseFlipsWrites(t *testing.T) {
	d, ft := openFake(Nano)
	d.SetInverse(true)
	if err := d.SetAll(RGB{R: 255}); err != nil {
		t.Fatalf("SetAll: %v", err)
	}
	want := append(fill(2, RGB{G: 255, B: 255}), fill(6, Off)...)
	if got := ft.frame(8); !slices.Equal(got, want) {
		t.Errorf("device frame = %v, want %v (unused slots stay zero)", got, want)
	}
}

func TestInverseReadsBackWhatWasWritten(t *testing.T) {
	d, _ := openFake(Nano)
	d.SetInverse(true)
	leds := []RGB{{R: 10}, {G: 20}}
	d.SetFrame(leds)
	got, err := d.Frame()
	if err != nil {
		t.Fatalf("Frame: %v", err)
	}
	if !slices.Equal(got, leds) {
		t.Errorf("Frame = %v, want %v", got, leds)
	}
	if c, _ := d.LED(1); c != (RGB{G: 20}) {
		t.Errorf("LED(1) = %v, want {0 20 0}", c)
	}
}

func TestInverseAppliesAfterBrightnessLimit(t *testing.T) {
	d, ft := openFake(Nano)
	d.SetBrightnessLimit(128)
	d.SetInverse(true)
	d.SetAll(White)
	if got := ft.frame(2); !slices.Equal(got, fill(2, RGB{127, 127, 127})) {
		t.Errorf("device frame = %v, want all 127 (255 - 128)", got)
	}
}

func TestInverseSetLEDKeepsOthers(t *testing.T) {
	d, ft := openFake(Square)
	d.SetInverse(true)
	d.SetAll(RGB{B: 50})
	if err := d.SetLED(3, RGB{R: 255}); err != nil {
		t.Fatalf("SetLED: %v", err)
	}
	want := fill(8, RGB{255, 255, 205})
	want[3] = RGB{G: 255, B: 255}
	if got := ft.frame(8); !slices.Equal(got, want) {
		t.Errorf("device frame = %v, want %v", got, want)
	}
}

func TestInverseMorph(t *testing.T) {
	stubSleep(t)
	d, ft := openFake(Nano)
	d.SetInverse(true)
	d.Off()
	to := RGB{R: 100, G: 200}
	if err := d.Morph(context.Background(), to, 100*time.Millisecond); err != nil {
		t.Fatalf("Morph: %v", err)
	}
	frames := ft.sentFrames(2)
	if got := frames[1][0]; got != (RGB{R: 20, G: 40}).Inverse() {
		t.Errorf("first step on device = %v, want inverse of {20 40 0}", got)
	}
	if got := ft.frame(2); !slices.Equal(got, fill(2, to.Inverse())) {
		t.Errorf("device frame = %v, want all %v", got, to.Inverse())
	}
}

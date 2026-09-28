package blinkstick

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
)

// stubSleep records sleeps without waiting.
func stubSleep(t *testing.T) *[]time.Duration {
	orig := sleep
	var slept []time.Duration
	sleep = func(ctx context.Context, d time.Duration) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		slept = append(slept, d)
		return nil
	}
	t.Cleanup(func() { sleep = orig })
	return &slept
}

func TestBlink(t *testing.T) {
	slept := stubSleep(t)
	d, ft := openFake(Square)
	if err := d.Blink(context.Background(), White, 200*time.Millisecond, 2); err != nil {
		t.Fatalf("Blink: %v", err)
	}
	want := [][]RGB{fill(8, White), fill(8, Off), fill(8, White), fill(8, Off)}
	if got := ft.sentFrames(8); !slices.EqualFunc(got, want, slices.Equal) {
		t.Errorf("frames = %v, want %v", got, want)
	}
	if want := slices.Repeat([]time.Duration{100 * time.Millisecond}, 4); !slices.Equal(*slept, want) {
		t.Errorf("slept = %v, want %v", *slept, want)
	}
}

func TestMorph(t *testing.T) {
	slept := stubSleep(t)
	d, ft := openFake(Nano)
	to := RGB{R: 100, G: 200}
	if err := d.Morph(context.Background(), to, 100*time.Millisecond); err != nil {
		t.Fatalf("Morph: %v", err)
	}
	frames := ft.sentFrames(2)
	if len(frames) != 5 {
		t.Fatalf("frames = %d, want 5 (100ms in 20ms steps)", len(frames))
	}
	if got := frames[0][0]; got != (RGB{R: 20, G: 40}) {
		t.Errorf("first step = %v, want {20 40 0}", got)
	}
	if got := frames[4]; !slices.Equal(got, fill(2, to)) {
		t.Errorf("last frame = %v, want all %v", got, to)
	}
	if len(*slept) != 4 {
		t.Errorf("sleeps = %d, want 4", len(*slept))
	}
}

func TestMorphStartsFromCurrentFrame(t *testing.T) {
	stubSleep(t)
	d, ft := openFake(Nano)
	d.SetFrame([]RGB{{R: 200}, {B: 100}})
	if err := d.Morph(context.Background(), Off, 40*time.Millisecond); err != nil {
		t.Fatalf("Morph: %v", err)
	}
	frames := ft.sentFrames(2)
	if got, want := frames[1], []RGB{{R: 100}, {B: 50}}; !slices.Equal(got, want) {
		t.Errorf("midpoint = %v, want %v", got, want)
	}
	if got := ft.frame(2); !slices.Equal(got, fill(2, Off)) {
		t.Errorf("end = %v, want off", got)
	}
}

func TestMorphZeroDurationJumps(t *testing.T) {
	stubSleep(t)
	d, ft := openFake(Nano)
	if err := d.Morph(context.Background(), White, 0); err != nil {
		t.Fatalf("Morph: %v", err)
	}
	if got := ft.sentFrames(2); len(got) != 1 || !slices.Equal(got[0], fill(2, White)) {
		t.Errorf("frames = %v, want one white frame", got)
	}
}

func TestMorphHonoursBrightnessLimit(t *testing.T) {
	stubSleep(t)
	d, ft := openFake(Nano)
	d.SetBrightnessLimit(128)
	d.Morph(context.Background(), White, 0)
	if got := ft.frame(2); !slices.Equal(got, fill(2, RGB{128, 128, 128})) {
		t.Errorf("frame = %v, want all 128", got)
	}
}

func TestPulse(t *testing.T) {
	stubSleep(t)
	d, ft := openFake(Nano)
	c := RGB{G: 200}
	if err := d.Pulse(context.Background(), c, 80*time.Millisecond, 1); err != nil {
		t.Fatalf("Pulse: %v", err)
	}
	// Off, two steps up, two steps down.
	frames := ft.sentFrames(2)
	if len(frames) != 5 {
		t.Fatalf("frames = %d, want 5", len(frames))
	}
	if !slices.Equal(frames[0], fill(2, Off)) {
		t.Errorf("first = %v, want off", frames[0])
	}
	if !slices.Equal(frames[2], fill(2, c)) {
		t.Errorf("peak = %v, want %v", frames[2], c)
	}
	if !slices.Equal(frames[4], fill(2, Off)) {
		t.Errorf("last = %v, want off", frames[4])
	}
}

func TestEffectsCancelled(t *testing.T) {
	stubSleep(t)
	d, ft := openFake(Nano)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := d.Blink(ctx, White, time.Second, 1); !errors.Is(err, context.Canceled) {
		t.Errorf("Blink = %v, want context.Canceled", err)
	}
	if err := d.Pulse(ctx, White, time.Second, 1); !errors.Is(err, context.Canceled) {
		t.Errorf("Pulse = %v, want context.Canceled", err)
	}
	if err := d.Morph(ctx, White, time.Second); !errors.Is(err, context.Canceled) {
		t.Errorf("Morph = %v, want context.Canceled", err)
	}
	if ft.sendCount() != 0 {
		t.Errorf("sends = %d, want 0 for an already cancelled context", ft.sendCount())
	}
}

func TestEffectsRejectBadArguments(t *testing.T) {
	stubSleep(t)
	d, _ := openFake(Nano)
	ctx := context.Background()
	checks := map[string]error{
		"Blink repeats 0":  d.Blink(ctx, White, time.Second, 0),
		"Blink period 0":   d.Blink(ctx, White, 0, 1),
		"Pulse repeats 0":  d.Pulse(ctx, White, time.Second, 0),
		"Pulse duration 0": d.Pulse(ctx, White, 0, 1),
		"Morph negative":   d.Morph(ctx, White, -time.Second),
	}
	for name, err := range checks {
		if !errors.Is(err, ErrOutOfRange) {
			t.Errorf("%s = %v, want ErrOutOfRange", name, err)
		}
	}
}

func TestSleepHonoursCancel(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	start := time.Now()
	if err := sleep(ctx, time.Hour); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("sleep = %v, want DeadlineExceeded", err)
	}
	if time.Since(start) > time.Second {
		t.Error("sleep ignored cancellation")
	}
}

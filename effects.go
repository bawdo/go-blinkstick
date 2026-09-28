package blinkstick

import (
	"context"
	"fmt"
	"time"
)

// effectStep is the time between frames in an effect.
const effectStep = 20 * time.Millisecond

// sleep waits for d, or until ctx is done. Tests replace it.
var sleep = func(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// Blink turns every LED to c for period/2, then off for period/2, repeats
// times. It ends with the LEDs off. If ctx is cancelled the LEDs are left as
// they are.
func (d *Device) Blink(ctx context.Context, c RGB, period time.Duration, repeats int) error {
	if err := checkEffect(period, repeats); err != nil {
		return err
	}
	for range repeats {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := d.SetAll(c); err != nil {
			return err
		}
		if err := sleep(ctx, period/2); err != nil {
			return err
		}
		if err := d.Off(); err != nil {
			return err
		}
		if err := sleep(ctx, period/2); err != nil {
			return err
		}
	}
	return nil
}

// Pulse fades every LED from off up to c over duration/2 and back down over
// duration/2, repeats times. It ends with the LEDs off. If ctx is cancelled
// the LEDs are left as they are.
func (d *Device) Pulse(ctx context.Context, c RGB, duration time.Duration, repeats int) error {
	if err := checkEffect(duration, repeats); err != nil {
		return err
	}
	for range repeats {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := d.Off(); err != nil {
			return err
		}
		if err := d.Morph(ctx, c, duration/2); err != nil {
			return err
		}
		if err := d.Morph(ctx, Off, duration/2); err != nil {
			return err
		}
	}
	return nil
}

// Morph fades every LED from its current value to `to` over duration, in
// 20 ms steps. A duration under one step jumps straight to `to`. It ends with
// every LED at `to`. If ctx is cancelled the LEDs are left as they are.
func (d *Device) Morph(ctx context.Context, to RGB, duration time.Duration) error {
	if duration < 0 {
		return fmt.Errorf("%w: duration %v", ErrOutOfRange, duration)
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	from, err := d.Frame()
	if err != nil {
		return err
	}
	d.mu.Lock()
	target := to.scale(d.limit)
	d.mu.Unlock()

	steps := max(1, int(duration/effectStep))
	leds := make([]RGB, len(from))
	for s := 1; s <= steps; s++ {
		for i, f := range from {
			leds[i] = lerp(f, target, s, steps)
		}
		d.mu.Lock()
		err := d.writeFrameLocked(leds)
		d.mu.Unlock()
		if err != nil {
			return err
		}
		if s < steps {
			if err := sleep(ctx, effectStep); err != nil {
				return err
			}
		}
	}
	return nil
}

func checkEffect(d time.Duration, repeats int) error {
	if d <= 0 || repeats < 1 {
		return fmt.Errorf("%w: duration %v, repeats %d", ErrOutOfRange, d, repeats)
	}
	return nil
}

// lerp returns the point s/n of the way from a to b.
func lerp(a, b RGB, s, n int) RGB {
	f := func(x, y uint8) uint8 { return uint8(int(x) + (int(y)-int(x))*s/n) }
	return RGB{R: f(a.R, b.R), G: f(a.G, b.G), B: f(a.B, b.B)}
}

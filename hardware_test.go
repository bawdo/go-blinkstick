//go:build hardware

// Hardware tests need a BlinkStick Nano and a Square attached, and someone
// watching the LEDs. Run them with: make test-hardware
//
// EEPROM RULE: these tests must never call SetInfoBlock, SetName or anything
// else that writes EEPROM, because EEPROM wears with use. Reading info blocks
// and names is fine. LED values live in RAM and cause no wear. Set a name by
// hand if you want the name tests to do more than read.

package blinkstick

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

// hold returns how long a step on d stays visible. The Square has four times
// as many LEDs as the Nano, so it is held twice as long.
func hold(d *Device) time.Duration {
	if d.Info().Model == Square {
		return 8 * time.Second
	}
	return 4 * time.Second
}

// pace scales effect durations: 1 for the Nano, 2 for the Square.
func pace(d *Device) time.Duration {
	return hold(d) / (4 * time.Second)
}

// palette holds saturated, easily told apart values, in LED order.
var palette = []RGB{
	{R: 255},         // red
	{B: 255},         // blue
	{G: 255},         // green
	{R: 255, G: 255}, // yellow
	{G: 255, B: 255}, // cyan
	{R: 255, B: 255}, // magenta
	White,
	{R: 255, G: 80}, // orange
}

// paletteNames names each entry of palette, in the same order.
var paletteNames = []string{"red", "blue", "green", "yellow", "cyan", "magenta", "white", "orange"}

// openBoth opens the attached Nano and Square and turns them off and closes
// them when the test ends.
func openBoth(t *testing.T) (nano, square *Device) {
	t.Helper()
	infos, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, info := range infos {
		d, err := OpenSerial(info.Serial)
		if err != nil {
			t.Fatalf("OpenSerial(%s): %v", info.Serial, err)
		}
		t.Cleanup(func() {
			d.Off()
			d.Close()
		})
		switch info.Model {
		case Nano:
			nano = d
		case Square:
			square = d
		}
	}
	if nano == nil || square == nil {
		t.Fatalf("need a Nano and a Square attached, found %+v", infos)
	}
	return nano, square
}

func eachStick(t *testing.T, f func(t *testing.T, d *Device)) {
	nano, square := openBoth(t)
	for _, d := range []*Device{nano, square} {
		t.Run(d.Info().Model.Name, func(t *testing.T) { f(t, d) })
	}
}

func TestHardwareIdentity(t *testing.T) {
	eachStick(t, func(t *testing.T, d *Device) {
		info := d.Info()
		t.Logf("%+v", info)
		if !strings.HasPrefix(info.Serial, "BS") || info.Version == "" {
			t.Errorf("serial %q, version %q", info.Serial, info.Version)
		}
		if info.Manufacturer != "Agile Innovative Ltd" {
			t.Errorf("manufacturer %q", info.Manufacturer)
		}
	})
}

func TestHardwareFrameRoundTrip(t *testing.T) {
	eachStick(t, func(t *testing.T, d *Device) {
		n := d.Info().Model.LEDs
		leds := palette[:n]
		t.Logf("watch: %s in LED order", strings.Join(paletteNames[:n], ", "))
		if err := d.SetFrame(leds); err != nil {
			t.Fatalf("SetFrame: %v", err)
		}
		time.Sleep(hold(d))
		got, err := d.Frame()
		if err != nil {
			t.Fatalf("Frame: %v", err)
		}
		if !slices.Equal(got, leds) {
			t.Errorf("Frame = %v, want %v", got, leds)
		}
	})
}

func TestHardwareSetLEDAndLED(t *testing.T) {
	eachStick(t, func(t *testing.T, d *Device) {
		last := d.Info().Model.LEDs - 1
		red := RGB{R: 255}
		d.Off()
		if err := d.SetLED(last, red); err != nil {
			t.Fatalf("SetLED: %v", err)
		}
		t.Logf("watch: only LED %d lit, red", last)
		time.Sleep(hold(d))
		got, err := d.LED(last)
		if err != nil || got != red {
			t.Errorf("LED(%d) = %v, %v; want %v", last, got, err, red)
		}
		if first, _ := d.LED(0); first != Off {
			t.Errorf("LED(0) = %v, want off", first)
		}
	})
}

func TestHardwareBrightnessLimit(t *testing.T) {
	eachStick(t, func(t *testing.T, d *Device) {
		d.SetBrightnessLimit(64)
		defer d.SetBrightnessLimit(255)
		if err := d.SetAll(White); err != nil {
			t.Fatalf("SetAll: %v", err)
		}
		t.Log("watch: all LEDs dim white, brightness limited to 64")
		time.Sleep(hold(d))
		got, _ := d.Frame()
		if want := fill(d.Info().Model.LEDs, RGB{64, 64, 64}); !slices.Equal(got, want) {
			t.Errorf("Frame = %v, want %v", got, want)
		}
	})
}

func TestHardwareEffects(t *testing.T) {
	eachStick(t, func(t *testing.T, d *Device) {
		ctx := context.Background()
		p := pace(d)
		t.Logf("watch: two white blinks (%v period), one green pulse (%v), then a fade to blue (%v), then blue held",
			800*time.Millisecond*p, 2*time.Second*p, 2*time.Second*p)
		if err := d.Blink(ctx, White, 800*time.Millisecond*p, 2); err != nil {
			t.Fatalf("Blink: %v", err)
		}
		if err := d.Pulse(ctx, RGB{G: 255}, 2*time.Second*p, 1); err != nil {
			t.Fatalf("Pulse: %v", err)
		}
		if err := d.Morph(ctx, RGB{B: 255}, 2*time.Second*p); err != nil {
			t.Fatalf("Morph: %v", err)
		}
		got, _ := d.Frame()
		if want := fill(d.Info().Model.LEDs, RGB{B: 255}); !slices.Equal(got, want) {
			t.Errorf("after Morph Frame = %v, want %v", got, want)
		}
		time.Sleep(hold(d))
	})
}

func TestHardwareInfoBlocksReadOnly(t *testing.T) {
	eachStick(t, func(t *testing.T, d *Device) {
		for n := 1; n <= 2; n++ {
			b, err := d.InfoBlock(n)
			if err != nil {
				t.Fatalf("InfoBlock(%d): %v", n, err)
			}
			t.Logf("info block %d: %q", n, b)
		}
	})
}

func TestHardwareMultiDevice(t *testing.T) {
	nano, square := openBoth(t)
	var wg sync.WaitGroup
	for _, d := range []*Device{nano, square} {
		wg.Go(func() {
			p := pace(d)
			for i := range 20 {
				c := RGB{R: 255}
				if i%2 == 1 {
					c = RGB{B: 255}
				}
				if err := d.SetAll(c); err != nil {
					t.Errorf("%s SetAll: %v", d.Info().Model.Name, err)
					return
				}
				time.Sleep(100 * time.Millisecond * p)
			}
		})
	}
	wg.Wait()

	if err := nano.Off(); err != nil {
		t.Errorf("Nano Off: %v", err)
	}
	if err := nano.Close(); err != nil {
		t.Fatalf("close Nano: %v", err)
	}
	green := RGB{G: 255}
	if err := square.SetAll(green); err != nil {
		t.Fatalf("Square after closing Nano: %v", err)
	}
	time.Sleep(hold(square))
	if got, _ := square.Frame(); !slices.Equal(got, fill(8, green)) {
		t.Errorf("Square Frame = %v, want all green", got)
	}
}

// TestHardwareReport10Probe reads the Square's undocumented report 10. It
// never writes to it.
func TestHardwareReport10Probe(t *testing.T) {
	_, square := openBoth(t)
	buf := make([]byte, 3)
	buf[0] = 10
	square.mu.Lock()
	err := square.getLocked(buf)
	square.mu.Unlock()
	t.Logf("report 10: err=%v bytes=% x", err, buf)
}

// TestHardwareConcurrentOpen lists and opens both sticks from several
// goroutines at once. Run with -race. It writes nothing to the devices.
func TestHardwareConcurrentOpen(t *testing.T) {
	infos, err := List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var wg sync.WaitGroup
	for range 8 {
		for _, info := range infos {
			wg.Go(func() {
				if _, err := List(); err != nil {
					t.Errorf("List: %v", err)
				}
				d, err := OpenSerial(info.Serial)
				if err != nil {
					// macOS opens devices exclusively, so a concurrent open of
					// the same stick may fail. Only a panic or race is a failure.
					return
				}
				d.Close()
			})
		}
	}
	wg.Wait()
}

// TestHardwareName reads each stick's name. It never writes one.
func TestHardwareName(t *testing.T) {
	eachStick(t, func(t *testing.T, d *Device) {
		name, err := d.Name()
		if err != nil {
			t.Fatalf("Name: %v", err)
		}
		t.Logf("name %q", name)
		if !utf8.ValidString(name) {
			t.Errorf("name %q is not valid UTF-8", name)
		}
	})
}

// TestHardwareListNamedAndOpenName lists names with no stick open, then opens
// each named stick by its name. It never writes a name.
func TestHardwareListNamedAndOpenName(t *testing.T) {
	named, err := ListNamed()
	if err != nil {
		t.Fatalf("ListNamed: %v", err)
	}
	count := map[string]int{}
	for _, n := range named {
		t.Logf("%s %s name %q busy %v", n.Serial, n.Model.Name, n.Name, n.Busy)
		if n.Busy {
			t.Errorf("%s busy with nothing else holding it", n.Serial)
		}
		count[n.Name]++
	}
	for _, n := range named {
		if n.Name == "" || count[n.Name] > 1 || n.Model.Name == "unknown" {
			continue
		}
		d, err := OpenName(n.Name)
		if err != nil {
			t.Errorf("OpenName(%q): %v", n.Name, err)
			continue
		}
		if d.Info().Serial != n.Serial {
			t.Errorf("OpenName(%q) opened %s, want %s", n.Name, d.Info().Serial, n.Serial)
		}
		d.Close()
	}
	if len(count) == 1 && count[""] > 0 {
		t.Log("no stick has a name, so OpenName was not exercised; set one by hand to cover it")
	}
}

// TestHardwareReconnect needs someone to unplug and replug the Nano, so it
// only runs with BLINKSTICK_UNPLUG=1: make test-reconnect. It writes LEDs
// only.
func TestHardwareReconnect(t *testing.T) {
	if os.Getenv("BLINKSTICK_UNPLUG") != "1" {
		t.Skip("set BLINKSTICK_UNPLUG=1 to run; needs the Nano unplugged and replugged by hand")
	}
	nano, _ := openBoth(t)
	orange := []RGB{{R: 255, G: 80}, {R: 255, G: 80}}
	if err := nano.SetFrame(orange); err != nil {
		t.Fatalf("SetFrame: %v", err)
	}

	t.Log("ACTION: unplug the Nano now (60 seconds)")
	var gone error
	for deadline := time.Now().Add(60 * time.Second); time.Now().Before(deadline); {
		if _, err := nano.Frame(); err != nil {
			gone = err
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if gone == nil {
		t.Fatal("no error seen while unplugged")
	}
	t.Logf("while unplugged: %v", gone)
	if !errors.Is(gone, ErrDisconnected) {
		t.Errorf("error while unplugged = %v, want ErrDisconnected", gone)
	}

	t.Log("ACTION: plug the Nano back in (60 seconds)")
	var got []RGB
	var err error
	for deadline := time.Now().Add(60 * time.Second); time.Now().Before(deadline); {
		if got, err = nano.Frame(); err == nil {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("never reconnected: %v", err)
	}
	t.Log("watch: the Nano should be orange again")
	time.Sleep(hold(nano))
	if !slices.Equal(got, orange) {
		t.Errorf("Frame after replug = %v, want %v repainted", got, orange)
	}
}

func TestHardwareInverse(t *testing.T) {
	eachStick(t, func(t *testing.T, d *Device) {
		d.SetInverse(true)
		defer d.SetInverse(false)
		red := RGB{R: 255}
		if err := d.SetAll(red); err != nil {
			t.Fatalf("SetAll: %v", err)
		}
		t.Log("watch: all LEDs cyan (red inverted)")
		time.Sleep(hold(d))
		got, _ := d.Frame()
		if want := fill(d.Info().Model.LEDs, red); !slices.Equal(got, want) {
			t.Errorf("Frame = %v, want %v flipped back", got, want)
		}
	})
}

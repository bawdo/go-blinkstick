package blinkstick

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"
)

// Fuzz targets cover the code that handles input from outside the program:
// colour strings from users, and serials and info blocks from the device.

func FuzzParseRGB(f *testing.F) {
	for _, s := range []string{"#ff8800", "#f80", "f80", "255,136,0", " 1, 2 ,3 ", "CornflowerBlue", "", "#", ",,", "256,0,0", "#ffff"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		c, err := ParseRGB(s)
		if err != nil {
			return
		}
		got, err := ParseRGB(c.Hex())
		if err != nil || got != c {
			t.Errorf("ParseRGB(%q) = %v, but its Hex %q parses to %v, %v", s, c, c.Hex(), got, err)
		}
	})
}

func FuzzNewInfo(f *testing.F) {
	f.Add("BS072777-3.0", uint16(0x0202))
	f.Add("BS073788-3.1", uint16(0x0201))
	f.Add("garbage", uint16(0))
	f.Add("-", uint16(0xffff))
	f.Fuzz(func(t *testing.T, serial string, release uint16) {
		got := newInfo(deviceInfo{serial: serial, release: release})
		if got.Serial != serial {
			t.Errorf("Serial = %q, want %q", got.Serial, serial)
		}
		if !strings.HasSuffix(serial, got.Version) {
			t.Errorf("Version %q is not a suffix of serial %q", got.Version, serial)
		}
	})
}

func FuzzInfoBlockRoundTrip(f *testing.F) {
	f.Add(1, []byte("kitchen"))
	f.Add(2, []byte{})
	f.Add(2, bytes.Repeat([]byte{0xff}, infoBlockSize))
	f.Add(1, []byte("a\x00b\x00"))
	f.Add(3, []byte("x"))
	f.Fuzz(func(t *testing.T, n int, data []byte) {
		d, _ := openFake(Nano)
		if err := d.SetInfoBlock(n, data); err != nil {
			if (n == 1 || n == 2) && len(data) <= infoBlockSize {
				t.Fatalf("SetInfoBlock(%d, % x): %v", n, data, err)
			}
			return
		}
		got, err := d.InfoBlock(n)
		if err != nil {
			t.Fatalf("InfoBlock(%d): %v", n, err)
		}
		if want := bytes.TrimRight(data, "\x00"); !bytes.Equal(got, want) {
			t.Errorf("InfoBlock(%d) = % x, want % x", n, got, want)
		}
	})
}

func FuzzNameRoundTrip(f *testing.F) {
	for _, s := range []string{"kitchen", "", "日本語", "a\x00b", "\xff", strings.Repeat("x", infoBlockSize+1)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, name string) {
		d, _ := openFake(Nano)
		err := d.SetName(name)
		valid := utf8.ValidString(name) && !strings.ContainsRune(name, 0) && len(name) <= infoBlockSize
		if err != nil {
			if valid {
				t.Fatalf("SetName(%q): %v", name, err)
			}
			return
		}
		if !valid {
			t.Fatalf("SetName(%q) accepted an invalid name", name)
		}
		got, err := d.Name()
		if err != nil || got != name {
			t.Errorf("Name() = %q, %v, want %q", got, err, name)
		}
	})
}

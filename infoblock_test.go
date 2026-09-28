package blinkstick

import (
	"bytes"
	"errors"
	"testing"
)

func TestInfoBlockRoundTrip(t *testing.T) {
	d, _ := openFake(Square)
	if err := d.SetInfoBlock(1, []byte("office")); err != nil {
		t.Fatalf("SetInfoBlock: %v", err)
	}
	if err := d.SetInfoBlock(2, []byte("desk")); err != nil {
		t.Fatalf("SetInfoBlock: %v", err)
	}
	for n, want := range map[int]string{1: "office", 2: "desk"} {
		got, err := d.InfoBlock(n)
		if err != nil {
			t.Fatalf("InfoBlock(%d): %v", n, err)
		}
		if string(got) != want {
			t.Errorf("InfoBlock(%d) = %q, want %q", n, got, want)
		}
	}
}

func TestSetInfoBlockReport(t *testing.T) {
	d, ft := openFake(Nano)
	d.SetInfoBlock(2, []byte("hi"))
	sent := ft.sends[0]
	want := make([]byte, 33)
	want[0], want[1], want[2] = 3, 'h', 'i'
	if !bytes.Equal(sent, want) {
		t.Errorf("sent % x, want % x", sent, want)
	}
}

func TestInfoBlockEmpty(t *testing.T) {
	d, _ := openFake(Nano)
	got, err := d.InfoBlock(1)
	if err != nil {
		t.Fatalf("InfoBlock: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("InfoBlock = %q, want empty", got)
	}
}

func TestSetInfoBlockFull(t *testing.T) {
	d, _ := openFake(Nano)
	data := bytes.Repeat([]byte("x"), 32)
	if err := d.SetInfoBlock(1, data); err != nil {
		t.Fatalf("SetInfoBlock(32 bytes): %v", err)
	}
	if got, _ := d.InfoBlock(1); !bytes.Equal(got, data) {
		t.Errorf("InfoBlock = %q, want %q", got, data)
	}
}

func TestInfoBlockOutOfRange(t *testing.T) {
	d, ft := openFake(Nano)
	if err := d.SetInfoBlock(1, make([]byte, 33)); !errors.Is(err, ErrOutOfRange) {
		t.Errorf("SetInfoBlock(33 bytes) = %v, want ErrOutOfRange", err)
	}
	for _, n := range []int{0, 3} {
		if _, err := d.InfoBlock(n); !errors.Is(err, ErrOutOfRange) {
			t.Errorf("InfoBlock(%d) = %v, want ErrOutOfRange", n, err)
		}
		if err := d.SetInfoBlock(n, nil); !errors.Is(err, ErrOutOfRange) {
			t.Errorf("SetInfoBlock(%d) = %v, want ErrOutOfRange", n, err)
		}
	}
	if ft.sendCount() != 0 {
		t.Errorf("sends = %d, want 0", ft.sendCount())
	}
}

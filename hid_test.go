package blinkstick

import (
	"errors"
	"testing"
)

// stubHID replaces hidapi Init and Exit with counters.
func stubHID(t *testing.T) (inits, exits *int) {
	origInit, origExit := hidInit, hidExit
	var i, e int
	hidInit = func() error { i++; return nil }
	hidExit = func() error { e++; return nil }
	t.Cleanup(func() {
		hidInit, hidExit = origInit, origExit
		hidRefs = 0
	})
	return &i, &e
}

func TestHIDRefCounting(t *testing.T) {
	inits, exits := stubHID(t)
	acquireHID()
	acquireHID()
	if *inits != 1 {
		t.Fatalf("inits = %d after two acquires, want 1", *inits)
	}
	releaseHID()
	if *exits != 0 {
		t.Fatalf("exits = %d with one device still open, want 0", *exits)
	}
	releaseHID()
	if *exits != 1 {
		t.Fatalf("exits = %d after last release, want 1", *exits)
	}
	acquireHID()
	if *inits != 2 {
		t.Errorf("inits = %d after reacquire, want 2", *inits)
	}
}

func TestHIDInitFailureDoesNotCount(t *testing.T) {
	stubHID(t)
	boom := errors.New("boom")
	hidInit = func() error { return boom }
	if err := acquireHID(); !errors.Is(err, boom) {
		t.Fatalf("acquireHID = %v, want %v", err, boom)
	}
	if hidRefs != 0 {
		t.Errorf("hidRefs = %d, want 0", hidRefs)
	}
}

func TestHIDReleaseWithoutAcquire(t *testing.T) {
	_, exits := stubHID(t)
	if err := releaseHID(); err != nil {
		t.Fatalf("releaseHID: %v", err)
	}
	if *exits != 0 || hidRefs != 0 {
		t.Errorf("exits = %d, hidRefs = %d, want 0, 0", *exits, hidRefs)
	}
}

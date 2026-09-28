package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/bawdo/blinkstick/go/blinkstick"
)

type fakeStick struct {
	set    []blinkstick.RGB
	setErr error
	closed bool
}

func (f *fakeStick) SetAll(c blinkstick.RGB) error {
	f.set = append(f.set, c)
	return f.setErr
}

func (f *fakeStick) Close() error {
	f.closed = true
	return nil
}

func opener(s *fakeStick) func() (stick, error) {
	return func() (stick, error) { return s, nil }
}

func TestRunOnSetsWhite(t *testing.T) {
	s := &fakeStick{}
	var stderr bytes.Buffer
	if code := run([]string{"on"}, opener(s), &stderr); code != 0 {
		t.Fatalf("exit = %d, stderr = %q", code, stderr.String())
	}
	if len(s.set) != 1 || s.set[0] != blinkstick.White {
		t.Errorf("set = %v, want [White]", s.set)
	}
	if !s.closed {
		t.Error("stick not closed")
	}
}

func TestRunOffSetsOff(t *testing.T) {
	s := &fakeStick{}
	if code := run([]string{"off"}, opener(s), &bytes.Buffer{}); code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if len(s.set) != 1 || s.set[0] != blinkstick.Off {
		t.Errorf("set = %v, want [Off]", s.set)
	}
}

func TestRunUsageErrors(t *testing.T) {
	for _, args := range [][]string{nil, {"blue"}, {"on", "extra"}} {
		s := &fakeStick{}
		var stderr bytes.Buffer
		if code := run(args, opener(s), &stderr); code != 2 {
			t.Errorf("args %q: exit = %d, want 2", args, code)
		}
		if !strings.Contains(stderr.String(), "usage: blink on|off") {
			t.Errorf("args %q: stderr = %q, want usage", args, stderr.String())
		}
		if len(s.set) != 0 {
			t.Errorf("args %q: device touched on usage error", args)
		}
	}
}

func TestRunOpenFailure(t *testing.T) {
	open := func() (stick, error) { return nil, errors.New("no device found") }
	var stderr bytes.Buffer
	if code := run([]string{"on"}, open, &stderr); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "no device found") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

func TestRunSetFailureStillCloses(t *testing.T) {
	s := &fakeStick{setErr: errors.New("set colour failed")}
	var stderr bytes.Buffer
	if code := run([]string{"on"}, opener(s), &stderr); code != 1 {
		t.Fatalf("exit = %d, want 1", code)
	}
	if !s.closed {
		t.Error("stick not closed after failure")
	}
	if !strings.Contains(stderr.String(), "set colour failed") {
		t.Errorf("stderr = %q", stderr.String())
	}
}

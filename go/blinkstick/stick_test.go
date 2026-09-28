package blinkstick

import (
	"bytes"
	"errors"
	"testing"
)

// fakeWriter fails the first len(errs) sends with those errors, then succeeds.
type fakeWriter struct {
	errs   []error
	sent   [][]byte
	closed bool
}

func (f *fakeWriter) SendFeatureReport(p []byte) (int, error) {
	f.sent = append(f.sent, append([]byte(nil), p...))
	if len(f.errs) > 0 {
		err := f.errs[0]
		f.errs = f.errs[1:]
		return 0, err
	}
	return len(p), nil
}

func (f *fakeWriter) Close() error {
	f.closed = true
	return nil
}

func TestSetAllSendsFrameReport(t *testing.T) {
	w := &fakeWriter{}
	if err := New(w).SetAll(White); err != nil {
		t.Fatalf("SetAll: %v", err)
	}
	if len(w.sent) != 1 {
		t.Fatalf("sends = %d, want 1", len(w.sent))
	}
	if want := FrameReport(0, White); !bytes.Equal(w.sent[0], want) {
		t.Errorf("sent % x, want % x", w.sent[0], want)
	}
}

func TestSetAllRetriesTransientFailure(t *testing.T) {
	w := &fakeWriter{errs: []error{errors.New("general error"), errors.New("general error")}}
	if err := New(w).SetAll(Off); err != nil {
		t.Fatalf("SetAll: %v", err)
	}
	if len(w.sent) != 3 {
		t.Errorf("sends = %d, want 3", len(w.sent))
	}
}

func TestSetAllGivesUpAfterThreeAttempts(t *testing.T) {
	boom := errors.New("general error")
	w := &fakeWriter{errs: []error{boom, boom, boom, boom}}
	err := New(w).SetAll(White)
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v, want wrapping %v", err, boom)
	}
	if len(w.sent) != 3 {
		t.Errorf("sends = %d, want 3", len(w.sent))
	}
}

func TestCloseClosesWriter(t *testing.T) {
	w := &fakeWriter{}
	if err := New(w).Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !w.closed {
		t.Error("writer not closed")
	}
}

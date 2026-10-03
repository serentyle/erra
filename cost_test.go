package erra

import (
	"errors"
	"testing"
)

func TestCaptureCount(t *testing.T) {
	callersCount.Store(0)
	if Wrap(nil, "x") != nil || Code("not_found").Wrap(nil, "x") != nil {
		t.Fatal("nil wrap returned an error")
	}
	if n := callersCount.Load(); n != 0 {
		t.Fatalf("nil wrap called Callers %d times", n)
	}

	err := New("a")
	if n := callersCount.Load(); n != 1 {
		t.Fatalf("New called Callers %d times", n)
	}
	Wrap(err, "b")
	Code("not_found").Wrap(err, "c")
	if n := callersCount.Load(); n != 1 {
		t.Fatalf("wrap over an origin called Callers %d times", n)
	}
	Wrap(errors.New("plain"), "d")
	if n := callersCount.Load(); n != 2 {
		t.Fatalf("wrap of a foreign error called Callers %d times", n)
	}
}

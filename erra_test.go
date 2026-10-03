package erra_test

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/serentyle/erra"
)

const (
	invalidArgument erra.Code = "user.invalid_argument"
	notFound        erra.Code = "user.not_found"
)

var bornInInit = erra.New("born in init")

func TestValidateEmail(t *testing.T) {
	err := invalidArgument.New("email is required", slog.String("field", "email"))
	if err.Error() != "email is required" {
		t.Fatalf("Error() = %q", err.Error())
	}
	if erra.CodeOf(err) != invalidArgument {
		t.Fatalf("CodeOf() = %q", erra.CodeOf(err))
	}
	attrs := erra.AttrsOf(err)
	if len(attrs) != 1 || attrs[0].Key != "field" || attrs[0].Value.String() != "email" {
		t.Fatalf("AttrsOf() = %#v", attrs)
	}
	if fmt.Sprintf("%+v", err) != err.Error() {
		t.Fatalf("%%+v = %q", fmt.Sprintf("%+v", err))
	}
	if fmt.Sprintf("%s", err) != err.Error() || fmt.Sprintf("%q", err) != err.Error() {
		t.Fatalf("verbs = %q %q", fmt.Sprintf("%s", err), fmt.Sprintf("%q", err))
	}
}

func TestFindVehicleAndLoad(t *testing.T) {
	err := findVehicle("veh-1")
	if err.Error() != "find vehicle: sql: no rows in result set" {
		t.Fatalf("Error() = %q", err.Error())
	}
	if erra.CodeOf(err) != notFound {
		t.Fatalf("CodeOf() = %q", erra.CodeOf(err))
	}
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("errors.Is(sql.ErrNoRows) = false")
	}
	attrs := erra.AttrsOf(err)
	if len(attrs) != 1 || attrs[0].Key != "vehicle_id" || attrs[0].Value.String() != "veh-1" {
		t.Fatalf("AttrsOf() = %#v", attrs)
	}
	stack := erra.StackOf(err)
	if len(stack) == 0 {
		t.Fatal("StackOf() is empty")
	}
	if stack[0].Function != "github.com/serentyle/erra_test.findVehicle" {
		t.Fatalf("stack[0].Function = %q", stack[0].Function)
	}
	if !strings.HasSuffix(stack[0].File, "/erra_test.go") {
		t.Fatalf("stack[0].File = %q", stack[0].File)
	}
	if stack[0].Line != 95 {
		t.Fatalf("stack[0].Line = %d", stack[0].Line)
	}
	wantLine := stack[0].Function + " " + stack[0].File + ":" + "95"
	if stack.String() == "" || !strings.HasPrefix(stack.String(), wantLine) {
		t.Fatalf("String() = %q", stack.String())
	}
	lines, ok := stack.LogValue().Any().([]string)
	if !ok || len(lines) != len(stack) || lines[0] != wantLine {
		t.Fatalf("LogValue() = %#v", stack.LogValue().Any())
	}

	loaded := erra.Wrap(err, "load vehicle", slog.String("vehicle_id", "veh-1"))
	if loaded.Error() != "load vehicle: find vehicle: sql: no rows in result set" {
		t.Fatalf("loaded Error() = %q", loaded.Error())
	}
	if erra.CodeOf(loaded) != notFound {
		t.Fatalf("loaded CodeOf() = %q", erra.CodeOf(loaded))
	}
	loadedStack := erra.StackOf(loaded)
	if len(loadedStack) != len(stack) || loadedStack[0] != stack[0] {
		t.Fatalf("stack grew: %#v", loadedStack[0])
	}
	if errors.Unwrap(loaded) != err {
		t.Fatal("Unwrap did not keep the wrapped erra value")
	}
}

func findVehicle(id string) error {
	return notFound.Wrap(sql.ErrNoRows, "find vehicle", slog.String("vehicle_id", id))
}

func TestBornInInitHasNoStack(t *testing.T) {
	if got := erra.StackOf(bornInInit); got != nil {
		t.Fatalf("StackOf(init) = %#v", got)
	}
	wrapped := erra.Wrap(bornInInit, "later")
	stack := erra.StackOf(wrapped)
	if len(stack) == 0 || stack[0].Function != "github.com/serentyle/erra_test.TestBornInInitHasNoStack" {
		t.Fatalf("stack = %#v", stack)
	}
}

func TestNilAndEmpty(t *testing.T) {
	if erra.Wrap(nil, "x") != nil || notFound.Wrap(nil, "x") != nil {
		t.Fatal("nil wrap returned an error")
	}
	if erra.CodeOf(nil) != "" {
		t.Fatal("CodeOf(nil) is not zero")
	}
	if erra.AttrsOf(nil) != nil || erra.StackOf(nil) != nil {
		t.Fatal("nil readers were not nil")
	}
	if got := notFound.New("").Error(); got != "user.not_found" {
		t.Fatalf("empty phrase Error() = %q", got)
	}
	if got := erra.New("").Error(); got != "" {
		t.Fatalf("empty New Error() = %q", got)
	}
	if got := erra.Wrap(errors.New("cause"), "").Error(); got != "cause" {
		t.Fatalf("empty wrap Error() = %q", got)
	}
	if erra.CodeOf(erra.Code("").New("x")) != "" || erra.New("x").Error() != "x" {
		t.Fatal("zero Code.New differed from New")
	}
}

func TestDuplicateAttrsAndJoin(t *testing.T) {
	inner := erra.New("inner", slog.String("id", "a"), slog.String("id", "b"))
	outer := erra.Wrap(inner, "outer", slog.String("id", "c"))
	attrs := erra.AttrsOf(outer)
	got := make([]string, len(attrs))
	for i, attr := range attrs {
		got[i] = attr.Key + "=" + attr.Value.String()
	}
	want := []string{"id=c", "id=a", "id=b"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("attrs = %v", got)
	}

	left := erra.New("left", slog.String("side", "L"))
	right := notFound.New("right", slog.String("side", "R"))
	joined := errors.Join(left, right)
	if erra.CodeOf(joined) != notFound {
		t.Fatalf("CodeOf(join) = %q", erra.CodeOf(joined))
	}
	joinedAttrs := erra.AttrsOf(joined)
	if len(joinedAttrs) != 2 || joinedAttrs[0].Value.String() != "L" || joinedAttrs[1].Value.String() != "R" {
		t.Fatalf("join attrs = %#v", joinedAttrs)
	}
	if erra.StackOf(joined)[0].Function != "github.com/serentyle/erra_test.TestDuplicateAttrsAndJoin" {
		t.Fatalf("join stack = %q", erra.StackOf(joined)[0].Function)
	}
}

type pathError struct{ Path string }

func (e *pathError) Error() string { return "path " + e.Path }

func TestAsAndAttrCopy(t *testing.T) {
	cause := &pathError{Path: "a.txt"}
	err := erra.Wrap(cause, "open")
	var got *pathError
	if !errors.As(err, &got) || got.Path != "a.txt" {
		t.Fatalf("errors.As = %#v", got)
	}

	attrs := []slog.Attr{slog.String("k", "v")}
	err = erra.New("m", attrs...)
	attrs[0] = slog.String("k", "changed")
	if erra.AttrsOf(err)[0].Value.String() != "v" {
		t.Fatal("caller mutated the stored attr")
	}
}

func TestForeignWrapperKeepsCode(t *testing.T) {
	inner := notFound.New("missing")
	mid := fmt.Errorf("mid: %w", inner)
	outer := erra.Wrap(mid, "outer")
	if outer.Error() != "outer: mid: missing" {
		t.Fatalf("Error() = %q", outer.Error())
	}
	if erra.CodeOf(outer) != notFound {
		t.Fatalf("CodeOf() = %q", erra.CodeOf(outer))
	}
	if erra.StackOf(outer)[0] != erra.StackOf(inner)[0] {
		t.Fatal("wrap over an origin recorded a second stack")
	}
}

//go:noinline
func deep(n int) error {
	if n == 0 {
		return erra.New("deep")
	}
	return deep(n - 1)
}

func TestStackKeepsYoungest32(t *testing.T) {
	err := deep(40)
	stack := erra.StackOf(err)
	if len(stack) != 32 {
		t.Fatalf("len(stack) = %d", len(stack))
	}
	if stack[0].Function != "github.com/serentyle/erra_test.deep" {
		t.Fatalf("youngest = %q", stack[0].Function)
	}
}

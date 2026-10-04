package erra

import (
	"log/slog"
	"runtime"
	"strconv"
	"strings"
)

// Location はスタックの 1 フレームである。
// Function、File、Line は runtime.CallersFrames の値である。File はフルパスである。
type Location struct {
	Function string
	File     string
	Line     int
}

// Stack は若いフレームから順に並ぶ。
type Stack []Location

// String は 1 フレームを 1 行の "function file:line" にして、若い順に改行でつなぐ。
// 空なら空文字列を返す。末尾に改行は付けない。
func (s Stack) String() string {
	if len(s) == 0 {
		return ""
	}
	var b strings.Builder
	for i, loc := range s {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(loc.line())
	}
	return b.String()
}

// LogValue は String の各行を要素にしたリストを返す。
func (s Stack) LogValue() slog.Value {
	lines := make([]string, len(s))
	for i, loc := range s {
		lines[i] = loc.line()
	}
	return slog.AnyValue(lines)
}

func (loc Location) line() string {
	return loc.Function + " " + loc.File + ":" + strconv.Itoa(loc.Line)
}

// StackOf は読み順で最初の起点のスタックを返す。
// 起点が無ければ nil を返す。フレームは若い順で、先頭は erra の外の呼び出し元である。
func StackOf(err error) Stack {
	var pcs []uintptr
	walk(err, func(fr frame) bool {
		if len(fr.pcs) > 0 {
			pcs = fr.pcs
			return false
		}
		return true
	})
	if len(pcs) == 0 {
		return nil
	}
	return resolve(pcs)
}

const maxFrames = 32

// callersSkip counts frames from runtime.Callers to the caller outside erra.
// 0 is Callers, 1 is capture, 2 is newError, 3 is New or Wrap.
// Those functions are marked noinline. A different depth would make the first frame land inside erra.
const callersSkip = 4

var doInitPCs []uintptr

func init() {
	var buf [maxFrames]uintptr
	n := runtime.Callers(0, buf[:])
	for _, pc := range buf[:n] {
		if pc == 0 {
			continue
		}
		fn := runtime.FuncForPC(pc - 1)
		if fn == nil {
			continue
		}
		if strings.HasPrefix(fn.Name(), "runtime.doInit") {
			doInitPCs = append(doInitPCs, pc)
		}
	}
}

//go:noinline
func capture() []uintptr {
	var buf [maxFrames]uintptr
	n := runtime.Callers(callersSkip, buf[:])
	if n == 0 {
		return nil
	}
	pcs := buf[:n]
	if inInit(pcs) {
		return nil
	}
	out := make([]uintptr, n)
	copy(out, pcs)
	return out
}

func inInit(pcs []uintptr) bool {
	if len(doInitPCs) == 0 {
		return false
	}
	for _, pc := range pcs {
		for _, want := range doInitPCs {
			if pc == want {
				return true
			}
		}
	}
	return false
}

func resolve(pcs []uintptr) Stack {
	frames := runtime.CallersFrames(pcs)
	out := make(Stack, 0, len(pcs))
	for {
		fr, more := frames.Next()
		out = append(out, Location{
			Function: fr.Function,
			File:     fr.File,
			Line:     fr.Line,
		})
		if !more {
			break
		}
	}
	return out
}

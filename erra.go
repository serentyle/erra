// Package erra は、比較できるコードと構造化属性と、起点で一度だけ記録するスタックを、標準の error の鎖に載せる。
//
// コードがある失敗は [Code.New] と [Code.Wrap] で作る。コードがまだ無い失敗は [New] と [Wrap] で作る。境界では [CodeOf]、[AttrsOf]、[StackOf] で読む。
//
// このパッケージはログを出さない。HTTP ステータスも決めない。通常の失敗で panic しない。
package erra

import (
	"fmt"
	"io"
	"log/slog"
)

// Code は比較できるエラーコードである。
// ゼロ値は未設定を表す。アプリケーションが定数として宣言する。
type Code string

// New はコード c を持つエラーを作る。
// 実行時に呼ぶと呼び出し元のスタックを記録する。パッケージ初期化中は記録しない。
//
//go:noinline
func (c Code) New(phrase string, attrs ...slog.Attr) error {
	return newError(c, nil, phrase, attrs)
}

// Wrap は err にコード c とフレーズと属性を足して返す。
// err が nil なら nil を返す。c がゼロ値なら、この層は分類しない。
// 原因の鎖に起点が無いときだけ、スタックを記録する。
//
//go:noinline
func (c Code) Wrap(err error, phrase string, attrs ...slog.Attr) error {
	if err == nil {
		return nil
	}
	return newError(c, err, phrase, attrs)
}

// New はコードの無いエラーを作る。
// 実行時に呼ぶと呼び出し元のスタックを記録する。パッケージ初期化中は記録しない。
//
//go:noinline
func New(phrase string, attrs ...slog.Attr) error {
	return newError("", nil, phrase, attrs)
}

// Wrap は err にフレーズと属性を足して返す。コードは変えない。
// err が nil なら nil を返す。
// 原因の鎖に起点が無いときだけ、スタックを記録する。
//
//go:noinline
func Wrap(err error, phrase string, attrs ...slog.Attr) error {
	if err == nil {
		return nil
	}
	return newError("", err, phrase, attrs)
}

// CodeOf は読み順で最初の非ゼロコードを返す。無ければゼロ値を返す。
// err が nil ならゼロ値を返す。
func CodeOf(err error) Code {
	var code Code
	walk(err, func(fr frame) bool {
		if fr.code != "" {
			code = fr.code
			return false
		}
		return true
	})
	return code
}

// AttrsOf は読み順で全部の層の属性を集めた新しいスライスを返す。
// 重複キーは残す。属性が無ければ nil を返す。
func AttrsOf(err error) []slog.Attr {
	var out []slog.Attr
	walk(err, func(fr frame) bool {
		out = append(out, fr.attrs...)
		return true
	})
	if len(out) == 0 {
		return nil
	}
	return out
}

type layer struct {
	code   Code
	phrase string
	attrs  []slog.Attr
	cause  error
}

type origin struct {
	layer
	pcs []uintptr
}

type frame struct {
	code  Code
	attrs []slog.Attr
	pcs   []uintptr
}

//go:noinline
func newError(code Code, cause error, phrase string, attrs []slog.Attr) error {
	attrs = copyAttrs(attrs)
	if cause != nil && hasOrigin(cause) {
		return &layer{code: code, phrase: phrase, attrs: attrs, cause: cause}
	}
	pcs := capture()
	base := layer{code: code, phrase: phrase, attrs: attrs, cause: cause}
	if pcs == nil {
		return &base
	}
	return &origin{layer: base, pcs: pcs}
}

func copyAttrs(attrs []slog.Attr) []slog.Attr {
	if len(attrs) == 0 {
		return nil
	}
	out := make([]slog.Attr, len(attrs))
	copy(out, attrs)
	return out
}

func hasOrigin(err error) bool {
	found := false
	walk(err, func(fr frame) bool {
		if len(fr.pcs) > 0 {
			found = true
			return false
		}
		return true
	})
	return found
}

// walk visits layers in the same order as errors.As.
func walk(err error, yield func(frame) bool) bool {
	for err != nil {
		switch e := err.(type) {
		case *origin:
			if !yield(frame{code: e.code, attrs: e.attrs, pcs: e.pcs}) {
				return false
			}
			err = e.cause
		case *layer:
			if !yield(frame{code: e.code, attrs: e.attrs}) {
				return false
			}
			err = e.cause
		default:
			switch u := err.(type) {
			case interface{ Unwrap() error }:
				err = u.Unwrap()
			case interface{ Unwrap() []error }:
				for _, child := range u.Unwrap() {
					if !walk(child, yield) {
						return false
					}
				}
				return true
			default:
				return true
			}
		}
	}
	return true
}

func (l *layer) Error() string {
	return errorText(l.phrase, l.code, l.cause)
}

func (o *origin) Error() string {
	return errorText(o.phrase, o.code, o.cause)
}

func (l *layer) Unwrap() error { return l.cause }

func (o *origin) Unwrap() error { return o.cause }

// Format は動詞を無視する。%+v が層のフィールドを出さないようにするためである。
func (l *layer) Format(s fmt.State, verb rune) {
	io.WriteString(s, l.Error())
}

// Format は動詞を無視する。%+v が層のフィールドを出さないようにするためである。
func (o *origin) Format(s fmt.State, verb rune) {
	io.WriteString(s, o.Error())
}

func errorText(phrase string, code Code, cause error) string {
	var causeText string
	if cause != nil {
		causeText = cause.Error()
	}
	switch {
	case phrase != "" && causeText != "":
		return phrase + ": " + causeText
	case phrase != "":
		return phrase
	case causeText != "":
		return causeText
	case code != "":
		return string(code)
	default:
		return ""
	}
}

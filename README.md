# erra

`erra` は、他の Go プログラムが import するエラーライブラリである。関数は標準の `error` を返す。コード、属性、スタックをエラーに載せて運び、境界のハンドラが一度だけ読む。`erra` はログを出さない。HTTP ステータスも決めない。

依存は標準ライブラリだけである。Go 1.27 で使う。

```
go get github.com/serentyle/erra
```

## コードを付けて返す

コードはアプリケーションの定数にする。衝突を避けるなら、`user.not_found` のようにパッケージ名を前に付ける。

```go
const NotFound erra.Code = "user.not_found"

return NotFound.Wrap(err, "find vehicle", slog.String("vehicle_id", id))
```

`NotFound.Wrap` は原因を残す。`errors.Is` は包む前のセンチネルに一致する。原因を残したくないときは `NotFound.New` を使う。

コードがまだ無いときは `erra.Wrap` と `erra.New` を使う。`erra.Wrap` はコードを変えない。

```go
return erra.Wrap(err, "load vehicle", slog.String("vehicle_id", id))
```

`Wrap` に `nil` を渡すと `nil` が返る。

## 境界で一度だけ読む

```go
func status(err error) int {
	switch erra.CodeOf(err) {
	case NotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
```

`CodeOf` は外側の非ゼロコードを一つ返す。`AttrsOf` は全部の層の属性を、外側から渡した順で返す。同じキーも残す。`StackOf` は起点のスタックを、若いフレームから順に返す。何度包んでもスタックは増えない。

`Error` の文字列はフレーズを `": "` でつないだものである。コード、属性、スタックは含めない。`%+v` も同じ文字列を出す。

フレーズは短い句にする。値は `slog.Attr` に置く。先頭を大文字にせず、末尾に句点を置かず、`failed to` を重ねない。

パッケージ変数のセンチネルは `errors.New` で作る。失敗した場所で `Wrap` する。初期化中の `New` はスタックを記録しない。

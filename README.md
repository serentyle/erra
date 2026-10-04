# erra

[![Go Reference](https://pkg.go.dev/badge/github.com/serentyle/erra.svg)](https://pkg.go.dev/github.com/serentyle/erra)

Go の標準 `error` に、比較できるエラーコード、構造化属性、スタックトレースを載せる軽量ライブラリです。

- **標準の `error` のまま使える**: 戻り値は `error` で、`errors.Is`、`errors.As`、`errors.Join` とそのまま組み合わせられます。
- **コード**: `erra.Code` 型の定数でエラーを分類し、HTTP ステータスやリトライ可否の判断に使えます。
- **構造化属性**: `log/slog` の `slog.Attr` を各層に付け、境界でまとめて取り出せます。
- **スタックは起点で一度だけ**: 何度 `Wrap` しても記録は最初の 1 回だけで、最大 32 フレームです。
- **依存なし**: 標準ライブラリだけで動きます。

## Installation

```sh
go get github.com/serentyle/erra
```

## Usage

### コードを付けてエラーを返す

コードはアプリケーション側で定数として宣言します。`user.not_found` のようにパッケージ名を前置すると、他のパッケージのコードと衝突しにくくなります。

```go
const (
	InvalidArgument erra.Code = "user.invalid_argument"
	NotFound        erra.Code = "user.not_found"
)

func findVehicle(id string) error {
	err := db.QueryRow(/* ... */).Scan(/* ... */)
	return NotFound.Wrap(err, "find vehicle", slog.String("vehicle_id", id))
}

func validate(email string) error {
	if email == "" {
		return InvalidArgument.New("email is required", slog.String("field", "email"))
	}
	return nil
}
```

- `Code.Wrap` は原因のエラーを保持するので、`errors.Is(err, sql.ErrNoRows)` のように包む前のエラーとも照合できます。
- 原因を持たない新しいエラーは `Code.New` で作ります。
- `Wrap` に `nil` を渡すと `nil` を返すため、`if err != nil` の分岐なしで呼べます。

### コードが無い場合

分類が決まっていない層では、パッケージ関数の `erra.New` と `erra.Wrap` を使います。`erra.Wrap` は内側のコードを変更しません。

```go
return erra.Wrap(err, "load vehicle", slog.String("vehicle_id", id))
```

### 境界で読み出す

HTTP ハンドラやジョブの最上位など、エラーを処理する場所で `CodeOf`、`AttrsOf`、`StackOf` を呼びます。

```go
func status(err error) int {
	switch erra.CodeOf(err) {
	case InvalidArgument:
		return http.StatusBadRequest
	case NotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

func logError(ctx context.Context, err error) {
	attrs := append(erra.AttrsOf(err),
		slog.String("code", string(erra.CodeOf(err))),
		slog.Any("stack", erra.StackOf(err)),
	)
	slog.LogAttrs(ctx, slog.LevelError, err.Error(), attrs...)
}
```

`erra.Stack` は `slog.LogValuer` を実装しているため、`slog.Any` に渡すと 1 フレーム 1 要素のリストとして出力されます。

## API

| 関数・型                             | 説明                                                                          |
| ------------------------------------ | ----------------------------------------------------------------------------- |
| `Code`                               | 比較できるエラーコード。ゼロ値は未設定を表します。                            |
| `Code.New(phrase, attrs...)`         | コード付きの新しいエラーを作ります。                                          |
| `Code.Wrap(err, phrase, attrs...)`   | `err` にコード、フレーズ、属性を足して返します。コードがゼロ値なら分類しません。 |
| `New(phrase, attrs...)`              | コードなしの新しいエラーを作ります。                                          |
| `Wrap(err, phrase, attrs...)`        | `err` にフレーズと属性を足して返します。コードは変えません。                  |
| `CodeOf(err)`                        | 最も外側にある非ゼロのコードを返します。無ければゼロ値を返します。            |
| `AttrsOf(err)`                       | 全層の属性を、外側から渡した順に返します。同じキーも残ります。                |
| `StackOf(err)`                       | 起点のスタックを、呼び出しに近いフレームから順に返します。                    |
| `Stack` / `Location`                 | スタックとそのフレーム。`String()` と `LogValue()` を持ちます。               |

## Behavior

### エラー文字列

`Error()` はフレーズを `": "` でつないだ文字列を返します。コード、属性、スタックは含みません。`%+v` を使っても同じ文字列になります。

```go
err := NotFound.Wrap(sql.ErrNoRows, "find vehicle")
err = erra.Wrap(err, "load vehicle")
fmt.Println(err) // load vehicle: find vehicle: sql: no rows in result set
```

フレーズが空のときは、原因の文字列、原因もなければコードの文字列を返します。

### スタックの記録

- スタックを記録するのは、原因のチェーンに起点が無いときだけです。`erra` のエラーを何度包んでもスタックは増えません。
- 標準の `fmt.Errorf("%w")` などで包まれていても、内側に `erra` の起点があれば、それが使われます。
- 記録するのは呼び出しに近い方から最大 32 フレームです。
- パッケージ変数の初期化中に作ったエラーにはスタックを記録しません。その場合は、実行時に最初に `Wrap` した場所が起点になります。

### 複数のエラー

`errors.Join` で束ねたエラーも読めます。`CodeOf`、`AttrsOf`、`StackOf` は、`errors.As` と同じ順序で各エラーを辿ります。

### 属性のコピー

`New` と `Wrap` は渡された属性をコピーします。呼び出し後に元のスライスを書き換えても、エラーに載った値は変わりません。

## Writing phrases

- 短い句にし、変動する値は `slog.Attr` に入れます。
- 先頭を大文字にせず、末尾に句点を付けません。
- `failed to` を重ねません。

## Development

```sh
go test ./...
```

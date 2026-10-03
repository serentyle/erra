// Package erra は、比較できるコードと構造化属性と、起点で一度だけ記録するスタックを、標準の error の鎖に載せる。
//
// コードがある失敗は [Code.New] と [Code.Wrap] で作る。コードがまだ無い失敗は [New] と [Wrap] で作る。境界では [CodeOf]、[AttrsOf]、[StackOf] で読む。
//
// このパッケージはログを出さない。HTTP ステータスも決めない。通常の失敗で panic しない。
package erra

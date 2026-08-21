---
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-19
change_kind: tooling
priority: p1
depends_on: []
spec_impact:
  kind: none
  reason: 契約と実装の食い違いを検出する検査を足すだけで、契約の意味は変えない。
---

# TypeSpec が宣言する要求・応答本体と Go が実際に読み書きするものを機械的に突き合わせる

## Motivation

`wi-381` は `UserAttributeDef` が 10 フィールド中 2 つしか宣言していないことを、`wi-382` はそれが TypeSpec 全体に広がっていることを直した。2 件とも人間が 1 つずつ handler と突き合わせて見つけたもので、同じ食い違いが明日また入っても誰も気づかない。`wi-382` の Out of Scope はこの検査を「今回の作業を将来の再発から守る唯一の手段」と書いている。

`wi-382` の作業中に、突き合わせが機械化できることは実証されている。Go の route 登録から handler を引き、handler が到達する `NoStoreJSON` / `c.JSON` / `c.Bind` / `DecodeJSON` の引数を読めば、封筒のキーと decode 先の構造体が取れる。317 operation のうち 271 でこれが成功した。残りは間接呼び出しで、そこは手で解いた。

## Scope

- operation ごとに、TypeSpec の `@body` が指す型と Go 側の実際の本体を突き合わせる検査を `tools/check` に足す。少なくとも次を検出する。
  - 応答の 1 段目のキー集合の不一致 (契約が封筒を持つ / 持たない)。
  - 要求本体のプロパティ集合と decode 先構造体の JSON タグの差。
  - path / query パラメータが要求本体のプロパティとして重複して宣言されていること。
- 突き合わせられなかった operation を沈黙させず、未解決として報告する。検査が「何を見ていないか」が読めることを優先する。
- `mise run check` に組み込む。

## Out of Scope

- ステータスコード集合の突き合わせ。`wi-386` が扱う。
- Go 側の構造体を TypeSpec から生成する方向。生成は既存 handler の全面書き換えを伴い、検査より桁違いに重い。まず差分を見えるようにする。

## Verification

- `mise run check`
- 手動確認: `wi-381` と `wi-382` が直した食い違いを意図的に 1 件戻すと、検査が落ちる。
- 手動確認: 突き合わせられなかった operation が未解決として報告され、沈黙しない。

## Risk Notes

Go の静的解析だけでは間接呼び出しを追い切れない。追えなかった operation を「合格」に数えると、検査があるのに守られていない状態になる。未解決を明示的に報告し、その件数を減らすこと自体を作業として扱う。

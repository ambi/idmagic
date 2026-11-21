---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-11-22
priority: p2
group: frontend-ui
depends_on: [wi-92664-model-ui-pages-and-interaction-flows-in-typespec]
change_kind: refactor
spec_impact: { kind: none, reason: "フォーム部品に照合用の静的な name 属性を加えるだけで、表示、送信する API の値、入力値の受理範囲、画面遷移は変更しない。" }
---

# フロントエンドの全フォーム部品に静的な name 属性を付ける

## 動機

[wi-92664](wi-92664-model-ui-pages-and-interaction-flows-in-typespec.md) は、TypeSpec の UI ページ宣言の入力項目と、実装のフォーム部品を静的な `name` 属性で照合する。
宣言にない名前は拒否できるが、静的な `name` を持たない部品は照合できず、生成するページ詳細に「実装と照合していない入力」として件数だけを示している。

2026-11-22 の測定では、`frontend/src/` のテスト以外の 93 ファイルにフォーム部品（`input`、`select`、`textarea`、`Input`、`Select`、`Textarea`、`Checkbox`、`Switch`、`Field`、`RadioGroup`、`NativeSelect`）が 381 個ある。
静的な `name` を持つのは 28 個で、式で `name` を決めるだけのものが 6 個、残りは `name` を持たない。
`value` と `onChange` だけで制御する部品や、`id` だけを持つ部品が多い。
`id` は `user-editor-username` のように DOM の識別子として項目名と別に命名されているため、照合には使えない。

この状態では、ページへ入力を足して TypeSpec の宣言を更新し忘れても、多くの場合に機械検査で検出できない。

## 対象範囲

- `frontend/src/routes/` と `frontend/src/features/` のフォーム部品に、宣言の入力項目と同じ静的な `name` 属性を付ける。
- `Field` のように `id` から `name` を作るラッパーは、呼び出し側で `name` を明示する形へ揃える。
- 繰り返しや定義駆動の入力（カスタム属性、通知のカテゴリなど）は、テンプレートとして宣言した入力項目の名前を共通の接頭辞として付ける。
- すべての部品が照合できるようになった後、wi-92664 の生成で「実装と照合していない入力」を失敗として扱う。

## 対象外

- 入力値のバリデーション規則の定義と実装。
- 表示、文言、レイアウト、送信する API の値の変更。
- `id` 属性と `htmlFor` の改名。

## 設計

部品の名前は、wi-92664 の宣言が照合に使う名前（`@field` の `name`、なければ `@bind` の束縛先のプロパティ名、なければ宣言のプロパティ名）に合わせる。
新しい名前を作るときは、API へ送るプロパティ名を優先する。

`name` 属性は、ネイティブのフォーム送信で `FormData` に値を含める。
`FormData` を読むフォームでは、名前を加えると送信内容が変わりうる。
このため、`FormData` を使う箇所を先に洗い出し、送信内容が変わらないことをテストで確かめる。

照合対象の部品は、ast-grep で静的に数えられる要素名に限る。
新しいフォーム部品のコンポーネントを追加したときは、wi-92664 の照合の対象タグにも加える。

## 計画

1. wi-92664 の完了後に、全ページの宣言とページ詳細の未照合の一覧を起点にする。
2. 機能スライスごとに `name` を付け、生成の未照合件数を減らす。
3. 未照合が 0 件になったら、照合できない部品を失敗として扱うよう生成の検査を変える。

## タスク

- [ ] T001 [Verify] `FormData` を読むフォームを洗い出し、送信内容を固定するテストを確かめる。
- [ ] T002 [App] 機能スライスごとにフォーム部品へ静的な `name` を付ける。
- [ ] T003 [Tooling] 照合できない部品を生成の失敗として扱う。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run generate-ui-pages -- --check`：全ページで未照合の部品がなく、宣言にない名前がないこと。
- `mise run test-ui-unit-file` と `mise run test-ui-e2e-file`：`FormData` を使うフォームの送信内容と、主要な画面の操作が変わらないこと。
- `mise run verify-changed -- <base>`：差分に応じた型検査、lint とテスト。

## リスク

- `FormData` を読むフォームで、名前の追加により意図しない値が送信される。T001 で該当箇所を先に固定する。
- 部品の数が多く、機能スライスごとの変更が長引く。未照合件数を進捗の指標にし、スライス単位でコミットする。

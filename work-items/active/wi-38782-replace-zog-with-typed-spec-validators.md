---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-07
priority: p2
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "受理する値と拒否する値、エラーが指すフィールドと文言、HTTP の 422 の応答を変えない。Zog の URL、UUID、メールアドレスの判定と受理範囲が異なると分かった場合は仕様の段階へ戻る。" }
---

# Zog を shared/spec の型付き検証関数へ置き換える

## 動機

[ライブラリ選定の見直し](../done/wi-90339-review-library-adoption.md)で、バックエンドの検証ライブラリ Zog をやめ、型付きの検証関数へ置き換えると決めた。
Zog v0.23.0 は v1 未満で、[採用版の README](https://github.com/Oudwins/zog/blob/v0.23.0/README.md) は minor 更新にも破壊的変更があり得ると説明する。
構造体とは別に `z.Shape` を保守しており、フィールド名を文字列で書くため、名前の誤りをコンパイラーが検出しない。
文字列の上限は、Zog がバイト数で数えるため既に使っていない（[バックエンド設計](../../docs/design/application/backend.md#文字列長の検証)）。

Go では、JSON のデコードの時点で値が型を持つ。
残る検証は、型の付いた値が条件を満たすかを確かめる述語であり、型付きの関数で読みやすく書ける。
go-playground/validator はロケールと翻訳などの依存が増え、ozzo-validation は `is` パッケージで govalidator に依存するため採らない。

## 対象範囲

- Zog を使う本番の 27 ファイル（`shared/spec` の 2 ファイル、`sharedsignals/domain` の 8 ファイル、各モジュールの domain など）を、`shared/spec` の型付き関数へ移す。
- 使っている規則は `Required`（203 か所）、`Contains`（214）、`String`（130）、`HasPrefix`（63）、`Min`（57）、`TestFunc`（53）、`Time`（49）、`Struct`（42）、`Ptr`（21）、`HasSuffix`（19）、`Default`（16）、`URL`（15）、`Slice`（14）、`UUID`（11）、`OneOf`（8）、`Len`（7）などである。
- `backend.md` の Zog に関する記述を、新しい関数の規則へ更新する。

## 対象外

- 新しい業務制約の追加と、既存の受理範囲の変更。
- 構築と更新の境界での不変条件の強制。これは [wi-15826](wi-15826-enforce-domain-state-invariants-at-construction-and-update.md) が扱う。

## 設計

置き換え後の形の例を示す。

```go
func (u User) Validate() error {
    return spec.All(
        spec.Required("ID", u.ID),
        spec.Chars("PreferredUsername", u.PreferredUsername, 1, spec.LengthName),
        spec.OptionalEmail("Email", u.Email, spec.LengthEmail),
    )
}
```

値を直接渡すため、フィールドの名前と型の誤りをコンパイラーが検出する。
エラーはフィールド名を持ち、現在の `spec.Validate` が返す形と文言を保つ。
URL は `net/url`、メールアドレスは `net/mail`、UUID は標準 `uuid` で判定する。
ただし Zog の判定と受理範囲が異なり得るため、移す前に現在の受理と拒否を特性化テストに固定する。

[wi-15826](wi-15826-enforce-domain-state-invariants-at-construction-and-update.md) と同じ domain のファイルに触れる。
先に完了した側に合わせ、後の項目は検証の呼び出し位置を変えずに中身だけを置き換える。

## 計画

1. Zog の URL、UUID、メールアドレス、`Required` とゼロ値の扱いを特性化テストに固定する。
2. `shared/spec` に型付きの関数とエラーの集約を作る。
3. モジュールごとに移し、最後に Zog を依存から外す。

## タスク

- [ ] T001 [Characterize] 判定の受理範囲とエラーの形式を特性化テストに固定する。
- [ ] T002 [Domain] `shared/spec` に型付きの検証関数を作る。
- [ ] T003 [Domain] モジュールごとにスキーマを関数へ移す。
- [ ] T004 [Docs] `backend.md` の検証の規則を更新する。
- [ ] T005 [Verify] Zog を依存から外す。

## 検証

- 触れるパッケージで `mise run test-go-package -- <package>` を実行する。
- 境界値の判定は[判定の両側と境界値](../../docs/development/testing.md#判定の両側と境界値)に従う。
- `mise run lint-go`、`mise run check-boundaries` を通す。

## リスク

手書きにすると、ゼロ値と欠落の扱いや正規化の順序を誤る可能性がある。
特性化テストで現在の判定を固定してから移す。

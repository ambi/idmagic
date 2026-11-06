---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-11-07
priority: p2
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "生成する ID の版（UUIDv4）と正規表記、解析で受け付ける表記、DB の保存形、アクショントークンの形式と失敗時の応答を変えない。受け付ける表記が変わると分かった場合は仕様の段階へ戻る。" }
---

# google/uuid の直接利用を標準 uuid へ移す

## 動機

[ライブラリ選定の見直し](../done/wi-90339-review-library-adoption.md)で、UUID の生成と解析を Go 1.27 で標準化された `uuid` パッケージへ移すと決めた。
リポジトリは既に Go 1.27.3 を要求しており、最低バージョンを上げずに移せる。
直接利用をやめれば、生成、解析、変換の経路が google/uuid の API と振る舞いに依存しなくなる。
google/uuid は go-webauthn と LDAP の間接依存として残り得るが、直接利用を移す判断には影響させない。

## 対象範囲

- 本番コードの 18 ファイルとテストの直接利用を標準 `uuid` へ移す。
  使っている API は `NewString`（19 か所）、`Validate`（5）、`UUID` 型（3）、`Parse`（2）、`NewRandom`（1）、`NewRandomFromReader`（1）である。
- `backend/shared/spec/uuid.go` の生成関数を、版を明示する `uuid.NewV4()` で実装する。
- `backend/shared/security/actiontoken/action_token.go` の乱数注入を、ID の供給方法と失敗時の契約を決めてから移す。
- 直接依存から google/uuid を外し、`go.mod` の require を間接依存の扱いへ戻す。

## 対象外

- go-webauthn と LDAP が持つ間接依存の除去。
- ID の版、表記、保存形の変更。

## 設計

標準の `uuid.New()` が選ぶ版は将来変わり得るため、現在の UUIDv4 の契約を保つ箇所では `uuid.NewV4()` を使う。
`NewV4()` は error を返さず、`NewRandomFromReader` による乱数の注入をそのまま置き換えられない。
`action_token.go` はトークンと ID の両方に注入した乱数を使い、読み取りの失敗を返す。
ID の生成を呼び出し側から注入する関数に分けるか、注入した乱数から 16 バイトを読んで版と変種のビットを設定し、`uuid.UUID` を組み立てるかを比較する。
後者を選ぶ場合は、標準 `uuid` にバイト列から組み立てる API があるかを先に確かめる。

解析は、google/uuid の `Parse` が受け付ける非正規の表記（`urn:uuid:` 接頭辞、波括弧、ハイフンなし）を標準が受け付けるかを確かめる。
受け付ける表記が狭まる場合は、外部入力に届く箇所（ページングのカーソル、管理 API のパスなど）ごとに、利用者が観測する結果が変わるかを判定する。
nil UUID の扱いと、pgx のテキスト用符号器（`RegisterUUIDAsText`）との変換も確かめる。

## 計画

1. 解析の受理範囲と nil UUID の扱いを、現在の実装で特性化テストに固定する。
2. 生成と解析を標準へ移し、`action_token.go` の設計を決めてから移す。
3. 直接依存を外し、`go mod tidy` 後の依存グラフを確かめる。

## タスク

- [ ] T001 [Characterize] 解析の受理範囲、nil UUID、DB との変換を特性化テストに固定する。
- [ ] T002 [App] 生成、検証、解析を標準 `uuid` へ移す。
- [ ] T003 [App] アクショントークンの ID の供給と失敗時の契約を決めて移す。
- [ ] T004 [Verify] 直接依存の除去と依存グラフを確かめる。

## 検証

- 触れるパッケージを `mise run test-go-package -- <package>` で確かめる。
- DB の変換を `mise run test-go` の PostgreSQL 統合テストで確かめる。
- `mise run lint-go` と `mise run audit-dependencies` を通す。

## リスク

受理する表記の違いで、外部から渡された ID の解析が失敗する可能性がある。
特性化テストで現在の受理範囲を固定してから移す。

---
status: in_progress
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-17
priority: p3
depends_on: []
change_kind: refactor
spec_impact:
  kind: none
  reason: "EmailSent の Go の型の置き場所だけを変える。イベントの名前、ワイヤ表現のフィールド、発行する条件、監査ログの記録、HTTP 契約は変えない。"
---

# EmailSent の具象の型を、発行するモジュールの domain に置く

## 動機

[バックエンド設計](../../docs/design/application/backend.md)は、具象のドメインイベントの構造体をそれが属するモジュールの `domain` パッケージに置き、共有ライブラリ `backend/shared/spec` にはイベントのエンベロープとワイヤ表現への変換だけを置くと定める。
`EmailSent` の構造体は `backend/shared/spec/events.go` にあり、この規則に反している。
発行するのは Authentication（パスワードのリセット、REQ-AUTHENTICATION-016）と IdManagement（メールアドレスの変更、REQ-IDMANAGEMENT-053）である。
語彙の検査は、この型のために、共有の木のイベントを TypeSpec の宣言元のモジュールへ帰属させる規則を持っている。

## 対象範囲

- `EmailSent` の構造体を `backend/authentication/domain` と `backend/idmanagement/domain` に置き、各モジュールの発行元とテストをそれへ向ける。
- `backend/shared/spec` から `EmailSent` を消す。
- 語彙の検査から、共有の木のイベントを帰属させる規則を消す。帰属させるイベントがなくなるためである。

## 対象外

- イベントの名前、フィールド、ワイヤ表現、発行する条件の変更。
- Tenancy のテスト送信（`SendTestNotification`）は、EX-TENANCY-018-01 が `EmailSent` を発行すると述べるが、実装は発行していない。
  仕様と実装のどちらを直すかは wi-95251 で決める。
- 通知の送信境界（`backend/shared/notification` の `Notifier`）がイベントを発行する形への変更。

## 設計

### 境界の判断の記録

入力の参照は次のとおりである。

| 入力 | 内容 |
| --- | --- |
| 要件 | REQ-AUTHENTICATION-016、REQ-IDMANAGEMENT-053 が、それぞれの送信を試みたときに `EmailSent` を発行すると定める |
| 契約 | ワイヤ表現は TypeSpec の `IdMagic.Contract.EmailSent`（`spec/modules/authentication/models.tsp`）が一つだけ定める |
| 作用の流れ | 各モジュールのユースケースが `Notifier.Notify` の戻り値を `Delivered` に入れて `Emit` へ渡す。監査は `spec.MarshalDomainEvent` でワイヤ表現へ変換して記録し、Go の型で分岐する消費者はない |
| import | IdManagement は Authentication の `password/ports` などを既に import するが、`authentication/domain` は import しない |

適用した判定基準は D2（所有者が異なる作用はそれぞれの所有者に残す）と D5（共有ライブラリはモジュールの型も業務上の意味も必要としない場合に限る）である。

| 候補 | 制約への適合 | 変更シナリオの波及 | 採否 |
| --- | --- | --- | --- |
| A. 現状を保ち、`backend/shared/spec` に置く | `backend.md` の規則に反する。用途（`purpose`）がモジュールの業務の意味を持つので D5 を満たさない | 用途を足すたびに共有ライブラリを変える | 不採用 |
| B. Authentication の `domain` に一つ置き、IdManagement が import する | IdManagement のメールアドレスの変更のイベントを Authentication が所有することになり、D2 に反する。IdManagement から `authentication/domain` への依存が増える | Authentication の `domain` の変更が IdManagement へ波及する | 不採用 |
| C. 発行する各モジュールの `domain` に同じ形の構造体を置く | D2 を満たし、依存を増やさない。ワイヤ表現の一次情報は TypeSpec のモデル一つのまま | 片方のモジュールのフィールドの追加は、TypeSpec のモデルの変更を経て両方へ及ぶ。契約の検査と監査の記録は型名ではなくワイヤ表現を見る | 採用 |

残る仮定は、`EmailSent` の Go の型で分岐する消費者が今後も現れないことである。
現れた場合は、モジュールごとの型ではなくワイヤ表現の `type` で分岐する。

## タスク

- [x] T001 [Domain] `EmailSent` を Authentication と IdManagement の `domain` に置き、発行元とテストを向け、`backend/shared/spec` から消す。
  振る舞いを保つ移動なので、既存の EX-AUTHENTICATION-016-01、EX-IDMANAGEMENT-053-03、EX-IDMANAGEMENT-053-04 のテストと監査の記録のテストが通ることで確かめる。
  `backend/shared/spec` の `EmailSent` の getter だけを確かめるテストは、対象の型と一緒に消した。
- [x] T002 [Tooling] 語彙の検査から、共有の木のイベントを帰属させる規則を消す。
- [x] T003 [Verify] 変更を検証する。
  検証欄の検査に加えて `mise run check -- main` を通した。DB のテストを含めるため、Go のテストはサンドボックスの外で実行した。

## 検証

- `mise run test-go-package -- ./backend/authentication/...`、`./backend/idmanagement/...`、`./backend/shared/spec/...`、`./backend/cmd/internal/bootstrap/...`
- `mise run lint-go`、`mise run check-boundaries`
- `mise run check-unspecified-vocabulary`、`mise run check-event-contract`
- `mise run test-tools-file -- check/src/unspecified-vocabulary.test.ts`、`mise run lint-tools`、`mise run typecheck-tools`

## リスク

- 二つの構造体のフィールドが食い違うと、同じ `type` のワイヤ表現が二通りになる。
  どちらも TypeSpec の `EmailSent` と同じタグを持つことをレビューで確かめる。

## 要約

`EmailSent` の構造体を `backend/shared/spec` から消し、発行する Authentication と IdManagement の `domain` にそれぞれ置いた。
二つの構造体は同じフィールドとタグを持ち、ワイヤ表現の一次情報は TypeSpec の `EmailSent` 一つのままである。
IdManagement が Authentication の `domain` を import する案は、IdManagement の送信のイベントの所有者を Authentication に移し依存も増やすので採らなかった。
語彙の検査から、共有の木のイベントを TypeSpec の宣言元へ帰属させる規則を消した。帰属させるイベントがなくなったためである。
`mise run spec-diff` は規範の差分を報告しない。
EX-TENANCY-018-01 はテスト送信で `EmailSent` を発行すると述べるが、実装は発行しておらず、この食い違いは wi-95251 で扱う。

---
status: in_progress
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-23
priority: p2
depends_on: [wi-90369-keep-unique-user-attribute-values-unique-on-every-write]
change_kind: bugfix
affected_spec:
  - { path: docs/modules/ws-federation/features/relying-party.md, requirement: REQ-WSFEDERATION-001 }
  - { path: docs/modules/ws-federation/features/passive-sign-in.md, requirement: REQ-WSFEDERATION-002 }
  - { path: docs/modules/ws-federation/features/active-sts.md, requirement: REQ-WSFEDERATION-004 }
  - { path: spec/modules/ws-federation/main.tsp, symbol: IdMagic.WsFederation.Operations.ConfigureEntraFederation }
  - { path: spec/modules/ws-federation/main.tsp, symbol: IdMagic.WsFederation.Operations.RegisterWsFedRelyingParty }
  - { path: spec/modules/ws-federation/models.tsp, symbol: IdMagic.Contract.EntraFederationProfile }
  - { path: spec/modules/ws-federation/models.tsp, symbol: IdMagic.Contract.WsFedRelyingParty }
  - { path: spec/modules/ws-federation/models.tsp, symbol: IdMagic.Contract.SourceAnchorNotUniqueError }
  - { path: spec/modules/ws-federation/models.tsp, symbol: IdMagic.Contract.SourceAnchorEditableError }
  - { path: spec/modules/ws-federation/models.tsp, symbol: IdMagic.Contract.SourceAnchorValueError }
  - { path: spec/modules/ws-federation/models.tsp, symbol: IdMagic.Contract.EntraRelyingPartyError }
primary_use_cases:
  - id: entra-source-anchor-protected
    requirement: REQ-WSFEDERATION-002
    observable_result: sourceAnchor の属性が unique でないか本人が編集できるテナントでは、Entra の RP への発行が 500 で拒否され、トークンが出ない。
    boundary: acceptance
    test: { path: backend/wsfederation/internal/handlers_http/entra_source_anchor_examples_test.go, name: TestEntraIssuanceRefusesAnUnprotectedSourceAnchor, task: test-go-race }
    fault_model: 構成の後に属性スキーマを変えて、sourceAnchor を本人が書き換えられるか重ねられる状態でも発行する。
  - id: entra-configuration-scope
    requirement: REQ-WSFEDERATION-001
    observable_result: フェデレーションするドメインの User だけを検査し、問題の種類ごとの type、件数、先頭のユーザー名で構成を拒否する。
    boundary: acceptance
    test: { path: backend/wsfederation/internal/handlers_http/entra_source_anchor_examples_test.go, name: TestEntraConfigurationChecksOnlyTheFederatedUsers, task: test-go-race }
    fault_model: フェデレーションしない User の欠けた値で構成を拒否するか、拒否の理由と件数を返さない。
---

# Entra の sourceAnchor を、構成の後も一意で、利用者が変えられない値に保つ

## 動機

Entra は、Assertion の ImmutableID（sourceAnchor から導く値）で Entra のユーザーを決める。
同じ ImmutableID を二人の User に発行すると、片方がもう一方の Microsoft 365 のアカウントでサインインする。
sourceAnchor の検査には次の欠陥がある。

| 欠陥 | 結果 |
| --- | --- |
| 欠落、重複、変換できない値の検査は、Entra の構成の時点に一度だけ行う。その後に作ったか、属性を変えた User は検査しない | 構成の後に、管理者の操作や CSV で別の User と同じ値を書けば、発行の時点で同じ ImmutableID を発行する |
| sourceAnchor の値を独自に変換する。GUID の表記は base64 へ変え、base64 として読める値はそのまま通し、それ以外を拒否する | 同じ ImmutableID になる GUID と base64 の二つの表記を受け付けるので、保存した値の一致では重複と分からない。Entra Connect が文字列の属性（employeeID など）をそのまま ImmutableID にする sourceAnchor は使えない |
| sourceAnchor の属性を `editable_by_user` にした属性スキーマを拒否しない | 利用者がセルフサービスのプロフィールで値を他人の値に書き換え、その人として Microsoft 365 にサインインできる |
| 構成の時点の検査は、テナントのすべての User を対象にする | Entra へフェデレーションしない User（ローカルの管理者や、ほかのドメインの User）が一人でも値を持たないと構成できない。初期データの User（`alice`）だけのテナントでも、`sourceAnchor attribute "object_guid" has no value` で拒否される |
| 拒否の理由は、英語の内部メッセージと User の ID をそのまま返し、管理画面がそれを表示する | 日本語の画面に英語の文が混ざり、どの User を直せばよいか、何件あるかが分からない |
| 汎用の RP の登録で、Entra の RP と同じ `wtrealm` を指定すると、定型設定と固定のクレームを黙って捨てて置き換える | フェイルクローズの定型設定を、手書きのクレームの設定で上書きできる |

## 対象範囲

- sourceAnchor の属性の値を、変換せずにそのまま ImmutableID として発行する。空の値と `@` を含む値は ImmutableID として使えない。
- sourceAnchor の属性は、wi-90369 の `unique` で、`editable_by_user` でない属性に限る。構成の時点と発行の時点の両方で、属性の定義を確かめる。
- 構成の時点の値の検査の対象を、構成した後の定型設定で UPN がフェデレーションするドメインに属する User に絞る。
- 構成の拒否は、問題の種類ごとの Problem Details の `type` と、値の問題では該当する User の件数と先頭の 5 人までのユーザー名を返し、管理画面は辞書の文言で表示する。
- 汎用の RP の登録で、Entra の定型設定を持つ RP と同じ `wtrealm` を指定する要求を拒否する。

## 対象外

- 値の重複の検査そのもの。wi-90369 が書き込みの時点で止める。
- 汎用の RP の削除の拒否。管理画面が Entra のフェデレーションを外す経路なので残す。
- AD の objectGUID を base64 に符号化して取り込むこと。Entra Connect と同じく、同期の側（wi-95 の Directory Connector）が行う。

## 設計

### T001 の決定

| 問い | 決定 | 根拠 |
| --- | --- | --- |
| sourceAnchor の値と ImmutableID の関係 | 属性の値をそのまま ImmutableID にする。GUID からの変換をやめる | [Entra Connect の設計の概念](https://learn.microsoft.com/en-us/entra/identity/hybrid/connect/plan-connect-design-concepts)は、sourceAnchor を immutableId と同じものとし、大文字と小文字を区別すると述べる。文字列でない属性（objectGUID）は Entra Connect が base64 に符号化し、文字列の属性はそのまま使う。値をそのまま使えば、wi-90369 の完全一致の一意性が ImmutableID の一意性になる |
| 使えない値 | 空の値と `@` を含む値 | 同じ文書は、`@` を含む属性を sourceAnchor に選べないと述べる |
| 利用者の確認 | GUID の文字列を保存しているテナントは base64 の値へ直す必要がある。直さなければその利用者のサインインは失敗し、他人のアカウントに入ることはない | 利用者が「値をそのまま使う」を選んだ |

重複を止める場所を比べ、書き込みの時点に置いた。

| 案 | 採否 | 理由 |
| --- | --- | --- |
| IdManagement が書き込みの時点で `unique` の属性の値の一意性を守る（wi-90369） | 採る | 値の持ち主が不変条件を守る。サインインの経路に処理を足さない |
| 発行の時点に、同じ値の User が一人だけであることを確かめる | 採らない | 属性に索引がなく、サインインのたびにテナントの User 数に比例する走査が入る。持ち主でないモジュールが不変条件を点検する |

### 型と操作

```go
// ImmutableIDOf は、sourceAnchor の属性の値を ImmutableID として返す。値は変換しない。
func ImmutableIDOf(attrs claimdomain.Attributes, profile *EntraFederationProfile) (string, error)

// CheckEntraSourceAnchorAttribute は、sourceAnchor の属性の定義が、重ならず本人が書き換えられない値かを確かめる。
func CheckEntraSourceAnchorAttribute(defs []userdomain.UserAttributeDef, profile *EntraFederationProfile) error // ErrSourceAnchorNotUnique、ErrSourceAnchorEditable

// SourceAnchorValueProblem は、構成の時点に値を使えない User の件数と先頭のユーザー名である。
type SourceAnchorValueProblem struct {
    Err           error // ErrSourceAnchorMissing または ErrSourceAnchorInvalid
    UserCount     int
    SampleUsernames []string // 先頭の 5 人まで
}

// CheckEntraSourceAnchorValues は、定型設定の UPN のドメインに属する User の値を確かめ、一種類の問題を返す。
func CheckEntraSourceAnchorValues(profile EntraFederationProfile, users []*userdomain.User) *SourceAnchorValueProblem
```

| 拒否 | 応答 |
| --- | --- |
| 属性が `unique` でない | 422 `source_anchor_not_unique` |
| 属性が `editable_by_user` | 422 `source_anchor_editable` |
| フェデレーションするドメインの User の値がない | 422 `source_anchor_missing`、`user_count`、`sample_usernames` |
| フェデレーションするドメインの User の値が使えない | 422 `source_anchor_invalid`、`user_count`、`sample_usernames` |
| 汎用の RP の登録が Entra の RP の `wtrealm` を指す | 409 `entra_relying_party` |

構成の時点の拒否を 400 から 422 に変えるのは、要求は読めたが業務の規則に反するからである（API の指針）。

### 引き継ぎ後の実装判断

- API クライアントは、既存の `AuthenticationAPIError` に応答本文を保持する。エラーコード、認証切れ、再試行の扱いを共通処理に保ち、Entra の画面で `user_count` と `sample_usernames` の型を確かめて表示する。
- 問題が複数種類ある場合の優先順位は仕様に固定しない（分類 b）。件数とサンプルは、返した種類について数える。
- TypeSpec のプロファイルの説明に残る GUID の変換を、採用した値の無変換の規則へ合わせる。HTTP の形は変えない。
- 共有テスト基盤では、User と同じ IdManagement の実装の属性スキーマを配線し、テストが保存先を参照できるようにする。別の保存先をスタブで足す案は、一意性を保証する所有者が分かれるため採らない。本番モジュールの境界と依存は変えない。
- 製品は未公開で、保存した属性値を自動変換しない。バイナリを戻してもデータの逆移行は不要だが、Entra の ImmutableID と保存値の一致は運用側で維持する。

## タスク

- [x] T001 [Plan] sourceAnchor の値と ImmutableID の関係を一次資料で確かめ、値をそのまま使うことを利用者と決めた。
- [x] T002 [Spec] REQ-WSFEDERATION-001、002、004、`ConfigureEntraFederation`、`RegisterWsFedRelyingParty`、Entra の定型設定の設計を改める。
- [x] T003 [Acceptance] 守られていない属性への発行、フェデレーションしない User による構成の拒否、Entra の RP の上書きを RED で確かめる。
- [x] T004 [App] 値をそのまま使う ImmutableID、属性の定義の検査、構成の検査の範囲と拒否の応答、RP の上書きの拒否を実装する。
- [x] T005 [UI] 構成の拒否を辞書の文言で表示し、件数とユーザー名を出す。
- [x] T006 [Verify] 変更を検証する。

## 検証

- `mise run test-go-changed -- main`、`mise run test-ui-unit-file`、`mise run check`、`mise run check-spec-impact`。
- ミューテーションテスト：属性の定義と値の判定、検査の範囲。
- Go の局所テストは `test-go-test`、変更と逆依存は `test-go-changed -- 91bdcc5e7` を使う。比較基準は実装前の `91bdcc5e7` とする。
- `test-go-mutation` は domain の変更行を対象とし、発行前の属性検査の切断と構成の検査の切断は手動注入で観測する。
- UI は画面の単体テスト、API エラーの単体テスト、`verify-ui`、Entra の追加画面を使う `test-ui-e2e-file` で辞書と応答の接続を確かめる。
- TypeSpec の説明とエラー契約は `check-spec`、`check-contract-drift`、`check-published-api-compat`、`render-docs`、`generate-contract` で同期する。

## リスク

- GUID の文字列を保存しているテナントでは、その文字列をそのまま発行するため、Entra 側の base64 の ImmutableID と一致せずにサインインが失敗する。製品は未公開なので自動移行せず、運用の文書で正しい保存値を案内する。

## 要約

REQ-WSFEDERATION-001、002、004 を改め、sourceAnchor の保存値を変換せずに ImmutableID として発行し、構成時と発行時に属性が `unique` で本人に編集できないことを確かめる。
構成時の値の検査をフェデレーションするドメインの User に絞り、問題別の 422 と、値の問題の件数と先頭の 5 人までのユーザー名を返す。
汎用 RP 登録による Entra の定型設定の上書きを 409 で拒否し、管理画面は既知のエラーを日本語と英語の辞書で表示する。
TypeSpec の二つの操作、プロファイルと RP の説明、四つのエラー型を同期した。

自動変異の対象は実装前の `91bdcc5e7` からの変更行に限定した。
共通発行処理のテストは別パッケージの HTTP 境界にあるため、同一パッケージだけを実行する自動変異では未被覆となる。
その分岐の反転、検査の切断、`unique` と本人編集の検査の除去、`@` の受理、値の前後の空白の除去は手動注入の対象とした。
変更していない行の検出能力と、変異ツールが作れない故障の網羅は主張しない。
値の重複の拒否は IdManagement が所有し、objectGUID の base64 への符号化は同期側の責務として残す。

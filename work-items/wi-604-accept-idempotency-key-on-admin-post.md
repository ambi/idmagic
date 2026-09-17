---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-09-18
priority: p1
depends_on: [wi-585-rewrite-api-rules-as-complete-api-guidelines]
change_kind: feature
affected_spec:
  - { path: spec/contexts/api-tokens/main.tsp, symbol: IdMagic.ApiTokens.Operations.IssueApiToken }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.CreateAdminOAuth2Client }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.CreateAdminApplication }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.CreateApplicationCategory }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.IssueApplicationClientSecret }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.RotateApplicationClientSecret }
  - { path: spec/contexts/saml/main.tsp, symbol: IdMagic.Saml.Operations.CreateSamlIdentityProviderProfile }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.CreateIdentityProviderConnection }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.IssueMfaEnrollmentBypass }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.StartUserCsvExport }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.StartGroupCsvExport }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.StartGroupMemberCsvExport }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.ImportAdminUsers }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.ImportAdminGroups }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.ImportAdminGroupMembers }
---

# 管理 API の POST で Idempotency-Key を受け付ける

## Motivation

[API ガイドライン](../docs/design/application/api-guidelines.md)の「冪等キー」は、管理 API のリソースの作成、長時間実行操作の開始、クレデンシャルを発行するカスタムメソッドで、任意指定の `Idempotency-Key` リクエストヘッダーを受け付けると定める。
現行の管理 API はこのヘッダーを受け付けない。

API アクセストークンで管理 API を呼ぶ自動化クライアントは、タイムアウトの後に最初のリクエストが処理されたかを判別できない。
再送すると、API トークン、OAuth クライアント、クライアントシークレットが二重に作成される。
重複したクレデンシャルは管理されないまま有効であり続け、攻撃対象領域を拡大する。
一意制約は自然キーを持つリソースにしか効かず、再送が 409 を受け取ったクライアントは、自身の最初のリクエストが成功したのか、他者が先に作成したのかを判別できない。

## Scope

- 冪等キーの保存、照合、レスポンスの再生を、共通のミドルウェアまたはハンドラーのラッパーとして実装する。
- `affected_spec` の API 操作で `Idempotency-Key` を受け付け、TypeSpec にヘッダーと、キーの不一致による 422、処理中の重複による 409 を宣言する。
- 保存したレスポンスを 24 時間後に削除する。
- 平文のシークレットを含むレスポンスを、可逆な秘密情報と同じエンベロープ暗号で保存する。
- API ガイドラインの適用状況を改める。

## Out of Scope

- アカウント API とブラウザー API。呼び出し元はファーストパーティーの UI であり、`POST` を自動で再送しない。
- 管理コンソールからのヘッダーの送信。必要性は別に判断する。

## Design

ヘッダーの意味は IETF の `draft-ietf-httpapi-idempotency-key-header` に従う。

保存のキーは `(tenant_id, principal_id, idempotency_key)` とする。
主体を含めるのは、別の主体が同じキーを指定したときに、他者のレスポンスを再生させないためである。
値は、リクエストのフィンガープリント（メソッド、ルート、正規化したボディのハッシュ）、状態（処理中、完了）、ステータスコード、レスポンスヘッダーの一部、レスポンスボディである。

処理の順序は次のとおりとする。

1. キーの行を「処理中」として挿入する。一意制約違反なら既存の行を読む。
2. 既存の行が完了済みで、フィンガープリントが一致すれば、保存したレスポンスを返す。一致しなければ 422 `idempotency_key_reused` を返す。
3. 既存の行が処理中なら 409 `idempotency_key_in_progress` を返す。
4. ハンドラーを実行し、2xx と 4xx のレスポンスを保存する。5xx は保存せず行を削除し、再送で再実行できるようにする。

処理中の行が異常終了で残った場合に備え、処理中の行にも期限を設ける。
期限はアドミッションコントロールの要求タイムアウトより長くする。

ハンドラーの作用と冪等キーの行の完了を同一トランザクションにできない API 操作（ジョブの投入など）では、作用が成功して行の更新が失敗すると再実行される。
このため、ハンドラーの作用自体も、Jobs の `dedup_key` と同じく冪等キーから導いた値で重複を排除する。

削除は既存の期限切れ行の削除と同じくジョブで行う。

採用しない案は、キーを必須にする案である。
既存の自動化クライアントがすべて失敗するうえ、再送しないクライアントには利点がない。

## Plan

1. 保存先のテーブルと暗号化の方式を `docs/design/data/database.md` に沿って設計する。
2. 共通の実装を Domain、Use Cases、Adapters の順に作る。
3. API 操作ごとに適用し、TypeSpec に宣言する。

## Tasks

- [ ] T001 [Design] テーブル、暗号化、期限を設計する。
- [ ] T002 [Spec] TypeSpec にヘッダーとエラーを宣言する。
- [ ] T003 [Acceptance] 同一キーの再送で API トークンが 1 件だけ作成されることを HTTP の境界で確認し、RED を記録する。
- [ ] T004 [App] 共通の実装を作る。
- [ ] T005 [App] API 操作ごとに適用する。
- [ ] T006 [Ops] 期限切れの行を削除するジョブを加える。
- [ ] T007 [Docs] API ガイドラインとデータベース設計を改める。
- [ ] T008 [Verify] 変更を検証する。

## Verification

- `mise run check-schema`
- `mise run check-status-drift`
- `mise run check-api-compat`
- `mise run verify`

## Risk Notes

保存したレスポンスは平文のクレデンシャルを含む。暗号化せずに保存すると、データベースの読み取りだけでクレデンシャルが漏えいする。暗号化の対象から漏れる API 操作がないことを、適用対象の一覧とテストで固定する。

再生したレスポンスで同じクライアントシークレットを返すことは、最初のレスポンスを受け取れなかったクライアントにとって必要な動作である。再生の範囲を、同じテナントの同じ主体に限る。

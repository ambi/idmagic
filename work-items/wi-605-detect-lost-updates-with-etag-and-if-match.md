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
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenantBranding }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenantUserAttributeSchema }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateTenantGroupAttributeSchema }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.UpdateTenantDefaultSignInPolicy }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.UpdateAppSignInPolicy }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateNotificationTemplate }
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.UpdateAdminSettings }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.UpdateAdminUser }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.UpdateGroup }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.UpdateAdminApplication }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.UpdateAdminOAuth2Client }
---

# 管理 API の更新で ETag と If-Match による競合を検出する

## Motivation

[API ガイドライン](../docs/design/application/api-guidelines.md)の「ETag と If-Match による競合検出」は、管理 API の単一リソースに対する `GET` が `ETag` を返し、`PUT`、`PATCH`、`DELETE` が `If-Match` を受け付けて、一致しなければ 412 を返すと定める。
現行の管理 API は、ライフサイクルワークフローの `expected_revision` を除き、更新の競合を検出しない。

複数の管理者が同一のリソースを同時に編集すると、後の書き込みが先の書き込みを通知なく上書きする。
ブランディング、属性スキーマ、サインインポリシーのように `PUT` で全体を置換する設定では、一方の管理者のセキュリティ設定の変更が気付かれないまま失われる。

## Scope

- 管理 API の単一リソースに対する `GET` と、更新後のリソースを返す `PUT` と `PATCH` で、強い `ETag` を返す。
- `affected_spec` の API 操作と、同じリソースの `DELETE` で `If-Match` を受け付け、不一致を 412 `precondition_failed` で返す。
- バージョンを持たないリソースにバージョン列を追加する。
- 管理コンソールが取得した `ETag` を `If-Match` で送信し、412 を受け取ったら最新の値を読み直すよう利用者に促す。
- TypeSpec にヘッダーと 412 を宣言する。
- API ガイドラインの適用状況を改める。

## Out of Scope

- SCIM の `ETag`。[[wi-250-scim-sort-and-etag]] が扱う。
- `If-Match` の必須化（428）。
- ライフサイクルワークフローの `expected_revision` の廃止。ドメインのリビジョンとして残す。

## Design

`ETag` の値はリソースのバージョン番号から算出する。
更新日時から算出すると、同一のマイクロ秒に 2 件の更新が起きた場合に区別できず、弱い比較になる。
バージョン番号は更新のたびに 1 ずつ増やし、条件付きの更新は `UPDATE ... WHERE version = $expected` で行って、影響行数が 0 なら 412 とする。
読み取りと比較を分けると、比較の後に他者の更新が入る。

ブランディングは既にバージョンを持ち、公開の `GET /api/branding` が `ETag` として返しているため、同じ値を用いる。

複数のテーブルにまたがるリソース（属性スキーマ、サインインポリシー）は、集約のルートにバージョン列を置く。

採用しない案は、各リソースの本文に `expected_revision` を持たせる案である。
リソースごとにプロパティの名前と位置を決めることになり、`DELETE` にはボディがないため同じ方式を適用できない。

## Plan

1. `affected_spec` の各リソースについて、集約のルートとバージョン列の有無を確認する。
2. 共通のヘッダー処理を作り、リソースごとにリポジトリの条件付き更新を加える。
3. 管理コンソールを追従させる。

## Tasks

- [ ] T001 [Design] 集約のルートとバージョン列を確認する。
- [ ] T002 [Spec] TypeSpec にヘッダーと 412 を宣言する。
- [ ] T003 [Acceptance] 古い `If-Match` による更新が 412 になることを HTTP の境界で確認し、RED を記録する。
- [ ] T004 [App] 共通のヘッダー処理とリポジトリの条件付き更新を作る。
- [ ] T005 [UI] 管理コンソールで `If-Match` を送信し、412 を表示する。
- [ ] T006 [Docs] API ガイドラインを改める。
- [ ] T007 [Verify] 変更を検証する。

## Verification

- `mise run check-schema`
- `mise run check-status-drift`
- `mise run check-api-compat`
- `mise run verify`
- `mise run test-ui-e2e`

## Risk Notes

`ETag` の比較を読み取りの後に行うと、比較と書き込みの間に他者の更新が入り、競合を検出できない。条件付きの `UPDATE` で比較と書き込みを一つの文にし、並行する 2 件の更新のうち 1 件だけが成功することをテストで固定する。

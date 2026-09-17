---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p2
depends_on: [wi-585-rewrite-api-rules-as-complete-api-guidelines]
change_kind: feature
affected_spec:
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.ListAgents }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.ListAdminApplications }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.ListApplicationAssignments }
  - { path: spec/contexts/provisioning/main.tsp, symbol: IdMagic.Provisioning.Operations.ListProvisioningDeliveries }
  - { path: spec/contexts/authentication/main.tsp, symbol: IdMagic.Authentication.Operations.ListAuthenticationEventBuckets }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.ListAdminOAuth2Clients }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.ListAdminConsents }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.ListGroups }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.GetAdminGroupImport }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.GetAdminGroupMemberImport }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.ListAdminUsers }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.GetAdminUserImport }
---

# ページングのレスポンスヘッダーを TypeSpec に宣言する

## Motivation

[API ガイドライン](../docs/design/application/api-guidelines.md)の「ページングのレスポンスヘッダー」は、`Link` と `Pagination-*` ヘッダーを TypeSpec に宣言すると定める。
Go のハンドラーはこれらのヘッダーを返しているが、TypeSpec はいずれのコレクションにも宣言していない。
生成したクライアントと API リファレンスからは、次ページの取得方法を読み取れない。

## Scope

- `Link`、`Pagination-Total-Items`、`Pagination-Total-Pages`、`Pagination-Current-Page`、`Pagination-Page-Size` を、TypeSpec の共有モデルとして定義する。
- キーセット方式のカーソルでページングする API 操作の 200 に、実際に返すヘッダーを宣言する。
- API ガイドラインの適用状況を改める。

## Out of Scope

- ページングしていないコレクションの改修。[[wi-596-paginate-remaining-collections-with-shared-cursor]] が扱う。
- `mise run check-contract-drift` によるヘッダーの検査。

## Design

`Link` だけを返す API 操作と、総件数を併せて返す API 操作で、共有モデルを二つに分ける。
一つのモデルに任意ヘッダーとしてまとめると、総件数を返さない API 操作でもクライアントがヘッダーの有無を確認する分岐を持つことになる。
どちらを返すかは、ハンドラーが `SetPaginationHeaders` を呼ぶかどうかで判定する。

## Plan

1. ハンドラーごとに、`SetPageLinks` と `SetPaginationLinks` のどちらを呼ぶかを確認する。
2. 共有モデルを定義し、各 API 操作に適用する。

## Tasks

- [ ] T001 [Spec] 共有モデルを定義する。
- [ ] T002 [Spec] 各 API 操作に宣言する。
- [ ] T003 [Docs] API ガイドラインの適用状況を改める。
- [ ] T004 [Verify] 変更を検証する。

## Verification

- `mise run check-spec`
- `mise run check-api-compat`
- `mise run check-rendered-spec`

## Risk Notes

宣言とハンドラーの実装が食い違っても検出する検査がない。レビューでハンドラーの呼び出しと突き合わせる。

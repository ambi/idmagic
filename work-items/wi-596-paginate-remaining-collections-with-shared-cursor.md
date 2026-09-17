---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-18
priority: p2
depends_on: [wi-585-rewrite-api-rules-as-complete-api-guidelines]
change_kind: feature
affected_spec:
  - { path: spec/contexts/tenancy/main.tsp, symbol: IdMagic.Tenancy.Operations.ListTenants }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdManagement.Operations.ListUserGroups }
  - { path: spec/contexts/authorization/main.tsp, symbol: IdMagic.Authorization.Operations.ListRelationTuples }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.ListJobs }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.ListSystemJobs }
---

# 増加するコレクションを共通のカーソルでページングする

## Motivation

[API ガイドライン](../docs/design/application/api-guidelines.md)の「ページング方式」と「カーソル」は、件数がテナントの規模または時間の経過に応じて増加するコレクションを、署名付きのキーセット方式のカーソルでページングすると定める。
次のコレクションがこのルールに従っていない。

- `ListTenants` と `ListUserGroups` はページングせず全件を返す。テナント数と所属グループ数に比例してレスポンスとクエリのコストが増える。
- `ListRelationTuples` は `limit` のみを受け付け、先頭から指定件数で打ち切るため、それ以降のタプルを取得できない。
- `ListJobs` と `ListSystemJobs` は独自のカーソルをレスポンスボディの `next_cursor` で返し、`Link` ヘッダーを返さない。

## Scope

- 上記 5 つの API 操作を、`support_http.ParsePageRequest` と `SetPageLinks` によるページングに揃える。
- Jobs の独自カーソルと `next_cursor` を廃止する。
- フロントエンドのページ送りを追従させる。
- API ガイドラインの適用状況を改める。

## Out of Scope

- ページングのレスポンスヘッダーの TypeSpec への宣言。[[wi-597-declare-pagination-response-headers]] が扱う。
- 総件数の算出。必要なコレクションは個別に扱う。

## Design

並び順はコレクションごとに固定する。
`ListTenants` は `created_at` と `id`、`ListUserGroups` はグループ名と `id`、`ListRelationTuples` はタプルの構成要素の辞書順、Jobs は現行と同じ作成日時の降順とする。
各並び順に対応するインデックスがあるかを `infra/schema/postgres.sql` で確認し、なければ追加する。

`ListUserGroups` はレスポンスに実効ロール（`effective_roles`）を含む。
実効ロールはページではなく全所属グループから算出する値なので、ページングの対象外として同じレスポンスに残すか、別の API 操作に分けるかを Plan で決める。

## Plan

1. `ListUserGroups` の実効ロールの扱いを決める。
2. 並び順とインデックスを確認する。
3. API 操作ごとに、TypeSpec、ハンドラー、リポジトリ、フロントエンドの順に改める。

## Tasks

- [ ] T001 [Decision] `ListUserGroups` の実効ロールの扱いを決める。
- [ ] T002 [Spec] TypeSpec の `cursor` と `limit` を宣言する。
- [ ] T003 [App] ハンドラーとリポジトリを改める。
- [ ] T004 [UI] フロントエンドのページ送りを改める。
- [ ] T005 [Docs] API ガイドラインの適用状況を改める。
- [ ] T006 [Verify] 変更を検証する。

## Verification

- `mise run check-contract-drift`
- `mise run check-api-compat`
- `mise run verify`
- `mise run test-ui-e2e`

## Risk Notes

`ListRelationTuples` の並び順にインデックスがないと、ページが深くなるほどクエリが遅くなる。実行計画を確認する。

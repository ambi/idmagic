---
depends_on: []
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-19
priority: p2
change_kind: bugfix
affected_spec:
  - { path: docs/domain/application/scenarios.feature.md, requirement: REQ-APPLICATION-007 }
  - { path: docs/domain/application/scenarios.feature.md, requirement: REQ-APPLICATION-008 }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.GetAdminApplication }
  - { path: spec/contexts/application/main.tsp, symbol: IdMagic.Application.Operations.GetApplicationIcon }
---

# テナントをまたぐ Application 参照の拒否契約を実装と一致させる

## Motivation

`EX-APPLICATION-007-03` は、別テナントの管理者が同じ Application id を指定した参照を `InvalidRequestError` で拒否すると定める。

`handleGetApplication` は解決済みテナントを検索キーに含め、見つからない場合は存在を隠す `404 application_not_found` を返す。

`EX-APPLICATION-008-03` も同様に、別テナントの `application_id` と id でのアイコン取得を `InvalidRequestError` で拒否すると定める。

`handleGetApplicationIcon` は `404 not_found` を返す。

テナント境界の拒否そのものはどちらも成立しており、越境した内容は返らない。

食い違うのは公開するエラー契約である。

さらに `GetAdminApplication` の TypeSpec は 200、400、401、403 だけを宣言しており、実装が返す 404 を宣言していない。

一方 `GetApplicationIcon` は 404 `ApplicationIconNotFoundError` を宣言しているため、こちらは具体例だけが実装と契約の双方から外れている。

## Scope

- 越境した Application 参照とアイコン取得について、`InvalidRequestError` と存在を隠す `404` のどちらを公開契約にするか決める。
- 決定に従い、`EX-APPLICATION-007-03`、`EX-APPLICATION-008-03`、TypeSpec、実装、テストを同じ契約へそろえる。
- `GetAdminApplication` が返す 404 を契約へ反映するか、実装を宣言済みの状態へ寄せるかを決める。
- 拒否応答に加え、別テナントの Application の名称、設定、アイコン内容が返らないことを検証する。

## Out of Scope

- Application 以外の Context が持つ越境参照のエラー契約。同じ判断は [[wi-564-name-the-refusal-a-cross-tenant-oauth-admin-token-gets]] と [[wi-573-agent-admin-api-answers-with-undeclared-statuses]] が別途扱う。
- 割り当ての主体を検査しない欠落。[[wi-625-application-assignment-accepts-any-subject]] が扱う。
- アイコンの保存形式、配信 URL の形式、キャッシュ方針の変更。

## Design

判断点は、存在を隠すテナント境界の原則を優先して `404` を規範へ反映するか、既存の `InvalidRequestError` の字面を優先して実装を変えるかである。

前者は現行実装と情報非開示の意図を保てる。

後者は具体例の字面を保てるが、別テナントに同じ id があるかを推測させない応答であることを別途確認する必要がある。

`GetApplicationIcon` は既に契約と実装が 404 で一致しているため、規範だけを動かす選択に対して費用が低い。

`GetAdminApplication` は契約が 404 を宣言していないため、どちらを選んでも TypeSpec を変更する。

この判断は公開契約を変えるため、実装前に決定する。

## Plan

1. 同種のテナント境界拒否がほかの Context で採る契約を確認する。
2. 2 つの具体例と TypeSpec の応答契約を決定する。
3. 正式な HTTP 入口から越境参照を試すテストを RED にする。
4. 必要な仕様と実装をそろえ、拒否応答と情報非開示を検証する。

## Tasks

- [ ] T001 [Decision] 越境した Application 参照とアイコン取得の公開エラー契約を決める。
- [ ] T002 [Spec] `EX-APPLICATION-007-03`、`EX-APPLICATION-008-03`、TypeSpec を決定済みの契約へそろえる。
- [ ] T003 [Acceptance] HTTP 入口で拒否応答と、Application 設定およびアイコン内容の不在を検証する。
- [ ] T004 [App] 必要な場合は handler の応答写像を変更する。
- [ ] T005 [Verify] 仕様、契約、テナント境界の検査を通し、2 件を被覆台帳から外す。

## Verification

- `mise run check-spec`
- `mise run check-contract-drift`
- `mise run check-api-compat`
- `mise run test-go-package -- ./backend/application/handlers_http`
- `mise run verify`

## Risk Notes

テナント境界の拒否を誤ると、別テナントの Application の存在または内容が漏れる。

ステータスとエラー型だけでなく、応答本文に対象の名称、プロトコル設定、アイコンのバイト列が含まれないことを観測する。

契約を `InvalidRequestError` へ寄せる選択は、id の存在を推測できる応答になりやすい。

その場合は、存在する id と存在しない id が同じ応答を返すことを観測に含める。

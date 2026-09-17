---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p2
depends_on: [wi-585-rewrite-api-rules-as-complete-api-guidelines]
change_kind: bugfix
affected_spec:
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ListAdminAuditEvents }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ExportAdminAuditEvents }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ListSystemAuditEvents }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ExportSystemAuditEvents }
---

# 監査イベントの絞り込み条件を個別のクエリパラメーターとして宣言する

## Motivation

[API ガイドライン](../docs/design/application/api-guidelines.md)の「フィルタリング」と「実際のボディの宣言」は、絞り込み条件を個別のクエリパラメーターとして TypeSpec に宣言し、サーバーが読み取る場所と宣言を一致させると定める。
監査イベントのコレクションとエクスポートは、条件を `query` という単一のオブジェクト型パラメーター（`AuditEventQuery`）として宣言している。
サーバーは `type`、`category`、`user_id`、`username`、`after`、`before`、`limit`、`q` などを個別のクエリパラメーターとして読み取るため、生成したクライアントが送るリクエストとサーバーの解釈が一致しない。

## Scope

- `ListAdminAuditEvents`、`ExportAdminAuditEvents`、`ListSystemAuditEvents`、`ExportSystemAuditEvents` のクエリパラメーターを、サーバーが読み取るものと一致させて個別に宣言する。
- API ガイドラインの適用状況を改める。

## Out of Scope

- 絞り込み条件の追加と変更。

## Design

`AuditEventQuery` の各プロパティを `@query` 付きのプロパティとして展開する。
TypeSpec の spread で共有モデルを 4 つの API 操作に適用し、条件の一覧を一か所に保つ。
ハンドラーが読み取るクエリパラメーターの一覧は `backend/audit/handlers_http/admin_audit_event_handler.go` から転記する。

## Plan

1. ハンドラーが読み取るクエリパラメーターと、`AuditEventQuery` のプロパティを突き合わせる。
2. 共有モデルを定義して適用する。

## Tasks

- [ ] T001 [Design] クエリパラメーターの一覧を確定する。
- [ ] T002 [Spec] TypeSpec を改める。
- [ ] T003 [UI] 生成した型を使うフロントエンドを追従させる。
- [ ] T004 [Docs] API ガイドラインの適用状況を改める。
- [ ] T005 [Verify] 変更を検証する。

## Verification

- `mise run check-contract-drift`
- `mise run check-api-compat`
- `mise run verify`

## Risk Notes

`filter` のように構造を持つ条件は、クエリ文字列での符号化方法を決める必要がある。ハンドラーの現行の解析方法に合わせる。

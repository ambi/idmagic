---
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-18
priority: p3
depends_on: [wi-585-rewrite-api-rules-as-complete-api-guidelines]
change_kind: tooling
spec_impact:
  kind: none
  reason: "OpenAPI 拡張で安定性区分を宣言するだけで、パス、リクエスト、レスポンスは変わらない。区分の値は現行のパスの形式が表すものと同じである。"
---

# API 操作の安定性区分を TypeSpec に宣言する

## Motivation

[API ガイドライン](../docs/design/application/api-guidelines.md)の「安定性区分の宣言」は、各 API 操作の安定性区分（`stable`、`beta`、`internal`）を TypeSpec に宣言すると定める。
現行の TypeSpec は区分を宣言しておらず、区分はパスの形式（`/api/admin/v1/`、`/api/account/v1/`、`/api/auth/`）から推測するしかない。
`beta` の API 操作を導入しても、パスの形式では `stable` と区別できない。

## Scope

- OpenAPI 拡張 `x-interface-stability` を定義し、汎用 API の全 API 操作に宣言する。
- 宣言の欠落と、パスの形式との不整合を拒否する検査を加える。
- 生成する API リファレンスに区分を表示する。
- API ガイドラインの適用状況を改める。

## Out of Scope

- `beta` の API 操作の導入。
- プロトコルエンドポイントと運用エンドポイント。

## Design

検査は `tools/check/src/admin-scopes.ts` と同じく、生成した OpenAPI を唯一の入力とする。
`internal` と宣言した API 操作が `x-api-token-scopes` に `interactive_session` 以外を宣言していれば拒否する。
`internal` の API 操作に API アクセストークンで到達できないという規則を、宣言どうしの整合として検査できる。

## Tasks

- [ ] T001 [Tooling] 検査を追加し、宣言がない状態で RED を記録する。
- [ ] T002 [Spec] 全 API 操作に宣言する。
- [ ] T003 [Tooling] API リファレンスに表示する。
- [ ] T004 [Docs] API ガイドラインの適用状況を改める。
- [ ] T005 [Verify] 変更を検証する。

## Verification

- `mise run check-spec`
- `mise run check-rendered-spec`
- `mise run test-tools`

## Risk Notes

ブラウザー API は `x-api-token-scopes` を宣言していない。`internal` との整合検査の対象を管理 API に限るか、ブラウザー API にも宣言を求めるかを実装前に決める。

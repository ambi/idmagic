---
status: accepted
authors: [tn]
created_at: 2026-07-04
---

# ADR-031: 管理ユーザー API と RBAC 基盤

## コンテキスト

Phase 4 の管理 API、テナント分離、管理 UI は、管理者主体と通常ユーザーを
区別する認可境界を必要とする。OAuth Client の scope は人間の管理権限とは
別概念であり、既存の Client 向け AuthZEN ルールだけでは `/admin/*` を保護
できない。

## 決定

`User.roles` に RBAC role 名を保存し、`admin` role を持ち `disabled_at` が無いユーザーだけを
`/admin/*` の認可対象とする。OAuth Client の scope は人間の管理権限とは別概念であり、既存の
Client 向け AuthZEN ルールだけでは `/admin/*` を保護できないため、session ベースの独立した
認可ゲートを設ける。tenant role や tenant membership は `roles` に埋め込まず、次の増分で独立
モデルとして追加する — グローバルな RBAC role とテナント単位の権限を同じ列に混在させないため。

現在の認可ゲート・CSRF/Origin 検証・`disabled_at`/`deleted_at` の区別・監査イベントの詳細は
[`backend/tenancy/ARCHITECTURE.md`](../backend/tenancy/ARCHITECTURE.md) に置く。

## 影響

- 最初の管理対象は User lifecycle に限定する。
- client / consent / key / audit-event 管理と管理 UI は同じ認可境界の上に追加する。
- デモ環境では seed user に `admin` role を付与する。本番では明示的な
  bootstrap 手順へ置き換える必要がある。

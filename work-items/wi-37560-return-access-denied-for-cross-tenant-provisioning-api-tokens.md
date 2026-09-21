---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-21
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-001 }
---

# 他テナントの Provisioning API トークンを AccessDeniedError として拒否する

## 動機

`EX-PROVISIONING-001-03` は、トークンのテナントと要求先のテナントが異なる場合に `AccessDeniedError` を返すと宣言する。
現在は他テナントで発行した有効な `provisioning:write` トークンを default tenant の入口へ提示すると、状態は変えないものの `401 invalid_token` を返す。

## 対象範囲

- Provisioning 管理 API でのクロステナント API トークン拒否を、契約どおりの `AccessDeniedError` にする。
- 拒否応答と接続・配信が増えないことを受け入れテストで検証する。

## 対象外

- 他 Context のクロステナント拒否形式の変更。
- API トークンの署名方式やトークン形式の変更。

## 設計

有効な別テナントトークンと改竄または失効したトークンは異なる拒否理由である。
検証段階で両者を一律に `invalid_token` へ畳まないよう、テナント解決とトークン検証の境界を明確にする。

## 計画

1. API トークン検証と tenant middleware の順序を確認する。
2. `EX-PROVISIONING-001-03` の HTTP RED を固定する。
3. 拒否形式を実装し、接続・配信が不変であることを確認する。

## タスク

- [ ] T001 [Readiness] トークン検証と tenant 解決の責務を決める。
- [ ] T002 [Acceptance] `EX-PROVISIONING-001-03` の 403 RED を確認する。
- [ ] T003 [Adapter] 有効な別テナントトークンを AccessDeniedError へ変換する。
- [ ] T004 [Verify] 対象テストと `mise run verify` を実行する。

## 検証

- `mise run test-go-package -- ./backend/shared/http/server_http`
- `mise run verify`

## リスク

別テナントトークンを許可するとテナント境界を越える。
拒否形式だけを変えて認可を緩めないよう、保存先の不変も同じテストで読む。

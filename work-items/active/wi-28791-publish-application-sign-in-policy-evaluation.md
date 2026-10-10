---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p3
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "アプリケーションのサインイン方針の評価を、Application の公開パッケージの経由へ変えるだけである。方針の判定、HTTP の応答、ドメインイベントは変えない。" }
---

# アプリケーションのサインイン方針の評価を Application の公開操作にする

## 動機

`tools/check/boundary-debt.json` の `private-import` のうち 2 件は、Authentication と OAuth2 の handler から `backend/application/usecases` への依存である。
OAuth2 の handler はサインイン方針の評価（`EvaluateSignInPolicy`、`EffectiveSignInRules`、`MfaEnrollmentPolicyFromRules`、`TrustedDeviceAllowedByRules` と判定の値）を、Authentication の handler は MFA の強制の開始日（`UpcomingMfaEnforcementStart`）を使う。

## 対象範囲

- サインイン方針の評価と MFA の強制の開始日を、Application の公開パッケージの操作として公開し、利用側を書き換える。
- 解消した違反 ID を台帳から消す。

## 対象外

- 方針の判定の変更。
- Application と Authentication、Application と OAuth2 の循環の解消。[依存の向きの決定](wi-21934-decide-module-dependency-order-and-break-cycles.md)で扱う。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D1、D3：サインイン方針の規則の所有者は Application である。利用側が必要とするのは評価の結果だけなので、評価の操作と結果の型を公開し、方針の保存先は公開しない。

## タスク

- [ ] T001 [App] 評価の操作と結果の型を公開し、利用側を書き換える。
- [ ] T002 [Tooling] 解消した違反 ID を台帳から消す。
- [ ] T003 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 評価の結果の型を移すときに、拒否と step-up の区別を取り違えると、方針を満たさないサインインを通す。
  利用側の拒否の振る舞いが規範 ID のテストで固定されているかを確かめてから書き換える。

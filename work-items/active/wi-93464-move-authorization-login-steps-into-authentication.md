---
status: pending
authors: [tn]
risk: high
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: [wi-87746-publish-authentication-operations-other-modules-use]
change_kind: refactor
spec_impact: { kind: none, reason: "認可エンドポイントの途中で行うログインと追加の認証の手順の実装を、OAuth2 の handler から Authentication へ移すだけである。手順の順序、各手順の判定、HTTP の応答、セッションの状態、ドメインイベントは変えない。" }
---

# 認可エンドポイントの対話の手順を Authentication へ移す

## 動機

`tools/check/boundary-debt.json` の `private-import` のうち 9 件は、`backend/oauth2/handlers_http` から Authentication の非公開パッケージへの依存である。
OAuth2 の handler は、認可要求の途中のログイン、パスワードの期限、MFA の登録、TOTP、WebAuthn、回復コード、信頼済みデバイスの各手順を、Authentication の九つの usecases と `webauthn/handlers_http` を直接呼んで編成している。

各手順の規則と状態は Authentication が持つので、Authentication の変更が OAuth2 の handler へ波及する。
Authentication と OAuth2 の共変更は 4 回で、共変更の組の上位にある。

## 対象範囲

- 認可要求の途中で行う認証の手順の実装を Authentication へ移す。
- OAuth2 には、認可要求を保持し、認証が完了したら要求を再開する責務だけを残す。
- 解消した違反 ID を台帳から消す。

## 対象外

- 手順の順序と各手順の判定の変更。
- サインイン方針の評価（Application への依存）。[Application のサインイン方針の公開](wi-28791-publish-application-sign-in-policy-evaluation.md)で扱う。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D1：各手順の規則と状態（チャレンジ、登録の途中の状態、試行の回数）の所有者は Authentication である。
- D8：OAuth2 と Authentication を独立して変えたい変更シナリオは、新しい認証方式の追加と、認可要求の形式の変更である。現状はどちらも `oauth2/handlers_http` を変える。

| 案 | 判断 |
| --- | --- |
| 手順の handler を Authentication へ移し、OAuth2 は認可要求の再開だけを受け持つ | 第一候補。新しい認証方式の追加が Authentication で閉じる |
| Authentication が手順ごとの公開操作を出し、OAuth2 が編成を続ける | 比較する。依存は公開パッケージへ移るが、手順の追加は引き続き OAuth2 を変える |

着手時に、認可要求の保持と再開の境界（どの値を Authentication へ渡し、何を受け取って再開するか）を決め、候補と問いを記録する。
認可の保証に関わる判断なので、決めきれない点は推測で確定せず利用者に確認する。

## タスク

- [ ] T001 [Design] 認可要求の保持と再開の境界を決め、案を比較する。
- [ ] T002 [Characterize] 移す手順の振る舞いのうち、規範 ID のテストが固定していないものを特性化テストで固定する。
- [ ] T003 [App] 手順の実装を移し、OAuth2 の handler を書き換える。
- [ ] T004 [Tooling] 解消した違反 ID を台帳から消す。
- [ ] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`
- `mise run test-ui-e2e`

## リスク

- 手順の移動で、追加の認証を経ずに認可要求が再開される経路が生じると、認証の保証が破れる。
  移す前に、認可エンドポイントを通した特性化テストで各手順の拒否と再開を固定する。

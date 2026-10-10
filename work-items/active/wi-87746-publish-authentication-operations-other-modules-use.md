---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p2
depends_on: []
change_kind: refactor
spec_impact: { kind: none, reason: "ほかのモジュールが使うセッション、パスワードの方針、step-up、信頼済みデバイス、ACR の判定を、Authentication の公開パッケージの経由へ変えるだけである。認証の判定、HTTP の応答、セッションの状態、ドメインイベントは変えない。" }
---

# ほかのモジュールが使う Authentication の判定と操作を公開する

## 動機

`tools/check/boundary-debt.json` の `private-import` のうち 10 件は、Application、IdManagement、Saml、WsFederation、Tenancy から Authentication の非公開パッケージへの依存である。

| 利用側 | 使っているもの |
| --- | --- |
| Saml、WsFederation、IdManagement の handler | セッションの解決（`SessionManager`、`SessionCookie`、`ErrSessionNotFound`） |
| IdManagement の usecases と handler、Tenancy の handler | パスワードの方針（`ValidatePasswordWith`、`ResolveTenantPolicy`、`PasswordPolicyError`、方針の上下限） |
| IdManagement の handler | step-up の判定（`StepUpSatisfied`、`ErrStepUpRequired`） |
| IdManagement の usecases、Application の usecases | 信頼済みデバイスの失効（`RevokeAllForUser`）と AMR の値 |
| Application の usecases | ACR と AMR の判定（`ACRSatisfies`、`ACRMFA`、`IsMfaAMR`） |

Authentication と IdManagement の共変更は 4 回で、共変更の組の上位にある。

## 対象範囲

- 上の表の判定と操作を、Authentication の公開パッケージとして公開し、利用側を書き換える。
- 解消した違反 ID を台帳から消す。

## 対象外

- 認可エンドポイントの対話の手順（`oauth2/handlers_http` からの依存）。[認可の対話の手順の移動](wi-93464-move-authorization-login-steps-into-authentication.md)で扱う。
- 認証の判定の変更と、Authentication の `internal/` への移動。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D1：セッション、パスワードの方針、step-up、信頼済みデバイス、ACR の規則の所有者は Authentication である。
- D3：利用側が使うのは判定の結果と少数の操作だけである。保存先、セッションの内部状態、方針の解決の途中の値は公開しない。

| 案 | 判断 |
| --- | --- |
| 判定と操作を公開パッケージにする | 採る |
| 方針の上下限を Tenancy へ写す | 採らない。同じ規則を二か所で保つことになる |
| `SessionManager` をそのまま公開する | 採らない。セッションの発行と失効まで利用側が呼べるようになる。利用側が必要とするのは解決と cookie の名前だけである |

Tenancy の handler からの依存は、Tenancy と Authentication の循環の辺でもある。
公開パッケージへの依存に変えても循環は残るので、辺の向きは[依存の向きの決定](wi-21934-decide-module-dependency-order-and-break-cycles.md)で決める。

## タスク

- [ ] T001 [Design] 公開する判定と操作の型を決める。
- [ ] T002 [App] 利用側を書き換える。
- [ ] T003 [Tooling] 解消した違反 ID を台帳から消す。
- [ ] T004 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- step-up とセッションの解決は認証の保証に関わる。
  書き換える前に、利用側の handler の拒否の振る舞いが `//spec:covers` のテストで固定されているかを確かめ、固定されていなければ特性化テストを先に置く。

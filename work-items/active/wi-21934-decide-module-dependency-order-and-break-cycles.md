---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p3
depends_on:
  - wi-35767-publish-claim-issuance-as-claimmapping-public-operations
  - wi-60465-publish-job-enqueue-and-handler-registration-as-jobs-ports
  - wi-87746-publish-authentication-operations-other-modules-use
  - wi-93464-move-authorization-login-steps-into-authentication
  - wi-13438-publish-oauth2-consent-revocation-and-client-administration
  - wi-93579-publish-idmanagement-user-operations-and-transactional-writer
  - wi-90942-relocate-saml-assertion-building-out-of-wsfederation
  - wi-28791-publish-application-sign-in-policy-evaluation
change_kind: refactor
spec_impact: { kind: none, reason: "モジュール間の依存の向きを決め、逆向きの依存を要求側のポートへ置き換えるだけである。HTTP の応答、認証と認可の判定、永続状態、ドメインイベントは変えない。" }
---

# モジュールの依存の向きを決めて循環を解く

## 動機

`tools/check/boundary-debt.json` の `module-cycle` は 57 件ある。
公開パッケージへの依存でも、二つのモジュールが互いに依存すれば循環は残る。
棚卸しの時点で、互いに直接依存する組は次の 13 組だった。

| 組 |
| --- |
| ApiTokens と OAuth2 |
| Application と Authentication |
| Application と OAuth2 |
| Application と Saml |
| Application と WsFederation |
| Audit と IdManagement |
| Audit と OAuth2 |
| Authentication と IdManagement |
| Authentication と OAuth2 |
| Authentication と Tenancy |
| IdManagement と OAuth2 |
| IdManagement と Sourcing |
| IdManagement と Tenancy |

残りの辺は、これらの組を通る長い循環に含まれる。
Tenancy と IdManagement の循環は、Tenancy が属性スキーマの型として IdManagement の domain を使うことによる。

## 対象範囲

- `private-import` の解消の項目の完了後に循環を再計測する。
- モジュールの依存の向きを決め、逆向きの辺ごとに、要求側がポートを定義して組み立て地点で結ぶ案と、担当を見直す案を比べる。
- 決めた向きを構造の文書へ書き、逆向きの辺を解消する。辺の数が多ければ、組ごとに解消の項目を起票する。
- 解消した違反 ID を台帳から消す。

## 対象外

- `private-import`、`shared-dependency`、`table-write` の解消。それぞれの項目で行う。

## 設計

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D4：逆向きの辺は、操作を要求する側が必要最小限のポートを定義し、実装する側が従い、組み立て地点で結ぶ案を比べる。実行時の再入、失敗時の順序、原子性を別に確かめる。
- D8：import を非循環にすることだけを理由にモジュールを分割しない。属性スキーマのように担当そのものが問われる組は、隠す判断と変更シナリオから担当を見直す。

依存の向きの候補は、テナントと署名鍵などの基盤のモジュールを下に、利用者と認証を中ほどに、プロトコルと管理の機能を上に置く順序である。
着手時の辺の実態から候補を確かめ、決めきれない担当の問いは利用者に確認する。

## タスク

- [ ] T001 [Inventory] 前提の項目の完了後に循環と辺を再計測する。
- [ ] T002 [Design] 依存の向きを決め、逆向きの辺ごとに案を比べる。
- [ ] T003 [Docs] 決めた向きを構造の文書へ書く。
- [ ] T004 [App] 逆向きの辺を解消するか、組ごとに解消の項目を起票する。
- [ ] T005 [Tooling] 解消した違反 ID を台帳から消す。
- [ ] T006 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run verify`

## リスク

- 逆向きの辺をポートへ置き換えると、実行時の呼び出しの循環（再入）は残ったまま import だけが消える。
  置き換えた辺ごとに、呼び出しの順序と失敗時の振る舞いを確かめる。

---
status: pending
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-10
priority: p3
depends_on:
  - wi-73604-let-shared-libraries-take-tenant-id-without-tenancy
  - wi-88544-let-table-owners-publish-transactional-writes
  - wi-35767-publish-claim-issuance-as-claimmapping-public-operations
  - wi-60465-publish-job-enqueue-and-handler-registration-as-jobs-ports
  - wi-87746-publish-authentication-operations-other-modules-use
  - wi-93464-move-authorization-login-steps-into-authentication
  - wi-13438-publish-oauth2-consent-revocation-and-client-administration
  - wi-93579-publish-idmanagement-user-operations-and-transactional-writer
  - wi-90942-relocate-saml-assertion-building-out-of-wsfederation
  - wi-28791-publish-application-sign-in-policy-evaluation
  - wi-61185-share-password-defaults-through-tenancy
  - wi-64444-read-user-source-through-an-identity-management-port
  - wi-49676-remove-audit-search-reverse-dependencies
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

### 前提の完了後に切り出した変更

2026-10-18 に wi-73604 と wi-88544 の完了を確認し、53件の循環を再計測した。
文脈の運搬と CSV のトランザクションは、もはや着手を妨げる前提ではない。
次の変更は製品の保証と所有者を変えずに独立して取り込めるため、別の項目で実装する。

| 項目 | 切り離す依存 | 判断 |
| --- | --- | --- |
| [wi-61185](../done/wi-61185-share-password-defaults-through-tenancy.md) | Tenancy → Authentication | パスワードの固定の既定値をテナント設定の型と同じ場所に置く。既定値4個のための注入ポートは増やさない |
| [wi-64444](../done/wi-64444-read-user-source-through-an-identity-management-port.md) | IdManagement → Sourcing | User と Group の由来表示は、保存型を受け取らず IdManagement の照会ポートを使う。Sourcing を含む循環の3件が不要になる |
| [wi-49676](../done/wi-49676-remove-audit-search-reverse-dependencies.md) | Audit → IdManagement・OAuth2 | username の解決は組み立て地点で Audit の関数型へ変換し、列挙値だけを仕様バインディングで共有する。Audit を含む循環の4件が不要になる |

これらの変更後の循環は45件であり、解消した8件だけを台帳から除く。
Sourcing → IdManagement・ApiTokens と、Authentication・IdManagement → Audit の import 自体は必要な非循環の依存として残る。

### 今回は一括して置き換えない範囲

残る45件はこの項目で引き続き扱う。
循環の件数だけを下げるために、次の担当と作用の問いを未解決のままポートを増やす変更は行わない。

| 対象 | 今は一括の置き換えを行わない理由 | 再開時に先に確定すること |
| --- | --- | --- |
| Tenancy と IdManagement の属性スキーマ | 設定の API と Repository の契約は Tenancy にあり、定義型、組み込み属性、保存先と動的 Group の参照は IdManagement にある。型を共有へ移すだけでは、どちらが不変条件を保つかが決まらない | REQ-TENANCY の属性スキーマと動的 Group の参照保護から、型、更新、参照保護の担当を一組で比較する。保存先の所有を動かす場合は移行と同じ Tx の要否も記録する |
| Application と OAuth2・Saml・WsFederation | Application の管理 API はプロトコルの設定と関連付けを一括して作り、失敗時の後始末も行う。Repository を写したポートを足すだけでは、その確定順と後始末の責任が残る | プロトコルごとに、作成・関連付け・失敗時の後始末を既存の成功と失敗のテストに照合し、編成を置く場所と公開する操作を比較する |
| Application と Authentication、Authentication と IdManagement・OAuth2、IdManagement と OAuth2 | サインインの再開、MFA の登録方針、同意、失効と完全削除が型と操作を共有する。個々の Repository 呼び出しを逆転するだけでは、再入と必須の作用を省ける経路を作りうる | 認証の再開と失効・削除を分けて操作列を列挙し、必要な作用を含む公開操作を選ぶ。完全削除では既存の再実行と各保存先の確定を保つ |
| ApiTokens と OAuth2 | API トークンの署名と検証に加え、OAuth2 の失効の組み込みクライアントの例外と Protected Resource Metadata のスコープ語彙が相互に参照する。業務のスコープ全体を shared へ移す案は D5 に適合しない | 管理発行トークンの操作と、組み込みリソースの公開情報・失効の認証を分け、後者を正式な組み立てで渡す案を比較する |
| Jobs・SigningKeys・ClaimMapping・SharedSignals・WorkloadIdentity を通る長い循環 | これらの辺だけを消しても、上の相互参照を原因とする循環の向きは決まらない。既存の非循環の依存まで削ると、必要のないポートが増える | 上の組を一つ解くたびに循環を再計測し、不要になった台帳の辺だけを除く。クォータ、鍵の対象テナントと失効の順序を import だけで判断しない |

これらは着手済みや解消済みとは扱わず、T002 以降に残す。
再開時は45件を固定した入力にせず、変更後の import と実行時の作用から再計測する。

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D4：逆向きの辺は、操作を要求する側が必要最小限のポートを定義し、実装する側が従い、組み立て地点で結ぶ案を比べる。実行時の再入、失敗時の順序、原子性を別に確かめる。
- D8：import を非循環にすることだけを理由にモジュールを分割しない。属性スキーマのように担当そのものが問われる組は、隠す判断と変更シナリオから担当を見直す。

依存の向きの候補は、テナントと署名鍵などの基盤のモジュールを下に、利用者と認証を中ほどに、プロトコルと管理の機能を上に置く順序である。
着手時の辺の実態から候補を確かめ、決めきれない担当の問いは利用者に確認する。

## タスク

- [x] T001 [Inventory] 前提の項目の完了後に循環と辺を再計測する。
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

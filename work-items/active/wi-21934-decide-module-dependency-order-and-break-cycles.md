---
status: in_progress
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

### 再調査前に残した担当と作用の問い

残る45件はこの項目で引き続き扱う。
循環の件数だけを下げるために、次の担当と作用の問いを未解決のままポートを増やす変更は行わない。

| 対象 | 今は一括の置き換えを行わない理由 | 再開時に先に確定すること |
| --- | --- | --- |
| Tenancy と IdManagement の属性スキーマ | 設定の API と Repository の契約は Tenancy にあり、定義型、組み込み属性、保存先と動的 Group の参照は IdManagement にある。型を共有へ移すだけでは、どちらが不変条件を保つかが決まらない | REQ-TENANCY の属性スキーマと動的 Group の参照保護から、型、更新、参照保護の担当を一組で比較する。保存先の所有を動かす場合は移行と同じ Tx の要否も記録する |
| Application と OAuth2・Saml・WsFederation | Application の管理 API はプロトコルの設定と関連付けを一括して作り、失敗時の後始末も行う。Repository を写したポートを足すだけでは、その確定順と後始末の責任が残る | プロトコルごとに、作成・関連付け・失敗時の後始末を既存の成功と失敗のテストに照合し、編成を置く場所と公開する操作を比較する |
| Application と Authentication、Authentication と IdManagement・OAuth2、IdManagement と OAuth2 | サインインの再開、MFA の登録方針、同意、失効と完全削除が型と操作を共有する。個々の Repository 呼び出しを逆転するだけでは、再入と必須の作用を省ける経路を作りうる | 認証の再開と失効・削除を分けて操作列を列挙し、必要な作用を含む公開操作を選ぶ。完全削除では既存の再実行と各保存先の確定を保つ |
| ApiTokens と OAuth2 | API トークンの署名と検証に加え、OAuth2 の失効の組み込みクライアントの例外と Protected Resource Metadata のスコープ語彙が相互に参照する。業務のスコープ全体を shared へ移す案は D5 に適合しない | 管理発行トークンの操作と、組み込みリソースの公開情報・失効の認証を分け、後者を正式な組み立てで渡す案を比較する |
| Jobs・SigningKeys・ClaimMapping・SharedSignals・WorkloadIdentity を通る長い循環 | これらの辺だけを消しても、上の相互参照を原因とする循環の向きは決まらない。既存の非循環の依存まで削ると、必要のないポートが増える | 上の組を一つ解くたびに循環を再計測し、不要になった台帳の辺だけを除く。クォータ、鍵の対象テナントと失効の順序を import だけで判断しない |

これらは、この段階では着手済みや解消済みと扱わなかった。
次節の再調査で向きと実装単位を決める。

[境界の負債の順位付け](../done/wi-33994-reinventory-the-remaining-boundary-debt.md)の判断を引き継ぐ。

- D4：逆向きの辺は、操作を要求する側が必要最小限のポートを定義し、実装する側が従い、組み立て地点で結ぶ案を比べる。実行時の再入、失敗時の順序、原子性を別に確かめる。
- D8：import を非循環にすることだけを理由にモジュールを分割しない。属性スキーマのように担当そのものが問われる組は、隠す判断と変更シナリオから担当を見直す。

### 確定した依存の向きと実装単位

2026-10-18 の `56de83f13` で、境界検査の `analyzeModuleBoundaries` により本番 import を再計測した。
依存は60辺、循環に含まれる辺は45件、直接の相互参照は9組である。
循環検出の入力はテストを除く Go ソースと責務表であり、台帳の件数を実測値の代わりに使わない。

以下の9辺を除いた依存には、Tenancy と Audit、Jobs と SigningKeys、IdManagement、ClaimMapping と WorkloadIdentity と SharedSignals、Application、Authentication、OAuth2 と Saml、ApiTokens と WsFederation と Authorization、残る利用側の順で、先に挙げた機能を依存先にする順序がある。
ここで先に挙げたものを依存先とし、同じ段の間には依存を作らない。
これは現在の辺に対する解消案の整合確認であり、モジュール別の固定階層や許可表を検査に追加する判断ではない。

| 除く辺と実装項目 | 入力と守る作用 | 候補の比較と採用理由 |
| --- | --- | --- |
| Tenancy → IdManagement：[wi-53317](wi-53317-decouple-tenant-attribute-schema-from-identity-types.md) | `manage_user_attribute_schema.go`、`admin_user_attribute_schema_handler.go`、REQ-TENANCY-002、020、043。スキーマの保存、定義の形式、組み込み属性、動的 Group の参照保護 | D1、D2、D4、D6。Tenancy のスキーマ表現と、IdManagement が実装する定義検証、組み込み属性照会、参照保護のポートを組み立て地点で結ぶ。保存は Tenancy、定義の意味は IdManagement に残す。全体を IdManagement へ移す案は管理 API と保存の担当を変え、共有へ定義を移す案は D5 に反するため採らない |
| Application → Authentication：[wi-51457](wi-51457-evaluate-application-policy-from-authentication-facts.md) | `domain/sign_in_policy.go`、`internal/usecases/access_gate.go`、REQ-APPLICATION-009、010、016。MFA、記憶済み端末、認証待ち、再認証時刻 | D1、D4、D6。Application が評価に使う認証事実の値を定義し、Authentication の ACR と AMR の意味を変換で保つ。ポリシーの担当を Authentication へ移す案は管理と評価を分散させ、AuthenticationContext を shared に移す案は業務の意味を漏らすため採らない |
| Application → OAuth2：[wi-60634](wi-60634-administer-oidc-applications-through-application-ports.md) | `application_provisioning.go`、`client_secret_lifecycle.go`、REQ-APPLICATION-001、002、005、006、007。クライアント作成、関連付け、設定、シークレット、削除 | D2、D4、D6、D7。Application が管理操作と表示のポートを定義し、OAuth2 の公開操作へ組み立て地点で変換する。Repository を写す案は保存型と不変条件を漏らし、管理 API 全体を OAuth2 へ移す案はカタログの編成を分散させるため採らない |
| Application → Saml：[wi-28013](wi-28013-administer-saml-applications-through-application-ports.md) | `application_provisioning.go`、REQ-APPLICATION-001、005、006、007。SP、IdP プロファイル、証明書検証、関連付け | D2、D4、D6、D7。Application の SP 管理ポートを Saml の公開操作で実装する。Repository の型だけを隠す案では証明書とプロファイルの検証が利用側へ残る。カタログの担当移動は採らない |
| Application → WsFederation：[wi-61016](wi-61016-administer-wsfed-applications-through-application-ports.md) | `application_provisioning.go`、REQ-APPLICATION-001、005、006、007。RP、reply URL、クレーム、関連付け | D2、D4、D6、D7。Application の RP 管理ポートを WsFederation の公開操作で実装する。Repository をそのまま公開する案とカタログの担当移動は上の2組と同じ理由で採らない |
| IdManagement → Authentication：[wi-18494](wi-18494-isolate-identity-credential-operations-behind-consumer-ports.md) | `AdminUserDeps`、`cascadeDeleteForSub`、`user_import_committer.go`、`account_handler.go`。User 作成、パスワード、履歴、MFA、セッション、完全削除 | D2、D4、D6、D7。IdManagement が必要な資格情報操作と本人照会のポートを定義する。CSV のトランザクション参加は既存の書き込み口へ変換し、確定条件を保つ。User のライフサイクルを Authentication へ移す案と、保存型を複製する Repository 群は採らない |
| IdManagement → OAuth2：[wi-62674](wi-62674-isolate-identity-oauth-lifecycle-and-account-queries.md) | `admin_users.go`、`admin_agents.go`、`account_data_handler.go`、REQ-IDMANAGEMENT-018、050、074、078。同意とトークンの削除、Agent とクライアントの束縛、本人エクスポート | D2、D4、D6、D7。IdManagement の要求側ポートに OAuth2 の公開操作を接続する。完全削除は既存の個別確定と PendingPurge の再実行を保ち、一括 Tx や非同期イベントへの変更は採らない。Agent の管理全体を OAuth2 へ移す案は主体の担当を割る |
| Authentication → OAuth2：[wi-75520](wi-75520-query-account-consents-through-authentication-ports.md) | `account_consents_handler.go`、`account_handlers_test.go`。本人の同意一覧と取り消し、ブラウザー要求検証、失効 | D2、D4、D6。Authentication が本人の同意管理のポートと表示値を定義し、OAuth2 の ConsentManager へ変換する。HTTP 経路の担当移動はルーターの責務まで変えるため採らない。認証再開は既に Authentication にあり、この辺の原因ではない |
| OAuth2 → ApiTokens：[wi-55762](wi-55762-inject-builtin-resource-metadata-and-revocation-policy.md) | `protected_resource_metadata.go`、`token_handler.go:handleRevoke`。管理 API の公開スコープと組み込み client の失効時の認証例外 | D1、D4、D5、D6。組み込みリソースの公開情報と失効用の識別情報を正式な組み立てから渡す。署名と検証を行う ApiTokens → OAuth2 は残す。業務スコープを shared へ移す案と認証例外の拡大は採らない |

各パスは対象モジュールの配下を指す。
新しい型やポートの最終シグネチャ、各保存失敗時の観測結果は、それぞれの実装項目で既存テストと HTTP 契約を照合してから確定する。
現状維持案は全組で循環を残すため採らない。
モジュールの分割と統合は、上の責務を変えずに解けるため採らない（D8）。

プロトコル設定を先に保存する現行コードには、後続の Application 作成失敗をすべて補償する処理はない。
動機段階の「失敗時の後始末」を実装済みの保証とみなさず、各項目で成功順序、途中の確定、拒否時の状態を照合する。
保証を強める補償処理や一括トランザクションは、このリファクタリングへ混ぜない。
User の使用量の照合は既存の [wi-83071](wi-83071-make-user-deletion-rerunnable-and-recount-quota-usage.md) が扱い、削除の再実行は既に存在する。

### 今回の採用と対象外

今回採用するのは、依存の向きの確定、所有文書への反映、9組の実装項目の起票である。
本番コード、公開契約、保存先、規範要件は変更しない。
先行3項目は実装済み、上の9組は今回起票、長い循環に含まれる残りの辺の独立した逆転は対象外とする。
9辺を除けば非循環になるため、Jobs、SigningKeys、ClaimMapping、SharedSignals、WorkloadIdentity へ追加のポートを作る理由はない。
保留する解消案はなく、すべての実装を名指した項目へ割り当てる。

台帳の45件は今回消さない。
起票は違反の解消ではなく、各実装で循環を再計測し、実際に消えた ID だけを削除する。

## 計画

1. 9組の入力、制約、候補の比較を確定する。
2. 独立して受け入れられる組ごとの実装項目を起票し、全体設計へ向きの基準と残存状態を書く。
3. Markdown の参照と形式、現行の境界と基準 revision からの負債増加を検査する。
4. 仕様差分を確認し、全文の正式コミットと縮約コミットを作る。

## タスク

- [x] T001 [Inventory] 前提の項目の完了後に循環と辺を再計測する。
- [x] T002 [Design] 9辺の逆向きを決め、各組の状態所有、要求側ポート、担当移動を D1〜D8 で比較した。
- [x] T003 [Docs] バックエンド設計に依存の向きを選ぶ基準を書き、既知リスクに実装の残存を記した。
- [x] T004 [App] 9組それぞれの解消を担う pending 項目を起票した。本番コードの変更は各項目で扱う。
- [x] T005 [Tooling] 今回は消えた違反がないため台帳の45件を保つ。各実装項目に再計測と消えた ID の削除を割り当てた。
- [x] T006 [Verify] 文書と境界に必要な検査を選び、規範仕様との差分を確認する。

## 検証

- `mise run check-work-items`：9組の起票と着手、完了の形式を検査する。
- `mise run check-links`：設計と作業項目の参照を検査する。
- `mise run check-boundaries`：現在の45件を残した台帳と実測の整合を検査する。
- `mise run check-boundary-debt-ratchet -- 56de83f13`：作業開始点から違反を増やしていないことを検査する。
- `mise run check-spec`、`mise run render-docs`、`mise run check-published-api-compat`：設計文書の形式と派生サイト、公開状態に応じた互換性を検査する。
- `mise run spec-diff`：規範の変更がないことを確認する。

最終差分は Markdown だけなので「振る舞いを保つ変更」として扱い、RED と変更耐性の検査は行わない。
製品のコードとテストを変更しないため、Go の通常テスト、race 検査、`verify` は選ばない。

## リスク

- 逆向きの辺をポートへ置き換えると、実行時の呼び出しの循環（再入）は残ったまま import だけが消える。
  置き換えた辺ごとに、呼び出しの順序と失敗時の振る舞いを確かめる。

## 要約

規範仕様の変更はない。
本番 import の60辺を再計測し、循環に含まれる45辺のうち逆向きの9辺を特定した。
状態の所有者を保ち、要求側の値とポートを組み立て地点で接続する方針をバックエンド設計へ反映し、9組の解消を wi-53317、wi-51457、wi-60634、wi-28013、wi-61016、wi-18494、wi-62674、wi-75520、wi-55762 に切り出した。
この9辺を除く残り51辺には非循環の順序があり、長い循環に含まれる基盤の辺を独立して逆転する必要はない。
本番コードは変更せず、45件の負債を維持した。
属性定義と保存の担当、プロトコル設定の途中確定、完全削除の再実行、本人の同意操作、組み込みトークンの失効の条件を、各実装で維持する境界として記した。

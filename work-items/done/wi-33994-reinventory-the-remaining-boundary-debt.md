---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-09
priority: p1
depends_on: [wi-17193-move-context-specific-code-out-of-support-http, wi-39119-publish-the-resolved-tenant-as-tenancy-public-language, wi-65906-inject-time-randomness-and-network-into-domain, wi-92970-adopt-information-hiding-modular-design]
change_kind: tooling
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: 境界の負債の台帳の理由を書き直し、解消の作業項目を起票するだけで、製品の振る舞いと公開の文書は変わらず、リリースの読者へ知らせることがない。
  references: []
initial_context:
  specification: []
  typespec: []
  source:
    - tools/check/boundary-debt.json
    - tools/check/src/boundary-fitness.ts
    - docs/design/application/design-guidelines.md
    - work-items/active/wi-97546-move-modules-without-private-callers-behind-go-internal.md
    - work-items/active/wi-75383-move-remaining-modules-behind-go-internal-and-retire-legacy.md
  tests: []
  stop_before_reading: [frontend, spec, docs/modules]
spec_impact: { kind: none, reason: "境界の負債の台帳の理由を書き直し、残りの解消を work item として起票するだけである。製品のコード、HTTP の応答、ドメインイベント、永続状態は変えない。" }
---

# 境界の負債を変更の局所性と開発コストで順位付けし、削減する作業へ分解する

## 動機

`tools/check/boundary-debt.json` の `private-import`、`module-cycle`、`shared-dependency` の項目の理由は、旧台帳から引き継いだ定型文か、移行のときに機械的に作った文である。
定型文の理由からは、どの依存をどう直すのかが読み取れず、解消の work item も起票されていない。

wi-17193、wi-39119、wi-65906 で、原因が一か所に集まった負債（`support_http` を経由した依存、Tenancy への依存、`domain` の作用）が消える。
残るのは、モジュールの組ごとに原因の異なる依存である。
現時点で多いのは、`private-import` の参照先の `backend/claimmapping/usecases`（11）と `backend/jobs/usecases`（9）、および 57 件の `module-cycle` である。

`table-write` の 7 件は、原子性の理由と参照を持つ具体的な理由を移行の時点で書いてある。
ただし、公開操作を別のトランザクションで呼ぶ形へ変えるだけでは原子性を保てないので解消にならない。

2026-10-10 の設計調査では、既存負債は合計 211 件だった。
これは調査時点の値であり、前提項目の完了後に再計測する。
負債の増加を拒否しても、既存の結合による探索範囲、変更の波及、業務規則を迂回できる公開範囲は残る。
件数だけでなく、繰り返し払う開発コストと守るべき保証から削減順を決める。

現在は未リリースであり、当面も未リリースを継続する。
旧版との互換性維持より、現在の設計を小さな文脈で変更できることを優先して p1 とする。
公開済み利用者がいるとは仮定せず、パッケージの変更では既存の製品仕様と原子性を維持する。

## 対象範囲

- 前提の 3 項目が完了した後の `mise run check-boundaries` の結果から、残った違反をモジュールの組ごとにまとめる。
- 組ごとに、[境界を選ぶ判断手順](../../docs/design/application/design-guidelines.md#境界を選ぶ判断手順)の D1〜D8 で直す方法を決める。
  非公開の処理への依存は D3、循環は D4、共有ライブラリからの依存は D5、変換の配置は D6、所有者外の書き込みは D2 と D7 を適用する。
- 直す方法と、それが必要な理由を、台帳の各項目の理由として書き直す。
- 直すための work item を、まとめて直せる単位で起票する。
- 変更頻度、共変更、参照する公開範囲、所有者外の書き込み、探索するモジュールの範囲から削減順と根拠を記録する。
- Repository や内部型を公開する案と、必要な業務操作だけを公開する案を比較する。
- 解消項目を既存の internal 移行項目へ対応付け、依存の解消と機械的な配置変更を重複起票しない。
- `table-write` の 7 件は、呼び出し側のトランザクションに参加する公開操作を所有者が公開する案（D2、D7）を比べ、採るなら解消の work item を起票する。

## 対象外

- 個々の依存と書き込みの解消そのもの。起票した work item で行う。
- 件数だけからの一律な分割と統合、マイクロサービス化。
- internal への配置変更そのものは[呼び出し側のないモジュールの移行](../active/wi-97546-move-modules-without-private-callers-behind-go-internal.md)と[残るモジュールの移行](../active/wi-75383-move-remaining-modules-behind-go-internal-and-retire-legacy.md)が扱う。

## 設計

前提項目が除く共通原因と、残るモジュールごとの原因を分ける。
依存の実態はコードと境界検査、共変更の履歴は mise run report-change-coupling から導き、手書きの依存グラフは作らない。
共変更の回数は調査候補の選択に使い、件数だけで境界を変更しない。
比較には履歴の対象期間、全体更新の除外条件、母数を添える。

各解消案は、隠す業務規則、必要な公開操作、変更が閉じる範囲、トランザクションの不変条件を説明する。
Repository を単に公開へ変えて違反を消す案や、所有者への呼び出しを別トランザクションにするだけの案は、保証を保つ根拠がなければ採らない。
本項目は順位付けと解消項目の起票を完了条件とし、配置変更や依存の削減が実施済みだとは報告しない。

### 再計測の結果

前提の 3 項目の完了後（`main` の 45110d5c4）の台帳は 137 件だった。

| 種類 | 件数 | 前提の完了前（2026-10-10 の調査） |
| --- | --- | --- |
| `private-import` | 57 | 111 |
| `module-cycle` | 57 | 57 |
| `shared-dependency` | 16 | 16 |
| `table-write` | 7 | 7 |
| `domain-effect` | 0 | 20 |

共変更は `mise run report-change-coupling` で測った。
条件は、first-parent で直近 500 コミット、6 個以上のモジュールを変えたコミット（10 件）を除外である。
母数は、モジュールを変えた 122 コミットである。
組の上位は IdManagement と Provisioning（6）、Authentication と IdManagement（4）、Authentication と OAuth2（4）、IdManagement と OAuth2（4）、IdManagement と Tenancy（4）だった。
共変更の回数は調査する組の選択だけに使い、境界の判断の根拠にはしない。

### 組ごとの原因と直し方

参照元のパッケージが使っているシンボルは、本番の `.go` の import の別名から抽出した。

| 解消の項目 | 対象の負債 | 原因（使っているもの） | 適用した判断基準と直し方 |
| --- | --- | --- | --- |
| [ClaimMapping の発行規則の公開](wi-35767-publish-claim-issuance-as-claimmapping-public-operations.md) | `private-import` 11（Application、OAuth2、Saml、WsFederation → ClaimMapping） | `IssueClaimsWithFloor`、`ResolveUserAttributes`、`ResolveTenantAttributeDefs`、`ValidateClaimReleaseRules`、`ClaimIssuanceResult`、`TenantAttributeSchemaRepo` | D1、D3。クレームの開示規則の所有者は ClaimMapping なので、四つのプロトコルが同じ規則を通るよう、発行と検証を公開操作として公開する。Repository を公開して各プロトコルに組ませる案は、開示の下限（floor）を迂回できるので採らない |
| [Jobs の投入と処理の登録の公開](wi-60465-publish-job-enqueue-and-handler-registration-as-jobs-ports.md) | `private-import` 9（DataKeys、IdGovernance、IdManagement、OAuth2、Provisioning → Jobs） | `Enqueue`、`EnqueueDeps`、`Handler`、`HandlerRegistry` | D3、D4。投入と処理の登録だけを Jobs の `ports` として公開し、重複排除と再試行の規則は Jobs に残す |
| [Authentication の公開操作](../active/wi-87746-publish-authentication-operations-other-modules-use.md) | `private-import` 10（Application、IdManagement、Saml、WsFederation、Tenancy → Authentication） | セッションの解決（`SessionManager`、`SessionCookie`、`ErrSessionNotFound`）、パスワードの方針（`ValidatePasswordWith`、`ResolveTenantPolicy`、方針の上下限）、step-up の判定、信頼済みデバイスの失効、ACR と AMR の判定 | D1、D3。規則の所有者は Authentication なので、利用側が必要とする判定と操作だけを公開する。保存先や内部の型は公開しない |
| [認可の対話の手順を Authentication へ移す](../active/wi-93464-move-authorization-login-steps-into-authentication.md) | `private-import` 9（`oauth2/handlers_http` → Authentication の各 usecases と `webauthn/handlers_http`） | 認可エンドポイントのログイン、MFA、TOTP、WebAuthn、回復コード、パスワードの期限、信頼済みデバイスの各手順 | D1、D8。各手順の規則と状態は Authentication が持つので、手順の実装を Authentication の handler へ移し、OAuth2 は認可要求の再開だけを受け持つ。OAuth2 が九つの usecases を直接編成する現状は、Authentication の変更が OAuth2 へ波及する主因である |
| [OAuth2 の同意、失効、クライアント管理の公開](../active/wi-13438-publish-oauth2-consent-revocation-and-client-administration.md) | `private-import` 6（Authentication、IdManagement、Application → OAuth2） | 同意の一覧と取り消し、セッション単位のトークン失効、管理用のクライアントの作成と秘密の発行 | D2、D3、D4。同意とクライアントの不変条件は OAuth2 に残し、操作を公開する。ログアウトでのトークン失効は Authentication が要求するポートとして定義し、組み立て地点で OAuth2 の実装を結ぶ |
| [IdManagement のユーザー操作の公開](../active/wi-93579-publish-idmanagement-user-operations-and-transactional-writer.md) | `private-import` 6（Authentication、IdGovernance → IdManagement） | プロファイルの取得、`ErrUserNotFound`、トランザクションに参加するユーザーの保存（`SaveUserTx`）、メモリーの保存先 | D2、D7。ユーザーの不変条件は IdManagement に残し、呼び出し側のトランザクションに参加する保存操作を公開する。wi-39119 の `QuotaRepositoryInTx` と同じ形を使う |
| [SAML のアサーション生成の置き場所](../active/wi-90942-relocate-saml-assertion-building-out-of-wsfederation.md) | `private-import` 4（Saml、Tenancy → WsFederation の `tokens_saml`） | `BuildAssertion`、`BuildSignedAssertion`、`Signer`、`SignerProvider` | D3、D5、D8。生成器は WsFederation の型とクレームの発行結果を使うので、そのままでは共有ライブラリの条件を満たさない。WsFederation の公開パッケージにする案と、プロトコルに依存しない部分を共有ライブラリへ分ける案を比べる |
| [Application のサインイン方針の公開](../active/wi-28791-publish-application-sign-in-policy-evaluation.md) | `private-import` 2（Authentication、OAuth2 → Application） | `EvaluateSignInPolicy`、`EffectiveSignInRules`、`UpcomingMfaEnforcementStart` など | D1、D3。サインイン方針の規則は Application が持つので、評価を公開操作にする |
| [共有ライブラリがテナント ID を Tenancy なしで受け取る](../active/wi-73604-let-shared-libraries-take-tenant-id-without-tenancy.md) | `shared-dependency` 5（`support_http`、`ratelimit/db_postgres`、`salts_memory`、`salts_postgres`、`storage/db_memory` → Tenancy） | 文脈からのテナント ID の取得とテナントの解決の middleware | D1、D5。共有ライブラリが必要とするのはテナント ID の文字列だけなので、引数で受け取るか、Tenancy の型を持たない受け渡しにする。テナントの解決の middleware は Tenancy へ移す |
| [JWT の署名器を OAuth2 へ移す](../active/wi-91505-move-the-jwt-token-signer-into-oauth2.md) | `shared-dependency` 5（`tokens_jose` → ClaimMapping、IdManagement、OAuth2、SigningKeys、Tenancy） | OAuth2 の ID トークンとアクセストークンの組み立て | D5、D8。OAuth2 の型とクレームの規則を使うので共有ライブラリの条件を満たさない。OAuth2 のパッケージへ移す |
| [固定データと観測の共有ライブラリ](../active/wi-26617-remove-module-types-from-fixtures-and-observability-libraries.md) | `shared-dependency` 6（`fixtures_postgres` → 4 モジュール、`metrics_prometheus` → Jobs、`sinks_console` → OAuth2） | モジュールの保存先と型を使う固定データの投入、Jobs と OAuth2 の型を名指す計測とイベントの出力 | D5、D6。固定データの投入は組み立て地点か Seeding へ移し、計測と出力はモジュールが提供する値だけを受け取る形にする |
| [所有者がトランザクションに参加する書き込みを公開する](../active/wi-88544-let-table-owners-publish-transactional-writes.md) | `table-write` 7（Application → OAuth2、Saml、WsFederation、IdManagement → Audit、Authentication、Jobs） | アプリケーションとプロトコル設定の結び付け、CSV 取り込みの監査、パスワード履歴、再評価のジョブ | D2、D7。各所有者が、呼び出し側のトランザクションに参加する書き込みを公開する。別のトランザクションで呼ぶ案は原子性を失うので採らない |
| [モジュールの依存の向きを決めて循環を解く](../active/wi-21934-decide-module-dependency-order-and-break-cycles.md) | `module-cycle` 57 | 相互に依存する 13 組（ApiTokens と OAuth2、Application と Authentication、Application と OAuth2、Application と Saml、Application と WsFederation、Audit と IdManagement、Audit と OAuth2、Authentication と IdManagement、Authentication と OAuth2、Authentication と Tenancy、IdManagement と OAuth2、IdManagement と Sourcing、IdManagement と Tenancy）と、それらを通る長い循環 | D4、D8。公開パッケージへの依存でも循環は残るので、ほかの項目の後に依存の向きを決め、逆向きの辺を要求側のポートへ置き換える |

### 削減の順序

| 順位 | 項目 | 根拠 |
| --- | --- | --- |
| 1 | Jobs の投入と処理の登録 | 6 モジュールが同じ二つの操作だけを使い、公開する範囲が小さい。IdManagement と Jobs、DataKeys と Jobs に共変更がある |
| 2 | ClaimMapping の発行規則 | 四つのプロトコルが開示の規則を使う。規則を迂回できる公開範囲を閉じる効果が大きい |
| 3 | Authentication の公開操作、認可の対話の手順 | 合わせて 19 件で最多。Authentication と OAuth2、Authentication と IdManagement の共変更が上位にあり、step-up の判定という認証の保証に関わる |
| 4 | JWT の署名器、共有ライブラリのテナント ID | 共有ライブラリの変更がモジュールの型の変更に引きずられる状態を解く |
| 5 | IdManagement のユーザー操作、所有者の書き込み | 原子性の保証を保ったまま、所有者外の書き込みを公開操作へ置き換える |
| 6 | OAuth2 の公開、SAML のアサーション生成、Application のサインイン方針、固定データと観測 | 件数が少なく、利用側が限られる |
| 7 | 依存の向きと循環 | ほかの項目が辺を減らした後でなければ、残る逆向きの辺を確定できない |

### 公開する範囲の比較

Repository や内部の型を公開して違反を消す案は、どの組でも採らない。
利用側が業務規則を迂回して状態を直接変えられるようになり、隠すべき判断が公開範囲へ漏れるためである。
各項目は、利用側が必要とする判定と操作だけを公開し、トランザクションの不変条件を保つ根拠を設計に書く。

### internal 移行との分担

依存の解消は本項目が起票した項目が行い、`internal/` への配置の変更は[呼び出し側のないモジュールの移行](../active/wi-97546-move-modules-without-private-callers-behind-go-internal.md)と[残るモジュールの移行](../active/wi-75383-move-remaining-modules-behind-go-internal-and-retire-legacy.md)が行う。
解消の項目は公開パッケージを決めて操作を移すところまでを受け持ち、残りのパッケージを `internal/` へ移す作業は含めない。
残るモジュールの移行の項目は、`private-import` の解消の 8 項目に依存する。

### 削減後に再評価する指標

- 種類ごとの台帳の件数（`mise run check-boundary-debt-ratchet -- main` の出力）。
- モジュールの組ごとの共変更（本節と同じ条件の `mise run report-change-coupling`）。
- 外から非公開パッケージへ import されるモジュールの数（再計測の時点は 7）。

## 計画

1. 前提項目の完了後に残る負債と参照元を再計測する。
2. 変更頻度、共変更、探索範囲と守るべき保証から優先する組を選ぶ。
3. D1〜D8 で候補を比較し、台帳の理由と解消項目へ反映する。
4. 既存の internal 移行との役割分担と依存を確認し、削減後に再評価する指標を記録する。

## タスク

- [x] T001 [Inventory] 残った違反をモジュールの組ごとにまとめ、変更頻度、共変更、探索範囲、原子性から削減順を決める。
  - 結果は設計の「再計測の結果」と「削減の順序」に記録した
- [x] T002 [Design] 公開する業務操作、隠す規則、変更の局所性を D1〜D8 で比較する。
  - 結果は設計の「組ごとの原因と直し方」と「公開する範囲の比較」に記録した
- [x] T003 [Tooling] 台帳の理由を書き直す。
  - 93 項目すべての理由を、原因、適用した判断基準、直し方、解消の項目の ID を含む文へ書き直した
  - 書き直す前は、解消の項目を名指す理由は 93 項目中 0 項目だった
- [x] T004 [Plan] 解消の work item を起票する。
  - 13 項目を起票した。`private-import` の解消の 8 項目を、[残るモジュールの移行](../active/wi-75383-move-remaining-modules-behind-go-internal-and-retire-legacy.md)の `depends_on` に足した
- [x] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-boundaries`
- `mise run check-boundary-debt-ratchet`
- `mise run report-change-coupling` の条件と母数を記録し、優先順位の根拠を説明する。
- すべての残る項目に、保証を維持する具体的な解消項目があり、既存の internal 移行と重複しない。
- `mise run check-work-items`

## リスク

- 理由を書き直すだけでは負債は減らない。
  この項目の完了の条件を、すべての残りの項目に解消の work item が対応していることにする。

## 完了

- **完了日**: 2026-10-10
- **要約**:
  `mise run spec-diff` は規範の差分なしを報告する。
  前提の 3 項目の完了後に境界の負債を再計測し、137 件（`private-import` 57、`module-cycle` 57、`shared-dependency` 16、`table-write` 7）を、原因の異なるモジュールの組ごとに 13 の解消の項目へ割り当てた。
  各組の直し方は D1〜D8 で決め、Repository や内部の型を公開する案と、所有者の操作を別のトランザクションで呼ぶ案は採らなかった。
  台帳の 93 項目すべての理由を、原因、直し方、解消の項目の ID が読み取れる文へ書き直した。
  削減の順序は、利用側の数、共変更、規則を迂回できる公開範囲、原子性の保証から決めた。
  依存と書き込みの削減そのものは行っておらず、負債の件数は変わらない。
- **受け入れ RED の証拠**:
  - **テスト**: N/A: 台帳の理由の書き直しと作業項目の起票であり、製品の振る舞いの境界がない。
  - **要件**: N/A: 規範上の振る舞いを変えず、対応する REQ がない。
  - **観測した失敗**: 代替の検査として、台帳の各項目の理由が実在する解消の項目の ID を名指すかを数えた。書き直す前の台帳（`main` の 45110d5c4）では 93 項目中 0 項目だった。
  - **検出できる理由**: 本項目の完了の条件（すべての残りの項目に解消の項目が対応すること）を直接数える。
- **単体 RED の証拠**:
  - **テスト**: N/A: 検査器のコードを変えていない。
  - **要件**: N/A: 規範上の振る舞いを変えず、対応する REQ がない。
  - **観測した失敗**: 受け入れ RED と同じ代替の検査である。書き直した後は 93 項目すべてが名指し、名指された 13 の ID はすべて `work-items/active/` に実在した。
  - **検出できる理由**: 理由の書き直しの漏れと、存在しない項目の名指しの両方を検出する。
- **変更耐性の結果**: N/A: 製品のコードと検査器を変えていないので、変異を加える対象がない。
- **検証結果**:
  - `mise run check-boundaries` - 成功
  - `mise run check-boundary-debt-ratchet -- main` - 成功（件数は変わらない）
  - `mise run check-work-items` - 成功
  - `mise run check-links` - 成功
  - `mise run report-change-coupling` - 条件と母数を設計の「再計測の結果」に記録した
  - `mise run verify` - 成功

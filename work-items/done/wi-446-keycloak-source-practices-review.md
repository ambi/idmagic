---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-08-30
priority: p2
depends_on: []
change_kind: docs
evidence_policy: risk-based-v2
initial_context:
  specification:
    - docs/contexts/authentication/scenarios.md#REQ-AUTHENTICATION-016
    - docs/contexts/identity-management/scenarios.md#REQ-IDMANAGEMENT-017
    - docs/contexts/system/scenarios.md#REQ-SYSTEM-001
    - docs/contexts/system/scenarios.md#REQ-SYSTEM-002
    - docs/contexts/system/scenarios.md#REQ-SYSTEM-016
    - docs/contexts/system/scenarios.md#REQ-SYSTEM-017
    - docs/api-rules.md
    - docs/deployment.md
    - docs/development/release.md
    - docs/development/specification-first-workflow.md
  source:
    - backend/authentication/password/ports/password_reset_token_store.go
    - backend/cmd/internal/bootstrap/config.go
    - backend/cmd/internal/bootstrap/configreference.go
    - backend/idmanagement/user/ports/email_change_token_store.go
    - backend/oauth2/authorization/usecases/authorize.go
    - backend/oauth2/client/domain/client.go
    - backend/shared/storage/fixtures_postgres/pgfixtures.go
    - backend/shared/storage/testing_postgres/pgtest.go
    - infra/k8s
  tests:
    - backend/cmd/internal/bootstrap/configreference_test.go
    - frontend/tests/e2e/fixtures.ts
    - frontend/tests/e2e/ui-scenario-actions.spec.ts
  stop_before_reading: [spec, frontend/src]
spec_impact:
  kind: none
  reason: Keycloak の実装と運用資材を一次資料から評価し、IdMagic へ導入する候補を選別する記録であり、仕様も実装も変更しない。
---

# Keycloak のソースコードと運用資材を評価し、IdMagic へ導入する改善を選別する

## Motivation

IdMagic は OAuth、OIDC、SAML、SCIM、認証、管理、監査、Kubernetes 運用を広く実装しているが、個々の機能が増えるにつれて、機能の成熟度、設定の正本、異なる版を混在させる更新、使い捨てリンク、プロトコル横断の安全策を一貫して扱う必要が高まる。長期運用されている Keycloak には、これらを個別機能ではなく開発・運用の仕組みとして扱う実装がある。一方、Java の SPI や realm 中心のモデルをそのまま移植すると、IdMagic の仕様優先、境界づけられた Context、型付きの作用境界を崩す。

本項目は、Keycloak の機能一覧を IdMagic のバックログへ転記するものではない。固定した公式ソースから再利用すべき設計原則と仕組みを抽出し、既に IdMagic が持つもの、個別の変更として起票すべきもの、採用しないものを区別する。

調査は Keycloak 公式 GitHub リポジトリの既定ブランチ `main`、コミット [`ae1a37058febedf3fe89e6ff3bc3b0f20176a43f`](https://github.com/keycloak/keycloak/commit/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f)（コミット日時 2026-08-28T18:48:48Z、確認日 2026-08-30）に固定した。引用先はこのコミットの permalink と Keycloak 公式文書だけである。

## Scope

- Keycloak の機能ライフサイクル、設定メタデータと文書生成、テスト資源と安定性検査、異版混在検査、Operator の状態と更新、クライアントポリシー、アクショントークンを調査する。
- 各仕組みについて、具体的なソース配置、仕組み、IdMagic へ導入する理由、導入候補、採用しない部分を記録する。
- 導入候補を、仕様・実装・運用の境界を混ぜない個別 work item へ切り出せる粒度と優先順位に整理する。
- 既存の IdMagic の仕組みで満たせている項目は、新しい抽象化を増やさず維持または局所的に強化する候補として分類する。

## Out of Scope

- 本項目内での TypeSpec、正準文書、Go、TypeScript、データベース、Kubernetes 資材の変更。
- Keycloak との機能同等性、移行互換性、性能比較、ベンチマーク、UI の外観比較。
- Keycloak の Java コード、Quarkus、Infinispan、Liquibase、Operator SDK、Maven、AsciiDoc/FreeMarker の採用。
- Keycloak の `server-spi` / `server-spi-private` に相当する汎用プラグイン機構の全面導入。
- 本調査から導く個別 work item の詳細設計。各変更が自身の仕様、受け入れ証拠、移行、後退条件を持つ。

## Design

### Evaluation Criteria

採用判断は「Keycloak に存在するか」ではなく、IdMagic で繰り返し起きる失敗を、単一の正本、閉じた型、観測可能な状態、実際の境界を通る検証によって減らせるかで行う。Keycloak 固有の実装技術は移植せず、IdMagic の TypeSpec、正準文書、Go の Context 境界、`mise` タスク、Kubernetes 資材へ翻訳できる仕組みだけを候補にする。

### Evidence Plan

- **Acceptance RED**: `wi-446` から採用対象の後続 work item を検索し、採用決定を実行可能な作業へ結び付ける参照がまだ存在しないことを確認する。製品挙動を変えない文書調査なので、規範シナリオに対する受け入れテストは N/A とする。
- **Unit RED**: `wi-446` に `Follow-up Decisions` が存在しないことを検索で確認する。変更する内部ロジックがないため Unit RED は N/A とし、候補ごとの所有範囲、変更種別、依存、再評価条件が未確定であることを代替検査とする。

### IdMagic Baseline

この表の「採用済み」は Keycloak と同じ構造を持つことではなく、候補が防ぐ失敗を IdMagic 固有の小さな仕組みですでに防いでいることを表す。「不足あり」は現行の正本または検査では防げない失敗が確認できた状態、「要件待ち」は仕組みを導入する根拠となる製品または運用要件をまだ確認できない状態である。

| Area | Classification | IdMagic Evidence | Baseline Decision |
|---|---|---|---|
| 機能ライフサイクル | **不足あり** | [`docs/api-rules.md`](../../docs/api-rules.md) は外部 API の `stable` と `beta`、非推奨化を定め、各 `standards.md` は標準の採否を持つ。一方、[`ProvisioningFeatureFlags`](../../backend/provisioning/domain/connection.go) は接続単位の操作許可に閉じており、製品全体の機能について成熟度、依存機能、既定の有効化、更新方針を結び付ける正本ではない。 | 文書上の API 安定性と標準採否は維持するが、それらを実行時機能の成熟度と同一視しない。製品横断の閉じた登録と配備前検証が未実装なので、独立した仕様変更が必要である。 |
| 設定メタデータ | **採用済み** | [`ConfigField`](../../backend/cmd/internal/bootstrap/config.go) はキー、型、既定値、制約、許容値、必須条件、秘密性を、実際に値を読む `ConfigLoader` から記録する。[`RenderConfigReference`](../../backend/cmd/internal/bootstrap/configreference.go) とその[テスト](../../backend/cmd/internal/bootstrap/configreference_test.go)は同じ記録から `CONFIGURATION.md` を生成し、説明の欠落と孤立を失敗にする。この経路は [[wi-103-startup-config-validation-and-reference]] で完了済みである。 | 「設定の正本から検証と文書を導く」という候補は満たしている。再起動影響、非推奨、関連設定は、それを必要とする設定変更が生じたときに同じ `ConfigField` を局所的に拡張し、別のメタデータ基盤は作らない。 |
| 機能と変更の文書責任 | **不足あり** | [`DOCUMENTATION_GUIDE.md`](../../DOCUMENTATION_GUIDE.md) と[仕様先行の開発ワークフロー](../../docs/development/specification-first-workflow.md)は正本の配置と同期を定め、[`docs/development/release.md`](../../docs/development/release.md) は版、検証、段階的展開を定める。しかし、リポジトリには `CHANGELOG` または更新ガイドがなく、`change_kind`、成熟度変更、破壊的変更、非推奨、削除から利用者向け告知と移行手順の要否を導く検査もない。 | 現在状態の正本を増やす必要はない。変更種別から、正準文書、変更履歴、更新手順のどれを更新すべきかを判定し、必要な利用者向け文書の欠落を失敗にする仕組みが必要である。 |
| 宣言的テスト資源 | **採用済み** | [`testing_postgres`](../../backend/shared/storage/testing_postgres/pgtest.go) は PostgreSQL の起動、スキーマ投入、パッケージ単位の寿命、隔離した実行領域、終了時清掃を所有し、[`pgfixtures`](../../backend/shared/storage/fixtures_postgres/pgfixtures.go) は決定的な時刻と共通の親資源を提供する。[`frontend/tests/e2e/fixtures.ts`](../../frontend/tests/e2e/fixtures.ts) は API、UI、コールバック、SMTP 受信器の起動、待機、清掃を一か所で所有する。PostgreSQL 側の共有化は [[wi-172-context-locality-pilot-application]] で完了済みである。 | 現在使う共有資源には所有者と寿命があり、Keycloak 型の依存グラフやテスト用 DI コンテナーを追加する根拠はない。新しい外部資源を複数のテスト群が共有するときも、現在の小さな型付き fixture を拡張する。 |
| 安定性と異版混在 | **不足あり** | [CI](../../.github/workflows/idmagic-ci.yaml) は race detector を含む通常の検証を 1 回実行するが、選択したテストを反復して一度の失敗を検出するタスクと、旧版と新版を同時に起動するタスクは無い。[`docs/development/release.md`](../../docs/development/release.md) は先行配備を定めるものの、リリース前に異版混在中のログイン、トークン更新、ジョブ引継ぎを検証しない。[[wi-131-testing-governance-and-ci-enforcement]] の夜間検査と [[wi-165-high-availability-and-failover-resilience-topology]] のゼロダウンタイム移行も未完了である。 | 不安定性の反復検査と異版混在検査のどちらも未実装であり、別々の検出目的として追加する必要がある。通常 CI の失敗を再試行で成功へ変える機能は含めない。 |
| 更新互換性 | **不足あり** | [`docs/development/release.md`](../../docs/development/release.md) は `check-api-compat` と同一コミットからの成果物生成を要求するが、旧配備の版と新配備の機能、設定、スキーマを読み取り専用で比較し、ローリング更新か再作成かを終了コードで返す検査はない。[[wi-165-high-availability-and-failover-resilience-topology]] は前後方互換と expand/contract 移行を予定するが未完了で、一般的な配備前判定までは対象にしていない。 | 公開 API の互換性検査は維持し、その外側にある実行時互換性を配備前に判定する仕組みを追加する必要がある。機能ライフサイクルと異版混在検査を入力または証拠として使う。 |
| Operator | **要件待ち** | [`docs/deployment.md`](../../docs/deployment.md) はクラウド製品に依存しない参照トポロジを定め、[`infra/k8s`](../../infra/k8s) は通常の `Deployment`、`CronJob`、`Service` などの静的資材を持つ。IdMagic 固有の CR、所有資源、継続的な再調整、status API を要求する現在の仕様はない。 | 配備前検査と静的資材で足りない reconciliation 要件が生じるまで Operator を起票しない。要件が生じた場合は、`observedGeneration` と conditions を含む status 契約から仕様化する。 |
| クライアントポリシー | **要件待ち** | [`OAuth2Client`](../../backend/oauth2/client/domain/client.go) は `RequirePushedAuthorizationRequests`、`DpopBoundAccessTokens`、閉じた `FapiProfile` を持ち、たとえば [Authorize ユースケース](../../backend/oauth2/authorization/usecases/authorize.go) はクライアント単位の PAR 要求を直接強制する。PAR と DPoP には個別の境界テストもあるが、現時点の調査では複数入口にまたがる同じ制約の重複または適用漏れを示す証拠を確認できなかった。 | 汎用ポリシーエンジンは導入しない。標準対応を追加して同じ制約を複数入口へ適用する際に、個別の閉じた判定では漏れを防げないことを Acceptance RED で示せた場合だけ再評価する。 |
| アクショントークン | **不足あり** | パスワード再設定の [`PasswordResetTokenStore`](../../backend/authentication/password/ports/password_reset_token_store.go) とメール変更の [`EmailChangeTokenStore`](../../backend/idmanagement/user/ports/email_change_token_store.go) は、ハッシュ、期限、単回消費を別々のほぼ同形の契約として持つ。両ユースケースはそれぞれリンクを発行して消費し、[E2E](../../frontend/tests/e2e/ui-scenario-actions.spec.ts) もあるが、目的を閉じた型で束縛する共通 envelope、作用との原子的な消費、先読みが作用を起こさない共通入口はない。 | 主要ユースケースは個別に検証済みでも、用途追加時に共通不変条件を再実装する構造が残る。用途別 payload と作用は各 Context に保ち、発行、検証、目的、期限、単回消費、監査の核だけを共有する仕様変更が必要である。 |

### Follow-up Decisions

| Candidate | Decision | Owner and Change Kind | Dependency or Revisit Condition |
|---|---|---|---|
| アクショントークン | [[wi-447-typed-action-token-core]] として採用する。目的、期限、単回消費、先読み安全性、作用との整合性だけを共通化し、用途別ペイロードと作用を各 Context に残す。 | Authentication と Identity Management、`feature`、`risk: high`。 | 先行依存はない。パスワード再設定とメール変更を一用途ずつ移行する。 |
| 機能ライフサイクル | [[wi-448-feature-lifecycle-and-update-policy]] として採用する。実行時に選択できる機能だけを閉じた登録簿にし、API 安定性と標準採否の正本を置き換えない。 | System、`feature`、`risk: medium`。 | 後続の更新互換性と文書ゲートがこの登録簿を入力にする。 |
| 更新互換性 | [[wi-449-deployment-update-compatibility-preflight]] として採用する。読み取り専用の決定表と安定した終了コードを提供し、配備自体は行わない。 | System operations、`operations`、`risk: high`。 | [[wi-448-feature-lifecycle-and-update-policy]] の機能版と更新方針を必要とする。 |
| 異版混在 | [[wi-450-mixed-version-release-acceptance]] として採用する。直前安定版と新成果物の版間契約を検査し、HA トポロジと障害訓練は [[wi-165-high-availability-and-failover-resilience-topology]] に残す。 | Release tooling、`tooling`、`risk: high`。 | [[wi-449-deployment-update-compatibility-preflight]] の `rolling_candidate` を動的に検証し、静的判定と合わせたリリース許可を所有する。`wi-165` から再利用できる境界にする。 |
| 安定性反復 | [[wi-451-stability-repetition-gate]] として採用する。一度の失敗を後続成功で上書きせず、低頻度失敗の仮説があるテストスイートだけを反復する。 | Test tooling、`tooling`、`risk: medium`。 | 独立した手動タスクまでを所有し、[[wi-131-testing-governance-and-ci-enforcement]] が夜間および必須検査への接続を所有する。[[wi-445-main-use-case-unit-and-e2e-evidence]] の主要ユースケース証拠は代替しない。 |
| 機能と変更の文書責任 | [[wi-452-feature-maturity-documentation-gates]] として採用する。現在状態、注目すべき差分、既存利用者の移行手順を別の文書責任として検査する。 | Development tooling、`tooling`、`risk: low`。 | [[wi-448-feature-lifecycle-and-update-policy]] の成熟度差分と [[wi-445-main-use-case-unit-and-e2e-evidence]] の証拠契約を入力にし、検証責任を重複させない。 |
| 設定メタデータ | 新しい work item を起票しない。[[wi-103-startup-config-validation-and-reference]] の `ConfigField` と生成器が目的を満たしている。 | 既存の System 設定境界。 | 更新影響、非推奨、関連設定を必要とする具体的な設定変更が生じたとき、その work item 内で `ConfigField` を局所的に拡張する。 |
| 宣言的テスト資源 | 新しい work item を起票しない。PostgreSQL と E2E の共有資源には所有者、寿命、清掃がある。 | 既存のテストフィクスチャ。 | 複数のテスト群が共有する新しい外部資源を導入し、既存の小さなフィクスチャで寿命を表現できない場合だけ再評価する。 |
| クライアントポリシー | 要件待ちとし、汎用ポリシーエンジンを採用しない。 | 現在は OAuth2 の用途別ドメインとユースケース。 | 同じ標準制約の適用漏れまたは重複を、複数の正式入口を通る Acceptance RED で示せた場合だけ、閉じた登録簿を起票する。 |
| Operator | 要件待ちとし、静的 Kubernetes 資材を維持する。 | 現在は System operations と `infra/k8s`。 | IdMagic 固有 CR の宣言状態、所有資源、継続的再調整、status API が必要になった場合だけ、`observedGeneration` と conditions を含む契約から起票する。 |

後続項目は Keycloak のコードを複製せず、固定コミットにある機構を IdMagic の型、Context 境界、`mise` タスクへ再設計する。将来ソースを複製する判断を行う場合は、その項目で Apache License 2.0 の帰属、ライセンス文、NOTICE の要否を先に確定する。

### Findings and Decisions

| Area | Keycloak Mechanism and Primary Sources | Decision for IdMagic | Adoption Candidate |
|---|---|---|---|
| 機能ライフサイクル | [`common/.../Profile.java`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/common/src/main/java/org/keycloak/common/Profile.java) は機能ごとに安定性（既定、既定無効、非推奨、プレビュー、実験的）、版、依存機能、利用可否、明示的な有効化、更新方針（ローリング可能、版変更時は停止、常に停止）を 1 つの列挙へ持たせる。同じ未版名の複数版を同時に有効化せず、依存機能が無効なら起動時に拒否し、非推奨・未安定機能をログで可視化する。 | **採用候補。** IdMagic は標準ごとの対応状況と API の安定性を文書化しているが、実行時に有効な機能の成熟度、依存、更新影響を横断して扱う正本は別の問題である。実験的な標準対応を既定有効にすること、必要な依存だけ無効な構成、ローリング更新できない変更を通常更新として流すことを機械的に拒否できる。 | 閉じた `Feature` 登録を設け、`id`、`version`、`maturity`、`default_enablement`、`dependencies`、`update_policy` を設定検証と配備前検査の入力にする。新しい仕様変更として別 work item にし、機能フラグ一般や任意コード読込みには広げない。 |
| 設定メタデータと文書 | [`quarkus/config-api/.../Option.java`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/quarkus/config-api/src/main/java/org/keycloak/config/Option.java) はキー、型、分類、非表示、ビルド時か実行時か、説明、既定値、許容値、大小文字、非推奨情報、関連オプションを保持する。[`docs/maven-plugin/.../Options.java`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/docs/maven-plugin/src/main/java/org/keycloak/guides/maven/Options.java) と [`docs/guides/GENERATE-DOCS.md`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/docs/guides/GENERATE-DOCS.md) はそのメタデータを設定リファレンスとガイドのテンプレートへ渡す。 | **原則は採用済み、局所強化候補。** IdMagic は構造化された設定定義から `CONFIGURATION.md` を生成し、差分を検査するため、別の文書生成系は不要である。Keycloak から取り入れる価値があるのは、設定の再起動・更新影響、非推奨、秘密性、関連設定を正本側へ増やし、生成文書と配備前検査へ同時に伝える考え方である。 | 現行の設定定義と生成器を唯一の正本のまま保ち、不足するメタデータを実需ごとに追加する。FreeMarker、AsciiDoc、Maven プラグインは採用しない。 |
| 機能と変更の文書責任 | [`docs/features.md`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/docs/features.md) は機能の成熟度ごとに、文書上の表示、試用上の注意、後方互換性、移行支援、廃止までの責任を定める。[`release_notes/topics/template.adoc`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/docs/documentation/release_notes/topics/template.adoc) は注目すべき新機能を扱い、[`upgrading/topics/changes/changes-template.adoc`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/docs/documentation/upgrading/topics/changes/changes-template.adoc) は破壊的変更、注目すべき変更、非推奨、削除を別の利用者行動として扱う。 | **考え方を採用候補。** IdMagic でも機能の説明、標準の対応状況、リリース時の要約、既存利用者が必要とする移行手順を同じ文章へ重複させず、変更の性質から更新すべき文書を決める必要がある。機能の成熟度を上げるゲートには、主要ユースケースの検証だけでなく、対象読者向け文書、互換性、移行、廃止予告の確認を含める。 | 現在の `docs/`、標準対応表、`docs/development/release.md` の責任分担を維持し、機能成熟度と変更種別から必要な文書更新を導く検査表または機械検査を別 work item で設計する。Keycloak の文書構成やテンプレート自体は移植しない。 |
| 宣言的テスト資源 | [`test-framework/docs/README.md`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/test-framework/docs/README.md) と [`BEST_PRACTICES.md`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/test-framework/docs/BEST_PRACTICES.md) は、テストが必要なサーバー、データベース、realm、client、user を宣言し、フレームワークが依存解決と class/method 単位のライフサイクル、汚染時の再作成、イベント専用 assertion を担う。実装は [`test-framework/core/.../DependencyGraphResolver.java`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/test-framework/core/src/main/java/org/keycloak/testframework/injection/DependencyGraphResolver.java) と [`ManagedTestResource.java`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/test-framework/core/src/main/java/org/keycloak/testframework/injection/ManagedTestResource.java) に分かれる。 | **限定採用候補。** IdMagic の単体/E2E 検証を独自フレームワークへ移さず、PostgreSQL、時刻、鍵、メール、外部 IdP、ブラウザーなど共有 fixture の所有者、寿命、清掃を宣言して、テストが暗黙の共有状態へ依存するのを防ぐ部分だけを取り入れる。主要ユースケースの単体/E2E 証拠は wi-445 の一般則に従い、本候補はその実行基盤に限る。 | まず現在の重複 fixture と汚染事故を棚卸しし、複数のテスト群が本当に共有する資源だけを型付き helper にする。テスト用 DI コンテナーや汎用依存グラフは導入しない。 |
| 安定性と異版混在 | [`.github/workflows/stability-base-reruns.yml`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/.github/workflows/stability-base-reruns.yml) は選択した統合テストを既定 50 回反復し、1 回でも失敗すれば失敗数で終了する。[`version-compatibility-matrix.yml`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/.github/workflows/version-compatibility-matrix.yml) と [`ci.yml`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/.github/workflows/ci.yml) は旧版と新しい版の組合せを生成して混在クラスタ検査へ渡す。 | **採用候補。** 通常の 1 回の GREEN は低頻度の競合を検出せず、単一版の E2E はローリング更新中のセッション、トークン、スキーマ、ジョブの互換性を証明しない。反復検査は再試行で成功扱いにするためではなく、不安定性を失敗として検出する専用タスクにする。異版混在はリリース成果物と直前安定版を同じ PostgreSQL と外部入口へ接続し、ログイン、トークン更新、ジョブ引継ぎ、起動・準備状態を確認する。 | 安定性反復タスクと混在版受け入れテストを別 work item にする。PR ごとの全件反復ではなく、並行性・時刻・分散状態を変更したテストと夜間検査から始める。再試行で通常 CI の失敗を隠さない。 |
| 更新互換性と Operator 状態 | [`docs/guides/server/update-compatibility.adoc`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/docs/guides/server/update-compatibility.adoc) は旧配備のメタデータと新しい版・設定を比較し、ローリング可、予期しない失敗、不正オプション、再作成必須を安定した終了コードで返す。利用者には内部メタデータを解析させない。[`operator/.../KeycloakStatus.java`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/operator/src/main/java/org/keycloak/operator/crds/v2beta1/deployment/KeycloakStatus.java) は `observedGeneration`、状態条件、準備済みインスタンス数、selector を CR status に持ち、[`UpdateLogicFactory.java`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/operator/src/main/java/org/keycloak/operator/update/UpdateLogicFactory.java) が更新戦略を選ぶ。 | **互換性判定を先に採用し、Operator は条件付きで先送り。** IdMagic に必要なのはまず、版、機能、設定、スキーマ移行から更新方式を決める読み取り専用の配備前ゲートである。独自 Operator は、継続的な reconciliation、CR の所有権、status API を必要とする要件が生じるまで保守費が勝る。将来 controller を持つなら `observedGeneration` と conditions を必須にし、「受理した設定」と「準備完了」を混同しない。 | `mise` から呼べる更新互換性検査を別 operations work item として設計する。入力メタデータは内部形式として版付けし、自動化は終了コードだけに依存させる。Operator の新設は含めない。 |
| クライアントポリシー | [`DefaultClientPolicyManager.java`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/services/src/main/java/org/keycloak/services/clientpolicy/DefaultClientPolicyManager.java) はプロトコルイベントごとに有効な policy を取り出し、条件の `YES` / `NO` / `ABSTAIN` を評価して、該当 profile の executor 群を実行する。strict policy は `ABSTAIN` も不成立として扱い、条件が無い policy は適用しない。 | **考え方を限定採用。** FAPI など複数入口へ同じ制約を適用する場合、イベント、対象条件、強制規則を閉じた型で登録し、適用漏れを一覧検査できる形は有効である。ただし、管理者が任意の条件・実行器を JSON で組む汎用エンジンや第三者 provider は、仕様で列挙した挙動より実行時設定が強くなり、検証可能性を下げる。 | OAuth2 の既存 FAPI・DPoP・PAR 制約の適用点を先に棚卸しし、実際の重複または漏れが確認された場合だけ、閉じた protocol-policy registry を別 work item で設計する。未知の条件は棄権ではなく拒否するフェイルクローズな意味を仕様化する。 |
| アクショントークン | [`ActionTokenHandler.java`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/services/src/main/java/org/keycloak/authentication/actiontoken/ActionTokenHandler.java) は token 型、追加 verifier、認証セッションへの参加または新規開始、イベント型、既定エラー、単回使用かを handler の契約にする。[`LoginActionsService.java`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/services/src/main/java/org/keycloak/services/resources/LoginActionsService.java) は共通入口で action id から handler を解決し、有効期限、issuer、署名、client、handler 固有条件、未使用を検証してから作用を実行し、`HEAD` はリンク検査で token を消費しない。 | **採用候補。** パスワード再設定、メール確認・変更、招待、管理者起点の必須操作は、発行者、対象、目的、期限、単回消費、セッション結合、監査、エラー秘匿の共通不変条件を持つ。用途ごとの handler は型付きのまま保ち、署名検証と単回消費を一つの核へ集約すれば、新しいリンク機能が不変条件を取りこぼしにくい。 | 既存の一時トークンを棚卸しし、共通 envelope、目的別 payload、原子的な単回消費、明示的作用、監査を別 feature work item で仕様化する。メールスキャナーの `HEAD` / 先読みが作用を起こさない E2E を主要ユースケースに含める。汎用 handler SPI や token だけで任意 required action を注入する仕組みにはしない。 |

### Adoption Order

1. **P0: アクショントークンの共通不変条件。** 既に複数のセキュリティ機微なリンク用途があり、用途追加のたびに署名・期限・単回消費・監査・先読み安全性を再実装するリスクが直接的である。
2. **P0: 機能ライフサイクルと更新方針の正本。** 実験的な標準対応と機能依存を既定値・配備方式まで一貫させ、後続の更新互換性判定の入力にする。
3. **P1: 異版混在テストと更新互換性ゲート。** 直前安定版と新しい版の実配線を検証し、ローリング更新か再作成かを自動化へ安定した終了コードで伝える。
4. **P1: 不安定性反復検査と共有 fixture の寿命。** wi-445 の主要ユースケース証拠を、共有状態の汚染や低頻度競合で空洞化させない。
5. **P1: 機能成熟度と変更種別に応じた文書責任。** 新機能の説明と既存利用者向けの移行情報を分け、成熟度変更、破壊的変更、非推奨、削除で必要な文書が欠けないゲートを設ける。
6. **P2: 設定メタデータの局所強化。** 現在の生成系を維持し、上の変更で必要になる更新影響、成熟度、非推奨、関連設定だけを加える。
7. **条件付き: 閉じたクライアントポリシー登録。** 既存制約の適用漏れまたは重複を棚卸しで確認できた場合だけ行う。
8. **先送り: Operator。** 配備前検査と既存 Kubernetes 資材では表現できない継続的 reconciliation の要件が生じた時点で再評価する。

### Rejected Approaches

- **SPI の全面導入。** Keycloak の [`ProviderFactory.java`](https://github.com/keycloak/keycloak/blob/ae1a37058febedf3fe89e6ff3bc3b0f20176a43f/server-spi/src/main/java/org/keycloak/provider/ProviderFactory.java) は provider の生成、順序、初期化、後処理を汎用化し、認証器など多くの拡張点は `server-spi-private` にも置かれる。IdMagic が同じ公開拡張面を持つと、Context 境界を越える任意コード、互換性を約束する API、動的な初期化順序、第三者コードの脅威モデルが新たに必要になる。必要な差し替えは現在どおり小さな型付き port と composition root で扱う。
- **Keycloak の成熟度名や既定値の丸写し。** 同じ標準でも IdMagic の実装証拠とサポート契約は別である。分類の軸を採用し、各機能の値は IdMagic 自身の仕様と検証から決める。
- **再試行で GREEN にする CI。** 反復実行は不安定性を検出するために使い、一度失敗したテストを成功扱いに変えない。
- **Operator を先に作ること。** controller は宣言状態、所有資源、再調整、status、アップグレード互換性の長期契約を生む。終了コードを返す読み取り専用の互換性検査で足りる段階では導入しない。
- **汎用ルールエンジンとしてのクライアントポリシー。** IdMagic がサポートする標準と拒否条件は TypeSpec と正準文書に閉じた語彙で定義し、未検証の実行器を実行時に追加させない。

## Plan

1. 現在の設定定義、実験的・未完了機能、一時トークン、FAPI 等の横断制約、テスト fixture、配備手順を上の候補へ対応付け、既に満たすものと不足を確認する。
2. P0 候補をそれぞれ独立した仕様変更 work item として起票する。アクショントークンと機能ライフサイクルは公開契約・セキュリティ境界が異なるため統合しない。
3. P1 の更新互換性、異版混在、安定性検査を operations/tooling の責任へ分ける。混在版の製品挙動が変わる場合は、先に対応する正準シナリオを更新する。
4. 機能成熟度と変更種別に応じた文書責任を既存の文書体系へ対応付け、重複したリリース文書を作らずに不足を検出する work item を起票する。
5. 設定メタデータとクライアントポリシーは、P0/P1 の実装で具体的な不足または重複が確認された範囲だけ起票する。
6. 各 follow-up は Keycloak のコードを移植せず、参照した設計原則、IdMagic 固有の型と境界、採用しなかった選択肢、主要ユースケースの単体/E2E RED 証拠を自身に記録する。

## Tasks

- [x] T001 [Research] 固定コミットの一次資料と IdMagic の現行実装を対応付け、各候補を「採用済み」「不足あり」「要件待ち」に確定する。
- [x] T002 [Decision] P0/P1 候補ごとに意味上の変更、所有 Context、変更種別、依存、再評価条件を確定する。
- [x] T003 [Work Items] アクショントークン、機能ライフサイクル、更新互換性・異版混在、安定性検査を、必要な単位へ分けて起票する。
- [x] T004 [Work Item] 機能成熟度と変更種別に応じた文書責任を既存の文書体系へ対応付け、不足する文書を検出する仕組みを起票する。
- [x] T005 [Decision] 設定メタデータと閉じたクライアントポリシー登録に実在する不足があるかを確認し、不足が無ければ起票しない理由を記録する。
- [x] T006 [Verify] すべての引用が固定 SHA または Keycloak 公式 URL を指し、採用・先送り・不採用の理由と再評価条件が揃っていることを検査する。

## Verification

- `mise run check-work-items`
- `mise run check-ids`
- 手動: すべての Keycloak の GitHub リンクがコミット `ae1a37058febedf3fe89e6ff3bc3b0f20176a43f` に固定され、主張するソース配置と仕組みを直接示す。
- 手動: 各導入候補が既存機能の重複起票になっていないこと、機能変更を docs 項目のまま実装しようとしていないことを確認する。

## Risk Notes

本項目自体のリスクは低く、仕様と実装を変更しない。最大のリスクは、大規模な Keycloak の構造を成熟した慣行という理由だけで模倣し、IdMagic に不要な抽象化と互換性責任を持ち込むことである。採用単位を失敗モードと検証可能な効果へ戻し、すべてを個別 work item で再設計する。

調査結果は固定コミット時点の事実である。Keycloak の分類や既定値を継続的な外部正本として追従せず、再評価する場合は新しいリリースタグまたはコミットを固定して差分を調べる。Keycloak の Apache License 2.0 は参照を妨げないが、本項目は設計評価に限定し、ソースを複製しない。

## Completion

- **Completed At**: 2026-08-30
- **Summary**:
  `mise run spec-diff` は `no normative specification change against main` を報告した。本項目は製品仕様を変更せず、Keycloak の固定コミットにある九候補を IdMagic の現状と対応付け、二件を採用済み、五件を不足あり、二件を要件待ちに確定した。不足は [[wi-447-typed-action-token-core]] から [[wi-452-feature-maturity-documentation-gates]] までの六項目へ分けた。
- **Acceptance RED Evidence**:
  - **Test**: `rg -n 'wi-447-typed-action-token-core|wi-448-feature-lifecycle-and-update-policy|wi-449-deployment-update-compatibility-preflight|wi-450-mixed-version-release-acceptance|wi-451-stability-repetition-gate|wi-452-feature-maturity-documentation-gates' work-items/wi-446-keycloak-source-practices-review.md`
  - **Requirement**: N/A: 製品挙動を変えない調査と work item の起票であり、規範シナリオに対する受け入れ境界はない。
  - **Observed Failure**: 終了コード 1 で一致がなく、採用対象を実行可能な後続項目へ結び付ける参照が存在しなかった。
  - **Detection Reason**: 候補の説明だけを増やして後続作業を一件も作らない不完全な調査では一致せず失敗する。完了時は `for id in wi-447-typed-action-token-core wi-448-feature-lifecycle-and-update-policy wi-449-deployment-update-compatibility-preflight wi-450-mixed-version-release-acceptance wi-451-stability-repetition-gate wi-452-feature-maturity-documentation-gates; do rg -n "\\[\\[${id}\\]\\]" work-items/done/wi-446-keycloak-source-practices-review.md || exit 1; done` により六参照を個別に必須化した。
- **Unit RED Evidence**:
  - **Test**: `rg -n '^### Follow-up Decisions$' work-items/wi-446-keycloak-source-practices-review.md`
  - **Requirement**: N/A: 変更する domain または use case の内部ロジックがない。
  - **Observed Failure**: 終了コード 1 で一致がなく、候補ごとの所有範囲、変更種別、依存、再評価条件を確定した表が存在しなかった。
  - **Detection Reason**: Keycloak の仕組みを列挙しただけの文書は `Follow-up Decisions` を持たず失敗する。完了時は `for candidate in アクショントークン 機能ライフサイクル 更新互換性 異版混在 安定性反復 機能と変更の文書責任 設定メタデータ 宣言的テスト資源 クライアントポリシー Operator; do rg -n "^\\| ${candidate} \\| (\\[\\[wi-|新しい work item|要件待ち)[^|]*\\|[^|]+\\|[^|]+\\|$" work-items/done/wi-446-keycloak-source-practices-review.md || exit 1; done` により九候補の十判断行に、判断、所有範囲、依存または再評価条件の各列があることを個別に必須化した。
- **Change-Resistance Results**:
  N/A: `risk: low` の文書調査であり、製品ロジック、検査ロジック、仕様を変更していない。代わりに、後続参照と最終判断表を別々に外した RED、固定 SHA 以外の Keycloak リンクを検出する検索、work item の依存とリンク検査を実行した。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run check-api-compat` - passed
  - `mise run spec-diff` - `no normative specification change against main`
  - `mise run check-work-items` - passed
  - `mise run check-ids` - passed
  - `mise run check-links` - passed
  - `mise run verify` - passed
  - 六件の後続参照を個別に必須化する完了検査 - passed
  - 九候補の十判断行と四列を個別に必須化する完了検査 - passed

## Left Undone

- [[wi-447-typed-action-token-core]] から [[wi-452-feature-maturity-documentation-gates]] の仕様変更と実装は、それぞれの証拠契約で行う。
- 設定メタデータと共有テストフィクスチャは現行の小さな仕組みを維持し、具体的な不足が生じるまで新しい基盤を作らない。
- Operator は継続的な再調整と状態契約の要件、クライアントポリシーは複数入口での適用漏れを示す Acceptance RED が生じるまで起票しない。

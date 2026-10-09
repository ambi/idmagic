# Application のアーキテクチャ

この文書は、Application の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

このモジュールがほかのモジュールと結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
このモジュールから外への依存の向きは次のとおりである。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `OAuth2`、`SAML`、`WS-Federation` | カタログが、プロトコル設定を作り、読み、`application_id` を設定する | 管理 API のハンドラーが、相手の Repository を使う |
| `ClaimMapping` | プロトコル設定の更新が、公開するクレームの規則を検査する | 相手の `ValidateClaimReleaseRules` と属性の定義の解決を呼ぶ |
| `IdManagement` | 割り当てが、主体の存在を確かめる | `SubjectDirectory` を相手が実装する |
| `Provisioning` | 割り当ての変化を下流へ伝える | `ProvisioningNotifier` を相手が実装する |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| 割り当てとポリシーの評価を、プロトコル設定の関連付けの確認と同じ関門に置く | 別のプロトコルを入口に選んで迂回させない |
| ポリシーを上書きで選び、一つの評価器で評価する | 実際に適用されるポリシーを一つの形で示す。詳細は[判断](decisions.md#テナントのデフォルトとアプリケーションごとのポリシーを合成せず上書きにする) |
| 一対一の関係をデータベースで保証する | 詳細は[判断](decisions.md#application-とプロトコル設定の関係を複合外部キーで保証する) |

## 構成要素

コードは機能スライスを持たず、一つの層の構成である。

| 機能仕様 | 主なユースケースとハンドラー |
| --- | --- |
| [アプリケーションのカタログ](../catalog/README.md) | `usecases/applications.go`、`usecases/categories.go`、`handlers_http/admin_application_handler.go`、`handlers_http/application_provisioning.go`、`handlers_http/client_secret_lifecycle.go` |
| [割り当て](../assignment/README.md) | `usecases/assignments.go`、`usecases/desired_state_assignments.go` |
| [サインインポリシー](../sign-in-policy/README.md) | `usecases/sign_in_policy.go` |
| [ポータルのアプリケーション](../portal/README.md) | `usecases/orderings.go`、`handlers_http/account_application_handler.go` |

| 層 | 責務 |
| --- | --- |
| `domain` | Application、割り当て、ポリシー、カテゴリ、並び順、ドメインイベント |
| `ports` | Repository、`SubjectDirectory`、`ProvisioningNotifier` |
| `usecases` | カタログ、割り当て、ポリシーの評価、並び順 |
| `handlers_http` | テナント単位の管理 API と、本人のアカウント API |
| `db_postgres`、`db_memory` | `ports` の PostgreSQL の実装と、テストとローカルの構成で使うメモリの実装 |
| `testing_contract` | 二つの実装が同じ契約を満たすことを確かめる共通のテスト |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| フェデレーションの関門 | OIDC の認可、SAML の SSO、WS-Fed のサインイン | 各プロトコルの処理が、発行の前にこのモジュールの評価を同期的に呼ぶ | [サインインポリシーの設計](../sign-in-policy/design.md) |
| あるべき状態の割り当て | `IdGovernance` のワークフローの手順 | `worker` が同期的に呼ぶ | [割り当て](../assignment/README.md) |
| 管理 API とアカウント API | 解決済みのテナントの要求 | `api` が、ハンドラーからユースケースを同期的に呼ぶ | 各機能仕様 |

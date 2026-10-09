# Provisioning のアーキテクチャ

この文書は、Provisioning の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

このモジュールがほかのモジュールと結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
このモジュールから外への依存の向きは次のとおりである。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `IdManagement` | タスクが User と Group の属性とメンバーを読み、インクリメンタル同期が User を列挙する | `source_idmanagement` のアダプターが相手の Repository を読む |
| `Application` | スコープの判定が割り当てを読む | 相手の割り当ての Repository を読む |
| `Jobs` | ディスパッチャーがタスクに Job を関連付け、`worker` がハンドラーを実行する | 相手の `JobRepository` と、Job の種類 `provisioning_task` を使う |
| `Tenancy` | ディスパッチャーが Job の数を上限で確かめる。隔離の通知がテンプレートを読む | 相手の `QuotaRepository` と通知の部品を使う |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| プロトコルに依存しない中核と、プロトコルごとの薄い機能に分ける | ドメインの形がほぼプロトコルに依存しない。詳細は[判断](decisions.md#プロトコルに依存しない中核とプロトコルごとの薄い機能に分ける) |
| 反映をインクリメンタル同期で保証し、イベント同期を近道にする | 取りこぼしの原因を問わずに回収する。詳細は[判断](decisions.md#下流への反映をインクリメンタル同期で保証する) |
| 内向きの SCIM と通信の中核を共有しない | 詳細は[判断](decisions.md#内向きの-scim-と通信の中核を共有しない) |

## 構成要素

コードは機能スライスを持たず、プロトコルに依存しない中核と、`ProvisioningTargetClient` を実装する `client_scim` からなる。

| 機能仕様 | 主なユースケースとハンドラー |
| --- | --- |
| [接続の管理](../connection/README.md) | `usecases/admin.go`、`usecases/full_resync.go`、`handlers_http/handlers.go` |
| [同期](../synchronization/README.md) | `usecases/capture.go`、`usecases/reconcile.go`、`usecases/dispatcher.go`、`domain/reconcile.go`、`domain/scheduled_deprovision.go` |
| [プロビジョニングタスクの実行](../task/README.md) | `usecases/execute_task.go`、`usecases/job_handler.go`、`client_scim` |

| 層 | 責務 |
| --- | --- |
| `domain` | 接続、リンク、タスク、予約、フル同期、属性の対応付け、インクリメンタル同期の計画、ドメインイベント |
| `ports` | Repository、イベント同期のポート、属性の取得、下流のクライアント |
| `usecases` | 管理、イベント同期、インクリメンタル同期、ディスパッチ、実行 |
| `client_scim` | 下流の SCIM 2.0 のクライアント、属性の対応付け、OAuth2 の資格情報 |
| `source_idmanagement` | IdManagement の User と Group の属性の取得 |
| `handlers_http` | テナント単位の管理 API |
| `db_postgres`、`db_memory` | `ports` の PostgreSQL の実装と、テストとローカルの構成で使うメモリの実装 |
| `testing_contract` | 二つの実装が同じ契約を満たすことを確かめる共通のテスト |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| イベント同期 | User と割り当ての変更の確定 | 変更したプロセスが、確定の後に同期的にタスクを作る | [同期の設計](../synchronization/design.md) |
| インクリメンタル同期 | `PROVISIONING_RECONCILE_INTERVAL` ごと | `worker` の周期のループ | [同期の設計](../synchronization/design.md) |
| ディスパッチ | 一定の間隔 | `worker` のディスパッチャーが、予約を実体化し、タスクに Job を関連付ける | [同期の設計](../synchronization/design.md) |
| タスクの実行 | `provisioning_task` の Job | `worker` の `default` のレーン | [プロビジョニングタスクの実行](../task/README.md) |
| 管理 API | 解決済みのテナントの管理 API への要求 | `api` が、ハンドラーからユースケースを同期的に呼ぶ | [接続の管理](../connection/README.md) |

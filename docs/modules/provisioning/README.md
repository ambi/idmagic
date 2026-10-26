# Provisioning

## 責務と境界

下流の SaaS へユーザーとグループを反映する、外向きのプロビジョニングを扱う。
情報の正は IdMagic の側の User と Group であり、下流のリソースはその複製である。
接続は Application 1 件につき最大 1 件とし、プロビジョニングの対象の範囲には既存の `ApplicationAssignment` を使う。

| 扱わないもの | 担当 |
| --- | --- |
| 外部からの取り込み | `Sourcing` |
| User と Group と割り当ての記録の正 | `IdManagement`、`Application` |
| タスクを実行する Job のキューと再試行 | `Jobs` |

IdManagement の User と Group、Application の割り当てを読み、下流の属性、無効化、削除への変換は Provisioning が所有する。
内向きの Sourcing とは処理の向き、記録の正、語彙が異なるので、`Tenancy`、`Application`、`IdManagement`、`Jobs` の公開するインターフェースを除いて、コードを共有しない。

## 主な流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| イベント同期 | User と割り当ての変更の確定 | 変更したプロセスが、確定の後に同期的にタスクを作る | [同期の設計](features/synchronization.md#実現方式) |
| インクリメンタル同期 | `PROVISIONING_RECONCILE_INTERVAL` ごと | `worker` の周期のループ | [同期の設計](features/synchronization.md#実現方式) |
| ディスパッチ | 一定の間隔 | `worker` のディスパッチャーが、予約を実体化し、タスクに Job を関連付ける | [同期の設計](features/synchronization.md#実現方式) |
| タスクの実行 | `provisioning_task` の Job | `worker` の `default` のレーン | [プロビジョニングタスクの実行](features/task.md) |
| 管理 API | 解決済みのテナントの管理 API への要求 | `api` が、ハンドラーからユースケースを同期的に呼ぶ | [接続の管理](features/connection.md) |

## 設計の要点

| 方針 | 理由 |
| --- | --- |
| プロトコルに依存しない中核と、プロトコルごとの薄い機能に分ける | ドメインの形がほぼプロトコルに依存しない。詳細は[判断](decisions.md#プロトコルに依存しない中核とプロトコルごとの薄い機能に分ける) |
| 反映をインクリメンタル同期で保証し、イベント同期を近道にする | 取りこぼしの原因を問わずに回収する。詳細は[判断](decisions.md#下流への反映をインクリメンタル同期で保証する) |
| 内向きの SCIM と通信の中核を共有しない | 詳細は[判断](decisions.md#内向きの-scim-と通信の中核を共有しない) |

## 機能一覧

| 機能 | 内容 |
| --- | --- |
| [接続の管理](features/connection.md) | 接続の登録と接続テスト、On-Demand Provision、フル同期、資格情報のローテーション |
| [同期](features/synchronization.md) | イベント同期、インクリメンタル同期、猶予期間つきの削除、誤削除ガード |
| [プロビジョニングタスクの実行](features/task.md) | 下流への反映、衝突と消失の解決、再試行と配信不能 |

## 文書

| 設計領域 | 文書 | 内容 |
| --- | --- | --- |
| アプリケーション設計 | [標準仕様](standards.md#system-for-cross-domain-identity-management-core-schema) | 管理 API はシステムのアプリケーション設計に従い、下流の形は標準仕様が定める |
| データ設計 | [Provisioning のデータ](data.md#集約) | Aggregate とその関係。テーブルの定義はスキーマファイルが扱う |
| セキュリティ設計 | [接続の管理](features/connection.md#セキュリティ上の考慮) | 下流の URL の検証とテナントの境界は接続の管理のセキュリティ上の考慮の節が扱う |
| 信頼性設計 | [同期の設計](features/synchronization.md#信頼性)、[接続の管理の設計](features/connection.md#信頼性) | 隔離からの回復と通知は同期の設計が、フル同期の完了は接続の管理の設計が扱う |

| 文書 | 内容 |
| --- | --- |
| [Provisioning の用語集](glossary.md) | このモジュールでの語義 |
| [Provisioning の標準仕様](standards.md) | 採用する外部標準仕様 |
| [設計判断](decisions.md) | 代替案を比べた重要な判断 |

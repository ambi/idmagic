# ランタイムアーキテクチャ

## 目的

この文書は、IdMagic を構成するプロセスと、その間の通信を示す。

## 実行単位

| 実行単位 | 入口 | 責務 | 状態 |
| --- | --- | --- | --- |
| API | `backend/cmd/idmagic` | OAuth2/OIDC、管理、アカウント、SCIM、Shared Signals などの HTTP リクエストを処理する | 正となる業務状態をプロセス内に保持しない |
| Worker | `backend/cmd/idmagic-worker` | キューに記録された遅延・再試行可能なジョブを処理する | PostgreSQL のジョブ状態を取得して確定する |
| Batch | `backend/cmd/idmagic-batch` | 保持期限に基づく削除など、横断的な保守処理を実行する | PostgreSQL の対象状態を更新する |
| Seed | `backend/cmd/idmagic-seed` | 初期データを投入する一回限りの管理実行単位 | PostgreSQL に結果を確定する |
| フロントエンドゲートウェイ | `frontend` | ブラウザー向け画面と同一オリジンの API 中継を提供する | ブラウザーセッション以外の正となる業務状態を保存しない |

API は複数のモジュールを一つのプロセスに組み立てる。
Worker と Batch は、長時間処理や再試行を HTTP リクエストから分離する。

## 実行単位が実装する機能

この節は、各実行単位が実装する機能の一覧と、詳細を定める文書を示す。
機能の振る舞いは所有するモジュールの文書が定め、ここには種類と参照先だけを置く。

### API

API は、API リファレンスに載るすべてのエンドポイントを、[API ガイドライン](../application/api-guidelines.md)に従って実装する。
API リファレンスは `spec/` の TypeSpec から生成し、`mise run render-docs` で生成する文書サイトの「API リファレンス」で読む。
各エンドポイントの振る舞いは、TypeSpec の操作が属するモジュールの[シナリオ](../../modules/README.md)が定める。

### Worker

Worker は、ジョブのハンドラーと、プロセス内で周期的に動く処理の二種類を実行する。
ハンドラーは `backend/cmd/idmagic-worker/worker.go` で `JobKind` ごとに登録する。
ジョブの取得、再試行、レーン、配信不能の扱いは [Jobs](../../modules/jobs/README.md) が定める。

| `JobKind` | 投入するモジュール | 処理 | 詳細 |
| --- | --- | --- | --- |
| `user_import_preview`、`user_import_apply` | IdManagement | 利用者の CSV インポートを検証し、確定する | [CSV の往復変換](../../modules/identity-management/design/csv-transfer.md) |
| `group_import_preview`、`group_import_apply` | IdManagement | グループの CSV インポートを検証し、確定する | [グループ CSV の設計](../../modules/identity-management/group-csv/design.md) |
| `group_membership_import_preview`、`group_membership_import_apply` | IdManagement | グループメンバーシップの CSV インポートを検証し、確定する | [グループ CSV の設計](../../modules/identity-management/group-csv/design.md) |
| `dynamic_group_reconcile` | IdManagement | 動的グループの規則を評価し、メンバーシップを収束させる | [動的グループ](../../modules/identity-management/dynamic-group/README.md) |
| `data_export` | IdManagement | 管理者が要求した CSV データエクスポートを作る | [データエクスポートの設計](../../modules/identity-management/data-export/design.md) |
| `data_key_reencryption` | DataKeys | DEK のローテーション後に、各モジュールの秘密情報を新しいバージョンの DEK で再暗号化する | [DEK のライフサイクルの設計](../../modules/data-keys/lifecycle/design.md) |
| `lifecycle_workflow_run` | IdGovernance | ライフサイクルワークフローを一回実行する | [IdGovernance](../../modules/identity-governance/README.md) |
| `provisioning_task` | Provisioning | 連携先のアプリケーションへ利用者とグループの変更を反映する | [Provisioning の内部設計](../../modules/provisioning/synchronization/design.md) |
| `backchannel_logout_delivery` | OAuth2 | OpenID Connect Back-Channel Logout の通知をクライアントへ送る | [ログアウト](../../modules/oauth2/logout/README.md) |
| `noop_echo` | Jobs | 入力をそのまま返す。Worker の起動と配線を確かめるためのジョブである | [永続キュー](../../modules/jobs/queue/README.md) |

周期的な処理は、ジョブのキューを通らずに Worker のプロセス内で動く。

| 処理 | 内容 | 詳細 |
| --- | --- | --- |
| ライフサイクルワークフローのディスパッチ | ジョブへ関連付けられていないワークフロー実行を探し、`lifecycle_workflow_run` のジョブを投入する | [IdGovernance](../../modules/identity-governance/README.md) |
| プロビジョニングのディスパッチ | 保留中のプロビジョニングタスクを `provisioning_task` のジョブへ関連付ける | [Provisioning の内部設計](../../modules/provisioning/synchronization/design.md) |
| プロビジョニングの照合 | イベント同期が作らなかった差分を探し、プロビジョニングタスクにする | [Provisioning の内部設計](../../modules/provisioning/synchronization/design.md) |
| 短命な状態の掃除 | 認可リクエスト、認可コード、デバイスコード、リプレイ防止、WebAuthn のセッション、流量制御など、期限を過ぎた短命な状態のレコードを削除して領域を回収する。有効期限は読み取り時に判定するので、掃除が遅れても期限は延びない | [データのライフサイクル](../data/lifecycle.md) |
| セキュリティイベントの配信 | Shared Signals の送信ストリームへ、配信期限が来たセキュリティイベントを送り、再試行と配信不能を管理する | [SharedSignals の状態遷移](../../modules/sharedsignals/stream/README.md#状態遷移) |
| キューの滞留数の記録 | レーンごとの待機中と実行中のジョブ数をメトリクスへ記録する | [監視設計](../observability/monitoring.md) |

### Batch

Batch は、`idmagic-batch <サブコマンド>` として一回ずつ起動する保守処理である。
定期実行の間隔は `infra/k8s/base/batch-cronjobs.yaml` の CronJob が一次情報である。

| サブコマンド | 処理 | 実行の契機 | 詳細 |
| --- | --- | --- | --- |
| `retention-sweep` | 保持期間を過ぎた監査イベント、認証イベントの集計、認証セッション、既知のサインイン端末、CSV の成果物を削除し、猶予期間を過ぎた削除予約の User を完全削除する | CronJob（毎時） | [データのライフサイクル](../data/lifecycle.md) |
| `signing-key-lifecycle` | 署名鍵の世代交代と、JWKS に古い鍵を残す猶予期間を管理する | CronJob（毎日） | [SigningKeys](../../modules/signing-keys/README.md) |
| `data-key-reencryption-sweep` | テナントと再暗号化の対象ごとに `data_key_reencryption` のジョブを投入する | 運用者が手で起動する。CronJob は宣言していない | [シークレット管理](../security/secrets.md) |
| `restore-consistency-check` | バックアップから復元したデータベースの件数、署名鍵、ジョブの重複を検査する | 復元手順の最後 | [バックアップ、復元、災害復旧の運用手順書](../../runbooks/backup-restore-dr.md) |

### Seed

Seed は、環境ごとの初期データを宣言したマニフェストを読み、計画して適用する。
`--mode dry_run` は計画だけを返し、`--mode apply` は各モジュールが公開するコマンドを呼んで適用する。
開発環境では `mise run seed -- <環境> <プロファイル>` で、Kubernetes では Job として起動する。

投入する初期データは、プロファイルごとに `seed/manifests/<プロファイル>.yaml` のマニフェストが宣言する。

| プロファイル | 投入する初期データ | 許される環境 |
| --- | --- | --- |
| `bootstrap` | 管理コンソールとマイページが使うファーストパーティーの OAuth クライアント | すべての環境。本番で許されるのはこのプロファイルだけである |
| `development` | `bootstrap` の内容に加え、デモ用の利用者（`alice`、`root`）、グループ、WS-Federation と SAML のデモ用アプリケーション | 本番以外 |
| `test` | `development` と同じ内容 | `test` 環境だけ |
| `performance` | `bootstrap` の内容に加え、負荷試験に使う大量の利用者。件数は `--count` で与える | 本番以外 |

マニフェストの文法、シークレットの参照、環境ごとの制限は [Seeding](../../modules/seeding/README.md) が定める。

### フロントエンドゲートウェイ

フロントエンドゲートウェイが配信する画面は、ログインと認証フロー、マイページ、管理コンソール、システム管理の四つに分かれる。
画面の一覧は[フロントエンド設計](../application/frontend.md#画面の一覧)が示す。

## 通信

ブラウザーはフロントエンドゲートウェイと通信し、フロントエンドゲートウェイが API へ HTTP リクエストを中継する。
ブラウザー Cookie を用いる画面と API は同一オリジンで公開する。
外部クライアントと上流の IdP は公開 HTTP エンドポイントへ到達し、API と Worker は PostgreSQL を共有する。

モジュール間の同期処理は、公開されたポートを `backend/cmd/internal/bootstrap` の組み立て地点で接続する。監査とセキュリティ通知に渡すドメインイベントは同じ組み立て地点の単一の配信点を通り、発行側と消費側を直接依存させない。公開するイベント語彙と互換性は [バックエンド設計](../application/backend.md#モジュール間イベント) で定める。

## 実行時の規則

- すべてのバックエンド実行単位は `backend/cmd/internal/bootstrap` の共通設定と依存注入を使い、その外で環境変数を直接読まない。
- API のレプリカ間で正となる状態を共有メモリに置かない。認証セッション、認可コード、ジョブ、スロットルなどの共有状態は PostgreSQL に置く。
- HTTP の生存性と準備状態は異なるプローブで表す。データベースへ到達できず新しいリクエストを安全に処理できない API は準備完了にしない。
- 停止時は新規要求の受け入れを止め、処理中のリクエストまたはジョブへ定めた猶予を与えてから終了する。

## 関連文書

- フロントエンドゲートウェイが配信する画面の構成と、中継する経路は[フロントエンド設計](../application/frontend.md)で設計する。
- 物理的なデプロイ構成と環境差は [デプロイメントアーキテクチャ](deployment.md) で定める。
- コンピューティングとストレージ、Kubernetes の方針は [プラットフォーム設計](../infrastructure/platform.md) で定める。
- 通信経路とネットワーク境界は [ネットワーク設計](../infrastructure/network.md) で定める。
- レプリカ配置と障害耐性は [可用性設計](../reliability/availability.md) で定める。
- 負荷、アドミッションコントロール、縮退は [性能設計](../performance/README.md) で扱う。
- 信頼境界と攻撃者モデルは [脅威モデル](../security/threat-model.md) で定める。

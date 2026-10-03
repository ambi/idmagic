# IdManagement のアーキテクチャ

この文書は、IdManagement の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

この Context がほかの Context と結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
依存の向きは次のとおりである。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `IdGovernance`、`Provisioning`、`Sourcing` | この Context が定めるポートを、相手が実装する | アプリケーションの組み立てで実装を注入する。この Context は相手のパッケージを参照しない |
| `Authentication`、`OAuth2`、`Authorization` | 相手がこの Context のポートを呼ぶ | 相手がこの Context の `ports` を参照する |
| `Tenancy` | この Context が相手のポートを呼ぶ | 属性スキーマとリソース上限を読む |
| `Jobs` | この Context が相手のポートを呼ぶ | CSV とデータエクスポートの非同期の実行を任せる |
| `OAuth2`、`Authentication` の記録 | この Context が相手のポートを呼ぶ | User の完全削除で、関連する記録を消す |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| `User`、`Group`、`Agent` をそれぞれ機能スライスに分ける | 一つの Aggregate の仕組みを一つのディレクトリの下で読めるようにする。詳細は[重要な設計判断](decisions.md)の「User、Group、Agent を機能ごとの縦割りのスライスで構成する」 |
| CSV のインポートを、プレビューと適用の二つの非同期ジョブにする | 適用の前に全行の検証結果を管理者に見せ、適用ではプレビューしたペイロードだけを実行する。詳細は[CSV の往復変換](csv-transfer.md) |
| User の削除を Tombstone にする | 追記専用の記録が参照する `sub` を壊さない。詳細は[判断](decisions.md#user-の削除を物理削除ではなく-tombstone-で行う) |
| 他の Context との協調をポートの注入で行う | この Context から下流の Context への依存を作らない |

## 構成要素

コードは、Aggregate root ごとの機能スライスと、スライスが共有するルートに分かれる。
各スライスの外への約束は、対応する機能仕様が定める。

| スライス | 責務 | 機能仕様 | 依存先 |
| --- | --- | --- | --- |
| `user` | `User` の作成、変更、ライフサイクル、属性、メールアドレスの変更、User の CSV | [ユーザー](../principals/user/README.md)、[アカウントのセルフサービス](../principals/account/README.md)、[ユーザー CSV](../bulk-transfer/user-csv/README.md) | ルート、`group`、`Tenancy`、`Jobs`、`Audit`、`OAuth2` と `Authentication` |
| `group` | `Group` の作成と変更、手動と動的のメンバーシップ、Group とメンバーシップの CSV | [グループ](../groups/group/README.md)、[動的グループ](../groups/dynamic-group/README.md)、[グループ CSV](../bulk-transfer/group-csv/README.md) | ルート、`user`、`Tenancy`、`Jobs` |
| `agent` | `Agent` の登録、資格情報のバインド、ライフサイクル | [エージェント](../principals/agent/README.md) | ルート、`user`、`Tenancy`、`OAuth2` |
| ルート | CSV の転送ポリシーと解析器、成果物ストア、データエクスポート、ロールの割り当ての判定、ドメインイベント | [データエクスポート](../bulk-transfer/data-export/README.md)、[ロール](../common/roles/README.md) | `Jobs` |

各スライスとルートは、同じ層で構成する。

| 層 | 責務 |
| --- | --- |
| `domain` | Aggregate、値オブジェクト、状態の検査、CSV の列の語彙 |
| `ports` | 永続化と、他の Context と結ぶ境界のインターフェース |
| `usecases` | 操作ごとの手順。認可済みの入力を受け、状態を検査し、保存し、イベントを発行する |
| `handlers_http` | 管理 API とセルフサービス API。認可、入力の変換、応答の型への変換 |
| `db_postgres`、`db_memory` | `ports` の PostgreSQL の実装と、テストとローカルの構成で使うメモリの実装 |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 管理 API とセルフサービス API | HTTP の要求 | `api` が、ハンドラーからユースケースを同期的に呼ぶ | 各機能の設計 |
| CSV のプレビューと適用 | 管理者の要求で `Jobs` のジョブを作る | `worker` | [CSV の往復変換](csv-transfer.md) |
| データエクスポート | 管理者の要求で、種類 `data_export` のジョブを作る | `worker` | [データエクスポート](../bulk-transfer/data-export/README.md) |
| ドメインイベントの発行 | 状態を変えた操作 | ユースケースが発行し、監査と下流へ渡す | [イベントと監査の記録](audit-events.md) |
| CSV の成果物の削除 | 外部のスケジューラーが Batch の `retention-sweep` を起動する | `batch` が、作成から 30 日を過ぎた成果物を消す | [CSV の転送](../bulk-transfer/csv-transfer/README.md#成果物の保持) |

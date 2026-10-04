# DataKeys のアーキテクチャ

この文書は、DataKeys の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に、鍵の種類ごとの扱いは[秘密情報の設計](../../../design/security/secrets.md)に従う。

## 文脈と範囲

この Context がほかの Context と結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
この Context から外への依存の向きは次のとおりである。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `Jobs` | ローテーションが再暗号化のジョブを投入し、`worker` がそのハンドラーを実行する | 相手の `ports.JobRepository` と、ジョブの種類 `data_key_reencryption` を使う |
| `Tenancy` | 健全性の一覧が、すべてのテナントを読む | 相手の `TenantRepository` を使う |
| 共有の部品 | DEK の生成、ラップ、アンラップ、項目の暗号化と復号 | `backend/shared/security/envelope_crypto` の `EnvelopeCrypto` と、マスターキーのプロバイダー |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| 暗号処理を共有アダプターに置き、この Context は鍵のライフサイクルだけを持つ | AEAD と鍵セットの処理を Tink へ委ねる。詳細は[判断](decisions.md#この-context-を-generic-に分類する) |
| 再暗号化と未移行の件数の数え方を、項目を持つ Context に委ねる | この Context が利用側のスキーマを知らないまま、安全に破棄できる。詳細は[判断](decisions.md#破棄の前の確認を項目を持つ-context-の-fieldmigrator-に委ねる) |
| アンラップした DEK をプロセスの中に保持する | 暗号化と復号のたびにマスターキーのプロバイダーを呼ばない |

## 構成要素

コードは機能スライスを持たず、一つの層の構成である。

| 機能仕様 | 主なユースケースとハンドラー |
| --- | --- |
| [DEK のライフサイクル](../lifecycle/README.md) | `usecases/lifecycle.go`、`usecases/reencrypt.go`、`usecases/migrator_registry.go`、`field_cipher.go` |
| [DEK の健全性の一覧](../health/README.md) | `usecases/tenant_data_key_health.go`、`handlers_http/admin_data_key_handler.go` |

| 層 | 責務 |
| --- | --- |
| `domain` | `TenantDataEncryptionKey`、状態、ドメインイベント、ライフサイクルのエラー |
| `ports` | `DataKeyRepository`、`CacheInvalidator`、`FieldMigrator` |
| `usecases` | ライフサイクルの操作、アンラップした DEK の保持、再暗号化のジョブのハンドラー、健全性の一覧 |
| ルートの `field_cipher.go` | 項目を持つ Context が使う暗号化と復号の入口。初回の暗号化で DEK を生成する |
| `handlers_http` | 制御面の健全性の一覧 |
| `db_postgres`、`db_memory` | `ports` の PostgreSQL の実装と、テストとローカルの構成で使うメモリの実装 |
| `testing_contract` | 二つの実装が同じ契約を満たすことを確かめる共通のテスト |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 項目の暗号化と復号 | 項目を持つ Context の保存と読み取り | 呼び出し元のプロセスが `FieldCipher` を同期的に呼ぶ | [DataKeys のセキュリティ](security.md) |
| 再暗号化 | ローテーションと、`idmagic-batch data-key-reencryption-sweep` | `worker` が `data_key_reencryption` のジョブを実行する | [DEK のライフサイクルの設計](../lifecycle/design.md) |
| 健全性の一覧 | 制御面の管理 API への要求 | `api` が、`RequireControlPlaneUser` で制御面主体を確かめてからユースケースを呼ぶ | [DEK の健全性の一覧](../health/README.md) |
| ローテーション、無効化、破棄 | なし | 呼び出す経路がない | [DataKeys のリスク](risks.md) |

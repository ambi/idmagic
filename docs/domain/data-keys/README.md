# DataKeys

## 責務と境界

テナントごとの `DataEncryptionKey`（DEK）のライフサイクル（生成、ローテーション、無効化、破棄）とメタデータを扱う。
DEK はマスターキーのプロバイダーでラップしてデータベースに保存し、データベースに保存する可逆なシークレット（TOTP の seed など）のエンベロープ暗号化に使う。
マスターキーのプロバイダーには OpenBao Transit 互換の実装を使い、開発環境とローカル環境では Tink の平文の鍵セットも使える。

| 扱わないもの | 担当 |
| --- | --- |
| 暗号処理そのもの | 技術的な共有アダプター `backend/shared/security` の `EnvelopeCrypto` ポートとその実装 |
| 暗号化する項目を持つレコードと、その再暗号化の手順 | 項目を持つ各 Context。`FieldMigrator` として登録する |
| 署名鍵（`private_jwk`） | `SigningKeys` |
| 再暗号化のジョブの実行 | `Jobs` |

この Context が外部へ公開するのは、項目の暗号化と復号の入口と、鍵のライフサイクルのメタデータだけである。

## モデル

| Aggregate | 境界の内側にあるもの | 他の Aggregate との関係 |
| --- | --- | --- |
| `TenantDataEncryptionKey` | テナントとバージョンの組ごとの `wrapped_dek`、マスターキーの ID、状態、有効化、退役、無効化、破棄の時刻 | `Tenant` を `tenant_id` で参照する |

テナントごとに、新規の暗号化に使える DEK は常に高々 1 本である。
復号できるバージョンは複数ありうる。
暗号文は、暗号化に使った DEK のバージョンとともに、項目を持つ Context が保存する。

## 公開する契約

HTTP の操作とモデルの形は TypeSpec の `Data Keys` のタグが定める。
次の表は、それ以外にほかの Context と結ぶ契約である。

| 契約 | 相手 | 向き | 内容 |
| --- | --- | --- | --- |
| `FieldCipher` の `Encrypt` と `Decrypt` | 可逆なシークレットを保存する Context（`Authentication` の TOTP と外部 IdP の接続） | この Context が提供する | テナントの DEK で一つの項目を暗号化し、使ったバージョンを返す。保存したバージョンで復号する |
| `FieldMigrator` | 暗号化した項目を持つ Context が実装し、起動処理が登録する | この Context が定める | 未移行の件数の報告と、アクティブなバージョンへの一括の再暗号化 |
| `data_key_reencryption` のジョブ | `Jobs` が実行する | この Context が定める | テナントと `FieldMigrator` の組ごとに再暗号化を進める |
| ドメインイベント | 監査と下流が購読する | この Context が発行する | `DataEncryptionKeyBootstrapped`、`DataEncryptionKeyRotated`、`DataEncryptionKeyDisabled`、`DataEncryptionKeyDestroyed` |

## 機能

| 機能 | 内容 |
| --- | --- |
| [DEK のライフサイクル](lifecycle/README.md) | 生成、ローテーション、無効化、破棄と、破棄の前の再暗号化 |
| [DEK の健全性の一覧](health/README.md) | 制御面主体による、テナントを横断した DEK の状態の一覧 |

| 文書 | 内容 |
| --- | --- |
| [DataKeys の用語集](glossary.md) | この Context での語義 |
| [DataKeys の設計](design/README.md) | 話題ごとの設計と重要な判断 |

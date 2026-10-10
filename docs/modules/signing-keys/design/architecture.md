# SigningKeys のアーキテクチャ

この文書は、SigningKeys の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に、鍵の種類ごとの扱いは[秘密情報の設計](../../../design/security/secrets.md)に従う。

## 文脈と範囲

このモジュールがほかのモジュールと結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
このモジュールから外への依存の向きは次のとおりである。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `Tenancy` | 健全性の一覧と定期実行のバッチが、すべてのテナントを読む | 相手の `TenantRepository` を使う |
| Vault | `VaultTransit` の提供元が、署名と鍵の作成を委ねる | `keys_vault` が Transit の HTTP API を呼ぶ |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| 差し替えられる `KeyProvider` の背後で、テナントごとに鍵を分離する | 全テナントでの鍵の共有と、プロトコルのアダプターへの提供元の埋め込みを避ける。詳細は[判断](decisions.md#署名鍵を差し替えられる-keyprovider-の背後でテナントごとに分離する) |
| 署名の処理はライブラリに委ね、鍵のライフサイクルと分離だけを持つ | 確立した実装がある処理は自作しない（[設計ガイドライン](../../../design/application/design-guidelines.md#確立した実装への委譲)） |
| 鍵を使う時点で作る | すべてのテナントを同じ手順で扱い、デフォルトテナントに固有の状態を設けない |

## 構成要素

コードは機能スライスを持たず、一つの層の構成である。
機能仕様に対応するコードは、要件の ID を `//spec:covers` で名指すテストからたどる。

| 層 | 責務 |
| --- | --- |
| `domain` | `SigningKey`、`KeyUsage`、ドメインイベント |
| `ports` | `KeyStore`、用途とスコープの文脈への載せ方 |
| `usecases` | ローテーション、期限切れの鍵のアーカイブ、健全性の一覧 |
| `keys_jose` | RSA の JWK の生成と取り込み、自己署名の X.509 証明書 |
| `keys_memory`、`db_postgres`、`keys_vault` | `Local`、`Database`、`VaultTransit` の提供元の `KeyStore` の実装 |
| `handlers_http` | テナント単位の鍵の管理 API と、制御面の健全性の一覧 |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 署名と公開鍵の一覧 | トークン、Assertion、JWKS、メタデータの要求 | 要求を受けたプロセスが、要求のテナント、用途、スコープの鍵を同期的に読む | [テナントと用途による鍵の分離](../separation/README.md) |
| 管理者によるローテーションと無効化 | `/api/admin/v1/...` の鍵の管理 API への要求 | `api` が、ハンドラーから同期的に行う | [署名鍵のライフサイクル](../lifecycle/README.md) |
| 定期のローテーションとアーカイブ | 外部のスケジューラーによる `idmagic-batch signing-key-lifecycle` | バッチのプロセスが、テナントごとに順に行う | [SigningKeys のリスク](risks.md) |

# ApiTokens のアーキテクチャ

この文書は、ApiTokens の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

このモジュールがほかのモジュールと結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
このモジュールから外への依存の向きは次のとおりである。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `OAuth2` | 発行が JWT を作り、認証が署名と期限を検証する | 相手の `ports.TokenIssuer` と `ports.TokenIntrospector` を使う |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| 通常の OAuth アクセストークンと同じ RFC 9068 の JWT を使う | 署名の鍵、イントロスペクション、失効の経路を `OAuth2` と共有する |
| 署名と管理のレコードの両方を検証する | 失効を即時に効かせ、偽造された `jti` も拒否する。詳細は[判断](decisions.md#署名と管理のレコードの二段でトークンを検証する) |

## 構成要素

コードは機能スライスを持たず、一つの層の構成である。

| 機能仕様 | 主なユースケースとハンドラー |
| --- | --- |
| [API アクセストークンの発行と失効](../token-management/README.md) | `usecases/usecases.go` の `Issue`、`List`、`Revoke`、`handlers_http/routes.go` |
| [API アクセストークンの認証と認可](../authentication/README.md) | `usecases/usecases.go` の `Authenticate`、`AuthenticateClaims`、`IntrospectAccessToken` |

| 層 | 責務 |
| --- | --- |
| `domain` | `ApiToken`、スコープの語彙と解析、`Principal` |
| `ports` | `Repository` |
| `usecases` | 発行、一覧、失効、認証、イントロスペクション |
| `handlers_http` | テナント単位の管理 API |
| `db_postgres`、`db_memory` | `ports` の PostgreSQL の実装と、テストとローカルの構成で使うメモリの実装 |
| `testing_contract` | 二つの実装が同じ契約を満たすことを確かめる共通のテスト |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 発行、一覧、失効 | 対話のセッションからの管理 API への要求 | `api` が、ハンドラーからユースケースを同期的に呼ぶ | [API アクセストークンの発行と失効](../token-management/README.md) |
| 認証と認可 | `Authorization` ヘッダーにトークンを付けた管理 API と SCIM API への要求 | `api` の認証の部品が、ハンドラーより先に認証し、操作の認可でスコープとロールを確かめる | [API アクセストークンの認証と認可](../authentication/README.md) |

# Tenancy のアーキテクチャ

この文書は、Tenancy の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

このモジュールがほかのモジュールと結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
このモジュールから外への依存の向きは次のとおりである。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `IdManagement` | このモジュールの属性スキーマが、相手の属性定義の型を使う。ユーザー属性スキーマの更新は、相手の動的グループの規則を読む | 相手の `user` と `group` の `domain` と `ports` を参照する。属性スキーマの永続化は相手が実装する |
| `Authentication` | テナント設定の取得が、パスワードポリシーのデフォルト値を読む | 相手のパスワードのユースケースを参照する |
| `WS-Federation` | 連携エンドポイントが、署名証明書の情報を組み立てる | 相手の SAML トークンの部品を参照する |
| 共有の部品 | 通知の描画と送信、画像の形式の検証、テナント解決のミドルウェア | `backend/shared/notification`、`backend/shared/mediavalidation`、`backend/shared/http/support_http` |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| テナントの識別子を UUID と realm に分ける | realm の改名で外部キーに触れない。詳細は[判断](decisions.md#テナントのキーを不変な-uuid-と可変な-realm-に分ける) |
| 一つのテナントに一つの正規ロケーションと一つの発行者を対応させる | 発行者と Discovery Metadata を標準に合わせる。詳細は[判断](decisions.md#テナントの正規ロケーションを一つにする) |
| 外装と属性スキーマを `Tenant` とは別の Aggregate にする | 認可と realm の解決が依存する `Tenant` を太らせない。詳細は[判断](decisions.md#外装と属性スキーマを-tenant-から独立した-aggregate-にする) |
| 上限の確認を、作成する側のトランザクションに委ねる | 作成と使用量の加算を同時に確定させる |

## 構成要素

コードは機能スライスを持たず、一つの層の構成である。
機能仕様に対応するコードは、要件の ID を `//spec:covers` で名指すテストからたどる。

| 層 | 責務 |
| --- | --- |
| `domain` | `Tenant`、`TenantBranding`、`TenantQuota` と値の検証、ドメインイベント |
| `ports` | 永続化のインターフェース。属性スキーマのポートは、実装を `IdManagement` が持つ |
| `usecases` | 操作ごとの手順。認可済みの入力を受け、検証し、保存し、イベントを発行する |
| `handlers_http` | テナント単位の管理 API、制御面のテナント管理 API、公開のブランド配信 |
| `db_postgres`、`db_memory` | `ports` の PostgreSQL の実装と、テストとローカルの構成で使うメモリの実装 |
| `testing_contract` | 二つの実装が同じ契約を満たすことを確かめる共通のテスト |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| テナントの解決 | すべての HTTP の要求 | `api` のミドルウェアが、ハンドラーより先に解決する | [テナントの解決の設計](../resolution/design.md) |
| テナント単位の管理 API | 解決済みのテナントの `/api/admin/v1/...` への要求 | `api` が、ハンドラーからユースケースを同期的に呼ぶ | 各機能仕様 |
| 制御面のテナント管理 API | 解決済みのテナントの `/api/admin/v1/tenants/...` への要求 | `api` が、`RequireControlPlaneUser` で `system_admin` と制御面テナントへの所属を確かめてからユースケースを呼ぶ | [テナントのライフサイクル](../lifecycle/README.md) |
| 上限の確認 | ほかのモジュールのリソースの作成 | 作成する側のユースケースが、作成のトランザクションの中で `CheckAndIncrement` を呼ぶ | [リソース上限](../quota/README.md) |
| デフォルトテナントの作成 | プロセスの起動 | 起動処理が、存在しなければ作成する | [テナントのライフサイクル](../lifecycle/README.md) |

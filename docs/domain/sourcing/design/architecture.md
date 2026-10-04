# Sourcing のアーキテクチャ

この文書は、Sourcing の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

この Context がほかの Context と結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
この Context から外への依存の向きは次のとおりである。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `IdManagement` | 取り込みが User と Group を読み書きし、User を止める | 相手の `user` と `group` の Repository と、`UserLifecycle` の実装を使う |
| `ApiTokens` | SCIM の要求を認証する | 相手の認証の部品を、ルーティングで SCIM の経路の前に置く |

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| 取り込み元ごとに機能を設け、共通の機構は実在する共通点が判明してから作る | 一つの取り込み元から推測した抽象を、ほかの取り込み元に押し付けない |
| 外部の表現を正規の Aggregate へ投影し、ワイヤ表現を保存しない | 正規のユーザー像を一つに保つ |
| 外部の削除を論理削除へ統合する | 外部の同期の誤りで個人識別情報を失わない。詳細は[判断](decisions.md#scim-の-user-の削除を論理削除へ統合する) |

## 構成要素

コードは機能スライス `backend/sourcing/scim` を一つ持ち、Context のルートにはファサードと組み立てだけを置く。

| 層 | 責務 |
| --- | --- |
| `scim/domain` | SCIM のモデル、フィルター、検索、書き込みと PATCH の解析、Discovery |
| `scim/ports` | 対応の記録の Repository、`UserLifecycle` |
| `scim/usecases` | User と Group の検索、作成、置換、部分更新、削除 |
| `scim/source_idmanagement` | IdManagement の User と Group の所有の判定 |
| `scim/handlers_http` | SCIM のエンドポイント |
| `scim/db_postgres`、`scim/db_memory` | `ports` の PostgreSQL の実装と、テストとローカルの構成で使うメモリの実装 |
| `scim/testing_contract` | 二つの実装が同じ契約を満たすことを確かめる共通のテスト |

すべての要件は[SCIM による取り込み](../scim/README.md)に属する。

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| SCIM の要求 | 外部の IdP の `/realms/{realm_id}/scim/v2/...` への要求 | `api` が、トークンを認証してからユースケースを同期的に呼ぶ | [SCIM による取り込みの設計](../scim/design.md) |

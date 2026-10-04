# Authorization のアーキテクチャ

この文書は、Authorization の文脈、解決戦略、構成要素、実行時の流れを扱う。
システム全体の構成は[論理アーキテクチャ](../../../design/architecture/logical.md)と[ランタイムアーキテクチャ](../../../design/architecture/runtime.md)に従う。

## 文脈と範囲

この Context がほかの Context と結ぶ契約は、仕様の[公開する契約](../README.md#公開する契約)が定める。
この Context から外への依存の向きは次のとおりである。

| 相手 | 向き | 実現方式 |
| --- | --- | --- |
| `IdManagement` | 判定が、代行チェーンの actor の状態を読む | `principals_idmanagement` のアダプターが、`PrincipalStatusResolver` を相手の Agent の Repository で実装する |
| `OAuth2` | 判定が、関係の事実を評価器へ渡す | 相手の AuthZEN の評価器の `resource:access` の規則を使う |

現在、データ資源を提供する呼び出し側のユースケースはなく、判定の入口は管理 API の診断の経路だけである。

## 解決戦略

| 方針 | 理由 |
| --- | --- |
| 関係の事実だけを求め、判断の合成は評価器に任せる | 外部の PDP へ差し替えても、合成の規則が重複しない。詳細は[判断](decisions.md#関係の事実を求め判断の合成を評価器に任せる) |
| 書き換え規則を和だけで組む | 評価が単調になる。詳細は[判断](decisions.md#書き換え規則を和だけで組み交差と差集合を入れない) |
| まず判定の正しさを優先し、列挙に索引を持たない | 詳細は[判断](decisions.md#リソースの列挙を上限付きの走査と-1-件ずつの判定で行う) |

## 構成要素

コードは機能スライスを持たず、一つの層の構成である。

| 機能仕様 | 主なユースケースとハンドラー |
| --- | --- |
| [認可モデル](../model/README.md) | `usecases/admin_model.go`、`domain/model.go` |
| [関係タプル](../relation-tuple/README.md) | `usecases/admin_tuples.go`、`domain/tuple.go` |
| [関係の判定](../check/README.md) | `usecases/check_access.go`、`domain/evaluator.go`、`domain/consistency.go` |

| 層 | 責務 |
| --- | --- |
| `domain` | 認可モデルと関係タプルの検証、関係グラフの評価、整合トークン、ドメインイベント |
| `ports` | モデルと関係タプルの Repository |
| `usecases` | モデルの登録、関係タプルの書き込み、判定、列挙 |
| `principals_idmanagement` | `PrincipalStatusResolver` の実装 |
| `handlers_http` | テナント単位の管理 API |
| `db_postgres`、`db_memory` | `ports` の PostgreSQL の実装と、テストとローカルのデモで使うメモリの実装 |
| `testing_contract` | 二つの実装が同じ契約を満たすことを確かめる共通のテスト |

## 実行時の流れ

| 流れ | 契機 | 実行する場所 | 詳細 |
| --- | --- | --- | --- |
| 管理 API | 解決済みのテナントの `/api/admin/v1/authorization/...` への要求 | `api` が、ハンドラーからユースケースを同期的に呼ぶ | 各機能仕様 |
| 判定と列挙 | 管理 API の診断の経路、またはデータ資源を提供する呼び出し側 | 呼び出したプロセスが、関係タプルを読みながら同期的に評価する | [関係の判定の設計](../check/design.md) |

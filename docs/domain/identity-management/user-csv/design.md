# ユーザー CSV の設計

この文書は、[ユーザー CSV](README.md)の規則を保証する仕組みのうち、User に固有のものを扱う。
User と Group が共有するプレビュー、適用、成果物、行のエラーの仕組みは、モジュールの横断的概念[CSV の往復変換](../design/csv-transfer.md)が扱う。

| 話題 | 記述した場所 |
| --- | --- |
| アーキテクチャ | [アーキテクチャ](#アーキテクチャ) |
| 設計判断 | 該当なし：モジュールの[重要な設計判断](../design/decisions.md)と規則の判断の欄に置く |
| アプリケーション | 該当なし：モジュールの設計に従う |
| データ | 該当なし：成果物と行の確定はモジュールの[データ](../design/data.md)と[CSV の往復変換](../design/csv-transfer.md)に従う |
| セキュリティ | 該当なし：仕様のセキュリティ上の考慮に置く |
| 信頼性 | 該当なし：モジュールの[信頼性](../design/reliability.md)に従う |
| 性能 | 該当なし：モジュールの[性能](../design/performance.md)に従う |
| オブザーバビリティ | 該当なし：モジュールの設計に従う |
| 検証 | 該当なし：モジュールの[性能](../design/performance.md)の確かめ方に従う |
| インフラストラクチャ | 該当なし：モジュールの設計に従う |
| リスク | 該当なし：モジュールの設計に従う |

## アーキテクチャ

| 構成要素 | 責務 | 依存先 |
| --- | --- | --- |
| 列の語彙（`UserCSVSchema`） | 組み込みの列と、実効的な属性スキーマから作る `attr:` と `custom:` の列 | `EffectiveUserAttributeSchemaReader` |
| 計画器（`PlanUserImport`） | 行の対象を決め、型付きの値と行の間の衝突を検証し、行操作を作る。プレビューと適用が同じ計画器を使う | `UserRepository`、`UserSourceOwnershipGuard` |
| 適用（`ApplyUserImport`） | 計画した行を一行ずつ確定する。作成する User に無作為なパスワードと必須操作 `update_password` を付ける | `UserImportRowCommitter` |
| エクスポーター（`UserCSVExporter`） | 選んだ列で User をページ単位に読み、成果物ストアへ書く | `UserRepository`、`CSVArtifactStore` |

`UserImportRowCommitter` は、一行の Aggregate、パスワードの履歴、使用量の変化、監査イベントを一つのトランザクションで確定する。

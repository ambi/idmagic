# リポジトリツール

`tools/` は、リポジトリで使う決定的なツールを一つの Bun workspace にまとめる。
利用者向けの入口は TypeScript ファイルではなく、リポジトリ root の `mise run <task>` である。

ツールは責任ごとに次の種類へ分ける。

| 種類 | 配置 | 責任 |
| --- | --- | --- |
| 検査 | `check/` | registry に登録した規則を共有 snapshot に対して実行し、所見を返す |
| 生成 | `generate-contract/`、`render-docs/` | 正準入力から派生成果物を作る |
| 照会 | `brief/`、`changed-packages/`、`task-timing/`、`spec-diff/`、`spec-review-candidates/`、`security-test-gap-report/` | 作業対象、差分、時間、未検査の拒否、仕様漏れの候補を人へ報告する |
| 作業記録の操作 | `work-item/`、`work-item-number/` | 作業項目の移動と参照更新、未使用番号の選択 |
| 検証実行と報告 | `verification/` | Go の実行イベント、操作ごとの静的解析、引用の存在を別々の事実として保存し、必要な証拠の欠落を判定する |
| 共有 | `workspace/` | リポジトリを発見して一度だけ読み、上のモジュールへ渡す |

検査の実行方式と採否の基準は[リポジトリ検査](check/README.md)が定める。
個別入口は `mise tasks` で確認する。

ツールは製品固有の値を正準入力と標準パスから導く。
TypeSpec の所有元は compiler の source location、OpenAPI の名前は `spec/` 以下、Go の import root は `go.mod` から得る。

検証結果は `.verification/` に生成し、手で編集または Git へ登録しない。
結果の区分、照会、代替証拠の記法は[テスト方針](../docs/development/testing.md#検証結果の区分)が定める。

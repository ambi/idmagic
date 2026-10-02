# グループ CSV

`Group` の CSV インポートとエクスポート、`lifecycle_action` による削除、一つの `Group` のメンバーシップの CSV インポートとエクスポートを扱う。
コードは `backend/idmanagement/group` の機能スライスにある。
`User` と共有する転送ポリシー、解析器、セル変換、成果物ストア、エクスポートの状態遷移は、Context のルートの[内部設計](../internals.md)と[状態遷移](../states.md)が扱う。

| 文書 | 内容 |
|---|---|
| [グループ CSV の設計判断](decisions.md) | 設計判断 |
| [グループ CSV の内部設計](internals.md) | 機構の説明 |
| [グループ CSV のシナリオ](scenarios.feature.md) | 受け入れシナリオ |

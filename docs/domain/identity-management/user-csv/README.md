# ユーザー CSV

`User` の CSV インポート（プレビューと適用）とエクスポートを扱う。
コードは `backend/idmanagement/user` の機能スライスにある。
`Group` と共有する転送ポリシー、解析器、セル変換、成果物ストア、エクスポートの状態遷移は、Context のルートの[内部設計](../internals.md)と[状態遷移](../states.md)が扱う。

| 文書 | 内容 |
|---|---|
| [ユーザー CSV のシナリオ](scenarios.feature.md) | 受け入れシナリオ |

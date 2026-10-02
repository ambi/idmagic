# グループ

`Group` の作成、更新、削除、連絡先とカスタム属性、手動のメンバーシップ、グループ由来のロールを含む実効ロールを扱う。
コードの機能スライスは `backend/idmanagement/group` である。
CEL の規則による所属は[動的グループ](../dynamic-group/README.md)が、CSV による一括の操作は[グループ CSV](../group-csv/README.md)が扱う。

| 文書 | 内容 |
|---|---|
| [グループの設計判断](decisions.md) | 設計判断 |
| [グループの内部設計](internals.md) | 機構の説明 |
| [グループのシナリオ](scenarios.feature.md) | 受け入れシナリオ |

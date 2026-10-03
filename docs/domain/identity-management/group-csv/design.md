# グループ CSV の設計

この文書は、[グループ CSV](README.md)の規則を保証する仕組みのうち、Group とメンバーシップに固有のものを扱う。
User と Group が共有するプレビュー、適用、成果物、行のエラーの仕組みは、Context の横断的概念[CSV の往復変換](../design/csv-transfer.md)が扱う。

| 話題 | 記述した場所 |
| --- | --- |
| アーキテクチャ | [アーキテクチャ](#アーキテクチャ) |
| 設計判断 | 該当なし：規則の判断の欄と、Context の[重要な設計判断](../design/decisions.md)に置く |
| アプリケーション | 該当なし：削除と解除を含む適用の明示の確認は、仕様のセキュリティ上の考慮に置く |
| データ | 該当なし：成果物と行の確定は Context の[データ](../design/data.md)と[CSV の往復変換](../design/csv-transfer.md)に従う |
| セキュリティ | 該当なし：仕様のセキュリティ上の考慮に置く |
| 信頼性 | 該当なし：Context の[信頼性](../design/reliability.md)に従う |
| 性能 | 該当なし：Context の[性能](../design/performance.md)に従う |
| オブザーバビリティ | 該当なし：Context の設計に従う |
| 検証 | 該当なし：Context の[性能](../design/performance.md)の確かめ方に従う |
| インフラストラクチャ | 該当なし：Context の設計に従う |
| リスク | 該当なし：Context の設計に従う |

## アーキテクチャ

| 構成要素 | 責務 | 依存先 |
| --- | --- | --- |
| Group の計画器（`PlanGroupImport`） | 対象を `id`、次に `name` で決め、`membership_type` の不変、動的規則の最終の状態、削除の意図を検証し、行操作を作る | `GroupRepository`、`GroupSourceOwnershipGuard`、`EffectiveGroupAttributeSchemaReader` |
| Group の適用（`ApplyGroupImport`） | 計画した行を一行ずつ確定する。削除の行は Group の削除、メンバーシップの解除、使用量の解放、監査の記録を一つのトランザクションで確定する | `GroupImportRowCommitter` |
| メンバーシップの計画器（`PlanGroupMembershipImport`） | 対象の User を `user_id`、次に `preferred_username` で決め、`membership_state` と現在の所属から行操作を作る | `GroupRepository`、`UserRepository`、`GroupSourceOwnershipGuard`、`UserSourceOwnershipGuard` |
| メンバーシップの適用（`ApplyGroupMembershipImport`） | 一行のメンバーシップの追加または解除と、監査の記録を一つのトランザクションで確定する | `GroupMembershipImportRowCommitter` |
| エクスポーター | Group の CSV では `lifecycle_action` を空に、メンバーシップの CSV では `membership_state` を `present` にして書き出す | `GroupRepository`、`CSVArtifactStore` |

### 動的規則の検証

動的規則の式と有効化は二つの列に分かれるが、検証は片方だけを見ない。
行が与えた列と、維持するもう一方の列を合わせた最終の状態を作り、それに対して規則をコンパイルする。
式は動的グループと同じ小さな CEL の環境でコンパイルし、参照する属性も同時に決める。

規則の削除は CSV から行わない。
Group の Aggregate に規則を削除する操作がなく、空のセルを「規則なし」と読むと、列を落としたファイルと空のセルのファイルの意味が入れ替わるからである。

### 削除の多層の守り

削除は不可逆で、メンバーシップの解除により、所属していた全員の実効ロールを一度に変える。
一つでも欠けると分割ファイルや編集の誤りが権限の一括の剥奪になるので、次の守りをすべて保つ。

| 守り | 内容 |
| --- | --- |
| プレビューと適用の束縛 | 適用は、管理者が確認したペイロードだけを実行する |
| 現在の状態からの再計画 | プレビューの後の変更を、適用の時点で判定し直す |
| 閉じた語彙 | `lifecycle_action` は `delete` だけを受け付け、未知の値を丸めない |
| 明示の確認 | 管理 UI は削除を含む適用に確認を求める |
| エクスポートでの空の書き出し | 編集していないエクスポートの再適用が削除にならない |
| 解決できない削除の拒否 | 対象を解決できない削除の行を作成として扱わない。綴りの誤りや古いファイルを `unchanged` として通さない |

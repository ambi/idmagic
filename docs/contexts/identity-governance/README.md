# IdGovernance

アイデンティティガバナンス (IGA) のポリシーとオーケストレーションを担う。JML を自動化する LifecycleWorkflow の定義、トリガー評価、WorkflowRun の実行を扱う。

記録の正は持たない。User と Group は `IdManagement`、Application の割り当ては `Application` が正の記録を持つ。`IdGovernance` は User のライフサイクルイベントを購読し、冪等なコマンドインターフェースを介してこれら記録系 Context の状態を変更する。

| 文書 | 内容 |
|---|---|
| [IdGovernance の用語集](glossary.md) | この Context での語義 |
| [IdGovernance の状態遷移](states.md) | 状態と遷移 |
| [IdGovernance の設計判断](decisions.md) | 設計判断 |
| [IdGovernance の内部設計](internals.md) | 機構の説明 |
| [IdGovernance Scenarios](scenarios.feature.md) | 受け入れシナリオ |

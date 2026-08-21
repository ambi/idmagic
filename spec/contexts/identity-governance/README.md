# IdGovernance

アイデンティティガバナンス (IGA) のポリシーとオーケストレーションを担う。JML を自動化する LifecycleWorkflow の定義、トリガー評価、WorkflowRun の実行を扱う。

記録の正は持たない。User と Group は `IdManagement`、Application の割り当ては `Application` が正の記録を持つ。`IdGovernance` は User のライフサイクルイベントを購読し、冪等なコマンドインターフェースを介してこれら記録系 Context の状態を変更する。

| File | Content |
|---|---|
| [glossary.md](glossary.md) | この Context での語義 |
| [states.md](states.md) | 状態と遷移 |
| [decisions.md](decisions.md) | 設計判断 |
| [internals.md](internals.md) | 機構の説明 |
| [scenarios.md](scenarios.md) | 受け入れシナリオ |

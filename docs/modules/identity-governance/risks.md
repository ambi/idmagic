# IdGovernance のリスク

この文書は、IdGovernance の既知の欠陥、技術的負債、受容したリスクを扱う。
脅威は[脅威モデル](../../design/security/threat-model.md)に置く。

| リスク | 現状 | 影響 |
| --- | --- | --- |
| 実行の記録の保持期間を適用する処理がない | WorkflowRun、WorkflowStep、通知の配送の記録を、Job と同じ 30 日で消す方針がある一方、それらを消す処理はない | 実行の記録が消えずに増え続ける |

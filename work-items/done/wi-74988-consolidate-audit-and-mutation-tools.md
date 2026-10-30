---
status: completed
---

# 脆弱性監査を OSV-Scanner、変異検査を gomutants に集約する

`mise run spec-diff` は main に対する規範仕様差分なしを報告した。
依存監査と Go の到達性解析を OSV-Scanner、変異検査を gomutants に集約し、govulncheck と Mutago の宣言、専用タスク、CI 呼び出し、Mutago 専用の検算コードと fixture を削除した。
OSV は `--all-vulns` を付け、未到達の検出も理由と期限付きで抑止する方針を維持する。
呼び出し側のテストは gomutants の統合モードを使い、準備時間の増加を受容する。
過去の完了記録と、gomutants の性能改善を扱う既存の作業項目は変更しない。

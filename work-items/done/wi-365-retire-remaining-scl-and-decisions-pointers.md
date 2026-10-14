---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# 削除済みの spec/scl.yaml と decisions/ を指したままの記述と設定を実態へ揃える

削除済みの `spec/scl.yaml` と `decisions/` を指したままの箇所を実態へ揃えた。root
`SPECIFICATION.md` の Structure から `decisions/` を除き、work-items が判断履歴も持つことを明記。
traceability の Design 記述には、導出ビューが citation から生成される現状を追記した。
`SCL_PATH`（読み手のいない環境変数、削除済みファイルを指す）を compose / k8s / Cloud Run の 3 設定
から除去した。Go コメント 5 箇所の `spec/scl.yaml` 参照を現在の所有者（root Design の context 関係、
tenancy の PasswordPolicyDefaults、`just check-boundaries`）へ付け替え、既に存在しない coherence
test への言及を削除した。`ListAdminRolePolicies` の `@doc` から SCL の語を外した。
加えて wi-359 で入れた cross-context 配置規約の誤りを修正した。root `SPECIFICATION.md` は Overview
と Design だけで Scenarios を持たず、context 横断シナリオは `docs/domain/system/SPECIFICATION.md`
が所有している。`SPECIFICATION_FORMAT.md` は製品非依存に「cross-context view の所有を宣言している
文書に置く」とし、`spec-change` skill でこのリポジトリの所有者を示した。この誤りは、wi-360 で入れた
`initial_context` の参照解決検査が本 work item 自身の `REQ-SYSTEM-001` 参照を落として発覚した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。

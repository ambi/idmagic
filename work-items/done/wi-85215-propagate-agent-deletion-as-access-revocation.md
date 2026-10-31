---
status: completed
---

# Agent の削除でも失効エポックを進め、AgentAccessRevoked を外部へ伝播する

REQ-SHAREDSIGNALS-001 の契機に Agent の削除を加え、`RevocationReason` に `AgentDeleted` を追加した。
管理 API の削除は、失効エポックを理由 `AgentDeleted` で記録し、`RevocationEpochAdvanced` と `AgentAccessRevoked` を発行して、送信側のストリームへ `initiating_entity` が `admin` の SET の配送を作る。
`agent_revocation_epochs` の `agents` への外部キーを外し、削除した Agent の行を無期限に保持する。

受理した SET もエポックを書くが、`resolvePrincipal` が同じテナントの Agent へ解決した後に限るので、外部の任意の値は入らない。
旧スキーマからの適用計画は `CHECK` の作り直しと外部キーの `DROP CONSTRAINT` で、`--enable-drop` なしでも同じ DDL が出る。
新しいアプリケーションより先にスキーマを適用する。逆の順序では、削除は確定したまま `CHECK` の違反で失敗する。
削除の確定後に使用量の減算か反応が失敗すると SET は送られず、再試行も 404 になる。ローカルのトークンは Agent の不在で無効のままなので、この項目では変えていない。

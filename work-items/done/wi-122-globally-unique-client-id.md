---
status: completed
archived_from: c846d5e549cbbe72c662efb5054e4abdb53d670b
---

# client_id のグローバルユニーク強制とスキーマ・コードの単純化

`client_id` をシステム自動生成のグローバルユニーク ID に強制し、複合キーの複雑性をスキーマ・コードから排除。
- SCL (`spec/contexts/oauth2.yaml` / `spec/contexts/identity-management.yaml`) を修正し、`OAuth2Client` 等から `tenant_id` キー情報を排除。
- Postgres スキーマ (`postgres.sql`) で主キー・複合外部キー制約を削除・簡略化（`clients`, `consents`, `refresh_tokens`, `agents`, `agent_credential_bindings`, `application_orderings`, `scim_user_refs`, `scim_group_refs`）。
- Go の struct (`OAuth2Client`, `Consent`, `AgentCredentialBinding`) およびバリデーションを簡素化。
- Postgres およびメモリの各リポジトリ実装、API ハンドラーの修正・調整。
- UI の `types.ts` および同意詳細画面の `tenant_id` 非依存化の調整。

全文は `git show c846d5e549cbbe72c662efb5054e4abdb53d670b:<このファイルのパス>` で読む。

---
id: idp-wi-117-postgres-referential-integrity-constraints
title: "Postgres schema の参照整合性制約を補完する"
created_at: 2026-07-04
authors: [tn]
status: completed
risk: medium
---

# Motivation
`deploy/schema/postgres.sql` には、tenant-owned なレコードや live aggregate 参照でありながら DB レベルの外部キーが無い箇所がある。
通常の repository 経路では検証されていても、移行、リプレイ、バグ、直接 SQL 実行で孤児行やクロステナント参照が入り得る。
Identity / Application / OAuth2 / SigningKeys の保存契約では tenant isolation と fail-closed な参照整合性が重要なので、DB 制約で守るべき参照を明確化して補完する。

# Scope
- `spec/contexts/identity-management.yaml`
  - Agent owner、Agent credential binding、User / Group 参照の整合性保証を確認し、必要なら models / invariants / scenarios を更新する。
- `spec/contexts/application.yaml`
  - Application、ApplicationCategory、ApplicationOrdering、ApplicationAssignment の tenant / subject / category / application 参照の保証を確認し、必要なら models / invariants / scenarios を更新する。
- `spec/contexts/oauth2.yaml`
  - RefreshToken、Consent、OAuth2 client 参照の delete / retain semantics を確認し、必要なら models / invariants / scenarios を更新する。
- `spec/contexts/signing-keys.yaml`
  - SigningKey が必ず既存 tenant に属する保証を確認し、必要なら models / invariants / scenarios を更新する。
- `spec/contexts/tenancy.yaml`
  - tenant 削除または無効化時に live aggregate と履歴データがどう扱われるかを確認し、必要なら models / invariants を更新する。
- Go / SQL
  - `deploy/schema/postgres.sql` に必要な外部キー、複合一意制約、チェック、または明示的な代替検証を追加する。
  - 影響する repository / migration / seed data を更新する。
  - schema bootstrap と不正参照拒否のテストを追加する。

# Initial Context
- `signing_keys.tenant_id` は SCL 上 tenant-scoped だが、schema では `tenants(id)` への FK が無い。
- `applications` と `application_categories` は tenant-owned だが、schema では tenant FK が無い。
- `application_orderings` は `tenant_id` / `user_sub` / `application_ids` を持つが、存在性は DB で保証されていない。
- `agent_credential_bindings` は `agent_id` だけ FK があり、`(tenant_id, client_id)` の OAuth2 client 参照と agent/client の tenant 一致は保証されていない。
- `agents.owner_sub` と `refresh_tokens.sub` は live principal 参照に見えるため、削除時の扱いを明示して DB 制約を検討する。
- `audit_events`、`authentication_event_buckets`、`outbox` は履歴・イベント用途なので、FK を持たない設計が妥当な可能性がある。

# Out of Scope
- 監査イベント、認証イベント集計、outbox の履歴保持方針を変えること。
- user / tenant / client 削除時に過去の audit record を削除すること。
- 参照整合性のために必要な場合を除く、Application category の配列保存方式の全面再設計。

# Affected Guarantees
- tenant-owned row の tenant isolation。
- Application assignment と portal ordering の fail-closed な参照整合性。
- OAuth2 refresh token / client binding の所有関係整合性。
- SigningKey の tenant isolation。

# Verification
- `just yaml-check-scl`
- `just scl-render`
- `just yaml-check-work-items`
- `just check-ids`
- `just verify-go`

# Risk Notes
外部キー追加は既存データに孤児行があると migration が失敗し、ON DELETE の選択を誤ると live data を消しすぎるか、削除不能にする。
実装時は既存データ形状を確認し、関係ごとに `RESTRICT` / `CASCADE` / `SET NULL` / 履歴保持の非 FK を明示的に選ぶ。
配列参照や polymorphic subject 参照は単純な FK で表現できないため、join table、trigger、または仕様化された application-level validation のいずれで守るかを決める。

# Completion
- **Completed At**: 2026-07-04
- **Summary**:
  SCL に tenant-local な参照整合性保証を明記し、Postgres 宣言的 schema に
  tenant / user / client / agent / application category / application ordering の
  外部キーと ApplicationAssignment subject 検証 trigger を追加した。既存データ向けには
  apply 前に孤児行を検出する preflight SQL を追加した。
- **Verification Results**:
  - `just yaml-check-scl`
    - result: ok (SCL YAML 12 files OK)
  - `just scl-render`
    - result: ok (spec/idmagic.html、spec/idmagic.full.html、spec/idmagic.models.schema.json、
      spec/idmagic.openapi.json を再生成)
  - `just yaml-check`
    - result: ok (SCL、work item YAML、id check が成功)
  - `GOCACHE=/tmp/idmagic-go-cache go test ./internal/shared/adapters/persistence/postgres`
    - result: ok
  - `just verify-go`
    - result: ok (sandbox 外実行。golangci-lint 0 issues、`go test -race ./...` 成功)
  - `just verify-ui`
    - result: ok (format / lint / typecheck / build 成功)
- **Affected Guarantees State**:
  tenant-owned live aggregate は DB 制約または明示された use case validation で
  tenant isolation と fail-closed な参照整合性を維持する。audit_events /
  authentication_event_buckets / outbox は履歴・イベント用途の非 FK 方針を維持した。

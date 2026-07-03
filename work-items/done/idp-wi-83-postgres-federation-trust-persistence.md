---
id: idp-wi-83-postgres-federation-trust-persistence
title: "SAML と WS-Federation の trust 登録を PostgreSQL に永続化する"
created_at: 2026-06-28
authors: ["Codex"]
status: completed
risk: medium
---
# Motivation
PERSISTENCE=postgres は durable state を PostgreSQL に置く方針だが、SAML SP と
WS-Federation RP の trust 登録だけが memory fallback に残っていた。Application
管理画面から federation binding を作成できても、postgres 構成では再起動で protocol
trust が失われるため、永続化アダプタの再生成可能性と運用上の期待に反する。

# Scope
- **scl_sections**:
- **contexts**: Saml, WsFederation, Infrastructure
- **code**: idmagic/internal/infrastructure/persistence/postgres/saml_service_providers.go, idmagic/internal/infrastructure/persistence/postgres/wsfed_relying_parties.go, idmagic/internal/bootstrap/postgres.go, idmagic/deploy/migrations/0018_federation_trusts.sql

# Out of Scope
- SAML / WS-Federation の wire 契約や request validation の変更
- Entra 実テナント接続検証
- 既存 memory adapter の意味変更

# Verification
- [object Object]
- [object Object]

# Risk Notes
既存 postgres 環境では AUTO_MIGRATE により新規テーブルが追加される。SAML / WS-Fed
trust を既に memory fallback にだけ持っていた実行中プロセスのデータは永続化されていないため、
再起動後は管理 API または Application 編集面から再登録が必要になる。

# Completion
- **Completed At**: 2026-06-28
- **Summary**:
  SAML SP / WS-Fed RP 用の PostgreSQL migration と repository adapter を追加し、
  PERSISTENCE=postgres の bootstrap で memory fallback ではなく PostgreSQL adapter を
  配線した。SCL の protocol 契約は変更せず、durable adapter の欠落だけを埋めた。
- **Verification Results**:
  - [object Object]
  - [object Object]

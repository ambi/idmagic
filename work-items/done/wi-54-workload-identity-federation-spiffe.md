---
status: completed
archived_from: ff62527e1f66f56cd65d792f548c84965898617e
---

# エージェントランタイム向け workload identity federation (SPIFFE 互換)

ADR-053 を accepted にし、WorkloadTrustBundle / AgentWorkloadBinding の 2 エンティティ構成で
実装した (起草時の WorkloadIdentityProvider/TrustDomain/SubjectMapping はこの 2 エンティティに
統合)。新規 bounded context `backend/workloadidentity` を、Domain → Verification Adapters
(`backend/shared/security/tokens_jose` の RS256 対応追加 + `verification_jose` adapter) →
Usecase (`VerifyWorkloadAttestation`、`backend/oauth2/token/usecases/exchange_token.go` への
`subject_token_type=urn:ietf:params:oauth:token-type:jwt` 分岐接続) → Admin API
(`handlers_http`、Postgres 永続化含む) → composition root (bootstrap memory/postgres、
`oauth2.Module.WorkloadVerifier`) の順で実装した。`just check` / `just test-go` /
`just lint-go` / `just build-go` すべて green。
**対応していないこと (Out of Scope のまま)**:
- X.509-SVID (mTLS) ベースの bootstrap — JWT-SVID のみ対応。
- SPIRE server / agent の同梱・運用。
- Transaction Tokens による内部サービス間伝播。
- Admin API (`backend/workloadidentity/handlers_http`) の httptest 統合テスト — usecase 層
(`backend/workloadidentity/usecases`) と postgres 層 (embedded-postgres) で CRUD ロジック・
制約は網羅したが、CSRF 検証には oauth2 の `/api/auth/account` 相当のセッション/CSRF cookie
発行経路が要り、この WI 単体での複製コストに見合わないため見送った。ハンドラの型検査と
ルート/OpenAPI contract test (`TestAssembledRoutesMatchGeneratedOpenAPI`) で配線のみ確認済み。

全文は `git show ff62527e1f66f56cd65d792f548c84965898617e:<このファイルのパス>` で読む。

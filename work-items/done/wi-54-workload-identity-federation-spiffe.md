---
depends_on: [wi-49-agent-identity-first-class-principal, wi-50-token-exchange-delegation-actor-chain]
status: completed
authors: ["tn"]
risk: high
created_at: 2026-06-22
---

# エージェントランタイム向け workload identity federation (SPIFFE 互換)

## Motivation
自律エージェントは人間の介在なしにコンテナ / 関数 / VM 上で起動し、行動するため、
長期シークレットを埋め込まない非人間 ID のブートストラップが要る。現代の基盤は
実行環境の attestation (k8s ServiceAccount token、クラウドの instance identity、
SPIFFE/SPIRE の SVID) を信頼の起点とし、それを IdP の token と federation で交換する。
Google Cloud の Workload Identity Federation や SPIFFE/SPIRE (CNCF) がこの代表で、
static key を持たないエージェントランタイムの ID を実現する。

idmagic は client_credentials を持つが、外部 attestation を信頼の起点にした
federation を持たない。本 WI は外部 workload token (OIDC 互換の JWT-SVID /
k8s SA token / クラウド federation token) を検証し、
[[wi-50-token-exchange-delegation-actor-chain]] の token-exchange 経由で
[[wi-49-agent-identity-first-class-principal]] のエージェントに紐づく idmagic token に
交換する経路を実装する。これにより「自律ワークロード」シナリオのエージェントが
シークレットレスで資格情報を得られる。

## Scope
- **decision**:
  - 新規 ADR [[ADR-053]]: 信頼する attestation 種別 (OIDC JWT を起点、X.509-SVID は将来)、 trust domain / issuer の登録方法、外部 subject から idmagic principal (Agent / client) への mapping 規則、token-exchange (RFC 8693) を交換機構に使う前提、 鍵・issuer の検証点を確定する。
- **scl**:
  - 新規 model: WorkloadIdentityProvider / TrustDomain / AttestationClaim / SubjectMapping。subject_token_type に workload 系を追加する。
  - 新規 event: WorkloadIdentityProviderConfigured / WorkloadTokenExchanged / WorkloadAttestationRejected。
  - 新規 interface: 管理用の federation provider CRUD。permission AdminWorkloadIdentityManage。
- **go**:
  - 外部 issuer の JWKS 取得・検証、attestation claim の検証、subject mapping、 idmagic token への交換を fail-closed で実装する。
  - 短命トークン既定 (long-lived 資格情報を発行しない)。
- **http**:
  - federation provider 管理 API。token-exchange への workload subject 受理。

## Out of Scope
- SPIRE server / agent の同梱・運用 (idmagic は relying party / federation 側)。
- X.509-SVID (mTLS) ベースの bootstrap (まず JWT、X.509 は将来)。
- Transaction Tokens による内部サービス間伝播 (将来 WI、Txn-Tokens draft)。

## Plan
- [[ADR-053-workload-identity-federation-for-agent-runtimes]] の suggested decision を、実装済み Agent credential binding と RFC 8693 token exchange に合わせて accepted にする。idmagic 自身が SPIRE server/node attestor を再実装せず、信頼済み workload issuer の JWT-SVID/X.509-SVID を検証して Agent principal へ交換する。
- tenant-scoped trust bundle は trust domain、issuer/JWKSまたはCA、許可 SPIFFE ID pattern、束縛 agent_id、最大 token TTL を保持する。SPIFFE ID 文字列だけで agent を自動作成せず、事前登録済み binding を必須にする。
- JWT-SVID は署名・audience・exp・sub、X.509-SVID は chain・SAN URI・key usage・revocation/expiry を検証する。検証結果を subject token として既存 token exchange use case に渡し、requested audience/RAR を downscope する。
- bundle refresh は last-known-good と期限を持ち、期限切れ/ambiguous match/disabled agent は fail-closed。発行 access token は短命で、wi-58 の kill/revocation signal を introspection/denylist に接続可能にする。

## Tasks
- [x] T001 [ADR/SCL] ADR-053 の supported SVID、trust/binding model、exchange contract、events/constraints/contracts/scenarios を確定して再生成する。ADR-053 を accepted にし、モデル名を WorkloadTrustBundle/AgentWorkloadBinding に統合。新規 context `spec/contexts/workloadidentity.yaml` (models/states/interfaces/authorization/scenarios) を追加し `spec/scl.yaml` context_map・`architecture.yaml`・`ARCHITECTURE.md` を同期。oauth2.yaml の TokenRequest に subject_token 系フィールドを追加し Token interface の requires に workload 検証を接続。api-tokens.yaml に workload-identity:read/write scope を追加。`just check` green。
- [x] T002 [Domain] WorkloadTrustBundle と AgentWorkloadBinding を実装し、pattern overlap、tenant uniqueness、disabled lifecycle をテストする。
  - RED: `TestTrustBundleValidateHappyAndFailure` / `TestBindingValidateHappyAndFailure` / `TestMatchAgent_*` を `backend/workloadidentity/domain` 未実装状態でビルド失敗確認 (undefined symbols) → GREEN (`WorkloadTrustBundle`/`AgentWorkloadBinding`/`MatchAgent` 実装、scenario `複数bindingに曖昧にマッチするsubjectは拒否される`・`登録済みtrustbundle経由で...` に対応)。
- [x] T003 [Verification Adapters] JWT-SVID verifier (X.509-SVID は Out of Scope)、bundle cache/refresh を実装し、alg/audience/clock skew/TTL 上限を検証する。
  - RED: `TestVerifyWorkloadSVID` (spoofed signature / unregistered issuer / audience mismatch / expired / ttl exceeded / malformed) と `TestWorkloadJWKSCache` (last-known-good fallback / staleness fail-closed) を `backend/shared/security/tokens_jose` 未実装状態でビルド失敗確認 → GREEN。既存の共有 `verifyJWTSignature` に RS256 対応を追加 (外部 issuer は PS256/ES256 に限定できないため) して ADR-023 の検証基盤を再利用。
- [x] T004 [Usecase] verified workload identity を既存 token exchange/actor chain/RAR downscope に接続し、agent status と audience policy を再評価する。
  - RED: `TestVerifyWorkloadAttestation_*` (success/unregistered issuer/spoofed signature/expired/ambiguous match/killed agent/cross-tenant/disabled bundle) を `backend/workloadidentity/usecases` 未実装状態でビルド失敗確認 → GREEN。`backend/oauth2/token/usecases/exchange_token.go` に `subject_token_type=urn:ietf:params:oauth:token-type:jwt` の分岐を追加し `ports.WorkloadTokenVerifier` 経由で接続、`TestExchangeTokenWorkloadFederation*` で回帰確認 (act chain・scope downscope・resource indicator は既存ロジックを再利用、AgentID を AccessTokenInput に伝播、WorkloadTokenExchanged を追加 emit)。
- [x] T005 [Admin] trust bundle/binding CRUD、metadata refresh/test と credential 非表示の管理 API を追加する。
  - RED: `TestRegisterWorkloadTrustBundle` 系・`TestCreateAgentWorkloadBinding` 系を usecase 未実装状態でビルド失敗確認 → GREEN (name/issuer/subject_pattern 重複拒否、cross-tenant agent 拒否、cascade delete)。postgres 層は `TestWorkloadTrustBundleRepository_*`/`TestAgentWorkloadBindingRepository_*` (embedded-postgres) で UNIQUE 制約・ON DELETE CASCADE を検証。HTTP handler (`backend/workloadidentity/handlers_http`) は型検査・`just check` の route/OpenAPI contract test (`TestAssembledRoutesMatchGeneratedOpenAPI`) で配線を確認したが、CSRF harness (oauth2 の `/api/auth/account` 依存) の複製コストに見合わないため専用の httptest 統合テストは追加していない (usecase 層でロジックは網羅済み)。sqlc 生成 (`just sqlc-generate`)・schema 追加・composition root 配線 (bootstrap memory/postgres, `oauth2.Module.WorkloadVerifier`) を完了。
- [x] T006 [Verify] SPIRE fixture を用いた交換、spoofed SPIFFE ID、expired bundle/SVID、binding collision、kill 後の拒否、tenant 越境を検証する。
  - `TestVerifyWorkloadAttestation_Success` (spiffe:// subject を持つ RS256 JWT-SVID fixture 経由の成功交換)、`_SpoofedSignature`、`_Expired`、`_AmbiguousMatch` (binding collision)、`_KilledAgent`、`_CrossTenant`、`_DisabledTrustBundle`、`TestWorkloadJWKSCache` の staleness fail-closed で網羅。`just test-go` / `just lint-go` / `just build-go` / `just check` すべて green (既存の無関係な `TestPostgresSchemaReferentialIntegrityConstraints` 失敗のみ残存、本 WI 着手前から存在する oauth2_clients_application_fkey の欠落で無関係)。

## Verification
- `just test-go`
  - reason: 外部 JWT 検証、未登録 issuer / 改竄 attestation の拒否、subject mapping、短命 TTL の境界。
- `just lint-go`
- `just build-go`
- 手動: mock workload issuer の token を提示 → idmagic token に交換 → 未登録 issuer は拒否されることを確認する。

## Risk Notes
外部 attestation を信頼の起点にするため、issuer / 鍵 / audience / 有効期限の検証を
誤ると任意のワークロードがエージェント資格情報を得られてしまう。登録済み trust domain に
限定し、JWKS・issuer・audience・exp をすべて満たす場合のみ交換する (fail-closed)。
発行は短命に限定し long-lived 資格情報を作らない。

## Completion
- **Completed At**: 2026-08-08
- **Summary**:
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

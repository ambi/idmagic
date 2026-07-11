---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-06-22
---

# エージェントの外部 API 代行アクセス向け Token Vault / federated connections

## Motivation
エージェントはユーザーに代わって多数の外部 SaaS API (Google・GitHub・Slack 等) を
呼ぶ。これらの third-party token をアプリやエージェントが直接保持すると、漏洩・
失効困難・最小権限の崩れを招く。Auth0 の Token Vault (federated connections) は、
ユーザーが各 upstream へ与えた同意とトークンを IdP 側で安全に保管・更新 (refresh)・
失効し、エージェントには必要時に最小権限のアクセスを仲介する。これにより
「エージェントが raw シークレットを持たずに外部 API を代行呼び出しする」が成立する。

idmagic は inbound federation / identity broker ([[wi-30-inbound-federation-and-identity-broker]])
を持つが、それはログイン用 federation で、外部 API 呼び出し用の upstream token 保管・
仲介機構ではない。本 WI は upstream connection ごとに user の token を tenant-scoped に
暗号化保管し、refresh を管理し、エージェント ([[wi-49-agent-identity-first-class-principal]])
からの取得要求を委譲チェーン ([[wi-50-token-exchange-delegation-actor-chain]]) と
scope で絞って仲介する Token Vault を導入する。

## Scope
- **decision**:
  - 新規 ADR [[ADR-054]]: upstream connection の定義 (provider・OAuth エンドポイント・scope)、 token の暗号化保管方式 (既存 KMS / KeyStore の流用)、refresh とローテーションの責務、 エージェントへの仲介方法 (直接返却 vs プロキシ)、最小権限と失効の伝播を確定する。
- **scl**:
  - 新規 model: FederatedConnection / UpstreamToken / ConnectionGrant / VaultTokenRequest / VaultTokenResponse。
  - 新規 event: ConnectionConfigured / UpstreamTokenStored / UpstreamTokenRefreshed / UpstreamTokenRevoked / VaultTokenIssued。
  - 新規 interface: 管理 connection CRUD、user の connection 連携 / 解除、 エージェント向け GetConnectionToken。permission AdminFederatedConnectionsManage。
- **go**:
  - upstream OAuth (authorization code) で token を取得・暗号化保管、refresh 管理、 失効、エージェント要求への仲介を実装する。仲介は委譲チェーンと connection scope で fail-closed に絞る。
- **http**:
  - connection 連携の開始 / コールバック、エージェント向け token 取得エンドポイント。
- **ui**:
  - end-user: 連携済み外部サービスの一覧・連携 / 解除 (account portal)。
  - admin: connection (provider) の定義・管理。

## Out of Scope
- 各 provider 固有 API のラッパー / SDK 同梱。
- エージェントの外部 API 呼び出しそのもののプロキシ実装 (token 仲介まで)。
- 暗号鍵管理の新設 (既存 KMS / KeyStore を流用)。

## Verification
- `just test-go`
  - reason: token の暗号化保管・refresh・失効、仲介時の scope / 委譲絞り込み、解除後の取得拒否の境界。
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just build-ui`
- 手動: ユーザーが外部 provider を連携 → エージェントが vault から token 取得 → 連携解除後は取得が拒否されることを確認する。

## Risk Notes
Token Vault は外部資格情報の集中保管庫であり、漏洩時の影響が大きい。保管は既存 KMS /
KeyStore で暗号化し、新たな鍵管理を作らない。仲介は connection scope と委譲チェーンで
最小権限に絞り、解除・失効を後続アクセスへ確実に伝播する (fail-closed)。

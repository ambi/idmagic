---
depends_on: [wi-50-token-exchange-delegation-actor-chain, wi-56-mcp-authorization-server]
status: pending
authors: ["tn"]
risk: high
created_at: 2026-06-22
---

# Cross-App Access (Identity Assertion Authorization Grant) でエージェントのアプリ間アクセスを仲介する

## Motivation
企業内のエージェントが、あるアプリ (app A / MCP クライアント) から別アプリや
MCP サーバー (app B) のデータへ、ユーザーごとの個別 OAuth 同意画面を介さずに
アクセスしたい。これを IdP が仲介し、企業が一元的に可視化・統制する標準が
Identity Assertion Authorization Grant (IETF draft-ietf-oauth-identity-assertion-
authz-grant、Okta の Cross-App Access / XAA) である。IdP が信頼する identity
assertion (id_token 等) を Token Exchange で app B 向けアクセストークンに交換する。
この仕組みは 2026-06 に MCP の "Enterprise-Managed Authorization" として stable 化し、
エージェント連携の企業向け認可の中心になりつつある。

idmagic は Token Exchange ([[wi-50-token-exchange-delegation-actor-chain]]) と
MCP 認可サーバー ([[wi-56-mcp-authorization-server]]) を備える前提で、本 WI は
identity assertion を起点にした app-to-app / agent-to-MCP のブローカ付与を実装する。
これにより per-app の再同意を排し、エージェントのアプリ間アクセスを IdP が
集中管理 (付与・可視化・失効) できる。

## Scope
- **decision**:
  - 新規 ADR [[ADR-056]]: 対応する draft 改訂 (draft-ietf-oauth-identity-assertion-authz-grant)、 identity assertion の受理条件 (信頼する issuer / audience)、app A → app B の許可関係 (どの client がどの resource を要求できるか) の登録モデル、MCP Enterprise-Managed Authorization との対応、企業管理者による付与・取消の責務を確定する。
- **scl**:
  - 新規 model: IdentityAssertionGrantRequest / AppAccessGrant / CrossAppAccessPolicy。 token-exchange の subject_token_type に identity assertion 系を追加する。
  - 新規 event: CrossAppAccessGranted / CrossAppAccessRejected / AppAccessPolicyChanged。
  - 新規 interface: 管理用の app-to-app 許可ポリシー CRUD。permission AdminCrossAppAccessManage。
- **go**:
  - identity assertion の検証 → 許可ポリシー照合 → app B 向け audience 限定トークンへの 交換を fail-closed で実装する。Token Exchange / Resource Indicators 基盤を再利用する。
- **http**:
  - token-exchange での identity assertion 受理、app-to-app 許可ポリシー管理 API。
- **ui**:
  - admin: app 間アクセス許可の付与・一覧・取消 (企業管理者ビュー)。

## Out of Scope
- 外部 IdP (Okta 等) との相互運用認証取得。
- MCP 認可サーバー基盤そのもの ([[wi-56-mcp-authorization-server]] が前提)。
- エンドユーザー個別同意フロー (本 WI は企業管理の app 間付与が対象)。

## Plan
- depends_on の wi-50 と wi-56 が提供する actor-chain token exchange と MCP resource-bound audience を前提に、[[ADR-056-cross-app-access-identity-assertion-grant]] を accepted にする。新 grant を別 token pipeline にせず、token exchange の subject token profile として追加する。
- source app が署名する Identity Assertion は iss/sub/aud/exp/iat/jti、user authorization、source client/agent、actor chain を持ち、registered source key と destination resource policy で検証する。assertion audience は idmagic exchange endpoint、requested resource は destination app に分ける。
- jti replay window、短い TTL、source/destination tenant一致、source app allowlist、user consent/enterprise policy、RAR downscope をすべて満たす場合だけ destination audience token を発行する。
- managed authorization の設定・失効は Application context が所有し、token exchange result は既存 `act` chain に source app/agent を追加する。assertion 本文・user data は保存しない。

## Tasks
- [ ] T001 [Dependency/ADR] wi-50/wi-56 の実装 contract を確認し、ADR-056 の assertion profile、policy ownership、token-exchange mapping を確定する。
- [ ] T002 [SCL] assertion model、managed cross-app policy、exchange interface/errors/events/constraints/contracts/scenarios を追加して再生成する。
- [ ] T003 [Application] source→destination resource allowlist、user/agent policy、key metadata と管理 use case を実装する。
- [ ] T004 [OAuth2] assertion verifier、jti replay store、subject-token dispatch、RAR/actor-chain downscope を既存 token exchange に追加する。
- [ ] T005 [Admin/Consent UI] cross-app policy 管理と、必要な場合の user consent に source/destination/agent/action を表示する。
- [ ] T006 [Verify] forged/wrong-aud/expired/replayed assertion、cross-tenant、policy revoke、actor depth、destination audience を E2E 検証する。

## Verification
- `just test-go`
  - reason: assertion 検証、許可ポリシー照合 (未許可は拒否)、audience 限定、取消後拒否の境界。
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just build-ui`
- 手動: app A の assertion で app B 向け token を取得 → 許可ポリシー外の app は拒否 → ポリシー取消後は拒否されることを確認する。

## Risk Notes
app 間アクセスの自動仲介は、許可関係の検証が緩いと横方向の権限拡大に直結する。
identity assertion の issuer / audience / 有効期限を厳格に検証し、管理者が登録した
app-to-app 許可ポリシーに合致する場合のみ交換する (fail-closed)。draft 段階の仕様のため
対象改訂を ADR で固定し、互換性変化に追従する。

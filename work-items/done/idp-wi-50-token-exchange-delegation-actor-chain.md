---
id: idp-wi-50-token-exchange-delegation-actor-chain
title: "Token Exchange (RFC 8693) によるエージェントの委譲・代行と actor チェーン"
created_at: 2026-06-22
authors: ["tn"]
status: completed
risk: high
---

# Motivation
AI エージェントの中核ユースケースは「ユーザーに代わって (on-behalf-of) API や
データへアクセスする」ことである。これを安全に表現する標準が OAuth 2.0 Token
Exchange (RFC 8693) で、subject token を別の token に交換し、`act` (actor) /
`may_act` claim で「エージェント A がユーザー B を代行している」関係と、さらに
サブエージェントへ連鎖する委譲チェーンを表現できる。委譲 (delegation) と
なりすまし (impersonation) を区別し、交換後トークンの送信先 (audience) を
Resource Indicators (RFC 8707) で特定リソース / ツールに限定することで、横取りや
リソース間の再利用 (token の取り違え) を防ぐ。

idmagic は既に DPoP・sender constraint・cnf を持ち、impersonation セッション
イベント (SessionImpersonationStarted/Ended) の素地もある。本 WI は
[[wi-49-agent-identity-first-class-principal]] の Agent を actor 主体として、
/token に urn:ietf:params:oauth:grant-type:token-exchange を実装し、actor チェーンと
audience 限定を一級の保証として導入する。これは後続の RAR
([[wi-51-rich-authorization-requests-agent-scopes]])・Cross-App Access
([[wi-57-cross-app-access-identity-assertion-grant]])・workload federation
([[wi-54-workload-identity-federation-spiffe]]) すべての交換基盤となる。

# Scope
- **decision**:
  - 新規 ADR [[ADR-049]]: delegation と impersonation の区別と既定方針 (delegation を基本、 impersonation は明示許可時のみ)、`act` のネスト方法と最大委譲深さ、`may_act` による委譲許可の事前宣言、subject_token_type / requested_token_type の対応、 Resource Indicators (RFC 8707) による audience 限定の必須化を確定する。
- **scl**:
  - 新規 model: TokenExchangeRequest / TokenExchangeResponse / ActorClaim (act) / MayActClaim。AccessTokenClaims / IdTokenClaims に `act` (ネスト可) と `may_act`、audience 限定を追加する。新規 GrantType: TokenExchange。
  - 新規 event: TokenExchanged / DelegationChainExtended / TokenExchangeRejected。
  - 新規 permission: TokenGrantTokenExchange。AuthZEN ポリシーに「この client / agent が要求した actor・audience の交換を許可するか」を追加する。
- **go**:
  - /token の token-exchange grant: subject / actor token の検証、delegation 時の `act` 連鎖構築、impersonation の明示許可確認、audience (resource) 限定、 最大委譲深さの強制を fail-closed で実装する。
  - introspection / userinfo が `act` チェーンを表示できるよう拡張する。
- **http**:
  - /token の token-exchange grant、discovery への grant_types / resource 反映。

# Out of Scope
- RAR (authorization_details) による細粒度スコープ ([[wi-51-rich-authorization-requests-agent-scopes]])。
- 内部マイクロサービス間の Transaction Tokens (将来 WI、Txn-Tokens draft)。
- 外部 IdP のアサーションを起点にした Cross-App Access ([[wi-57-cross-app-access-identity-assertion-grant]])。

# Verification
- `go test ./...` (in: idmagic)
  - reason: delegation / impersonation の区別、`act` ネスト構築、audience 限定 (別 resource で拒否)、最大深さ超過の拒否、`may_act` 未許可の拒否の境界。
- `golangci-lint run ./...` (in: idmagic)
- `go build ./...` (in: idmagic)
- 手動: エージェントがユーザーの token を on-behalf-of で交換 → audience 限定の確認 → サブエージェントへ再交換し act が連鎖することを確認する。

# Risk Notes
Token Exchange は委譲・なりすましを扱うため、誤実装は権限昇格や横展開に直結する。
既定を delegation とし、impersonation は明示許可時のみ。audience 限定 (RFC 8707) を
必須にして交換後トークンの再利用を断つ。`act` の最大深さを設け、`may_act` /
AuthZEN ポリシーを満たす交換のみ成功させる (すべて fail-closed)。

# Completion
- **Completed At**: 2026-06-22
- **Summary**:
  ADR-049 に基づき、OAuth 2.0 Token Exchange (RFC 8693) を spec-first で実装した。
  /token に grant_type=urn:ietf:params:oauth:grant-type:token-exchange を追加し、
  subject_token / actor_token を自己発行トークンとして検証 (署名・active)、委譲のみ
  (delegation only) で結果トークンの sub を subject の sub に保ち、現在アクター
  (actor_token.sub または認証済み client_id) を最外 act とし subject_token の既存 act を
  内側にネストする。委任の最大深さ (既定 3) を超える交換と、subject_token の may_act に
  反するアクターは拒否する (fail-closed)。Resource Indicators (RFC 8707, constrained) を
  必須・単一とし発行トークン aud=[resource] に限定、scope は subject_token の部分集合に
  downscope する。refresh token は発行しない。発行時は DPoP/mTLS の sender constraint を
  引き継ぐ。`AccessTokenClaims` / introspection に act / aud を、署名・検証経路に
  act/may_act/array-aud を追加し、TokenGrantTokenExchange 権限と TokenExchanged /
  TokenExchangeRejected イベントを導入した。
- **Verification Results**:
  - `go build ./...` (in: idmagic)
    - result: ok
  - `go test ./...` (in: idmagic)
    - result: ok, no failures (act チェーン構築/ネスト・最大深さ・inactive subject・may_act・scope downscope・single-resource・no-refresh の use case テストと、自己発行トークンを発行→交換し aud==[resource]/act/issued_token_type を確認する e2e を含む)
  - `golangci-lint run ./...` (in: idmagic)
    - result: ok, 0 issues
  - `bun run yaml-check:scl` (in: tools)
    - result: ok
  - `bun run yaml-check:work-items` (in: tools)
    - result: ok
- **Affected Guarantees State**:
  - 委譲の追跡性: pass。delegation only で常に act を積み、subject_token の既存 act をネストして user→actor の連鎖を保持する。
  - audience 限定: pass。resource 必須・単一で aud=[resource]、scope は subject の部分集合に downscope する。
  - 委譲制御: pass。may_act 不一致と最大深さ超過を fail-closed で拒否。grant 宣言は token_handler の中央ゲート (client.GrantTypes 未宣言なら unauthorized_client) で強制する。
  - 監査: pass。TokenExchanged (actorSub/subjectSub/audience/delegationDepth) と TokenExchangeRejected を outbox (oauth2.token.v1) へ emit する。

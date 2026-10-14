---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# Token Exchange (RFC 8693) によるエージェントの委譲・代行と actor チェーン

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

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。

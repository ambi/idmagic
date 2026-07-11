---
depends_on: []
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-06-22
---

# CIBA による human-in-the-loop なエージェント行動承認

## Motivation
自律エージェントが高リスクな行動 (送金・データ削除・外部公開など) を行う前に、
人間が帯域外 (out-of-band) で承認できる仕組みが要る。OpenID Client-Initiated
Backchannel Authentication (CIBA) はこの「非同期・decoupled な承認」を標準化し、
Auth0 の "Async Authorization"・Okta などエージェント向けに広く採用されている。
エージェント (consumption device) が承認を起票し、ユーザーの authentication device
(スマホ等) に push して承認/拒否を得る。

idmagic は対話的な認可・同意・step-up は持つが、呼び出し元とユーザーが分離した
非同期承認フローを持たない。本 WI は CIBA Core (poll / ping / push の token
delivery) を実装し、エージェントが行動前に人間承認を得る経路を提供する。RAR
([[wi-51-rich-authorization-requests-agent-scopes]]) と組み合わせ「何を承認するか」を
構造的に提示し、既存の通知 (email sender) や将来の push と接続する。

## Scope
- **decision**:
  - 新規 ADR [[ADR-051]]: token delivery mode (poll を既定、ping / push は将来)、binding_message と user_code の扱い、認証要求の有効期限・ポーリング間隔、authentication device への 通知チャネル (email / push)、CIBA と既存 step-up / consent の責務分担を確定する。
- **scl**:
  - 新規 model: BackchannelAuthRequest / BackchannelAuthResponse / AuthReqId / BackchannelAuthState / TokenDeliveryMode。
  - 新規 event: BackchannelAuthRequested / BackchannelAuthApproved / BackchannelAuthDenied / BackchannelAuthExpired。
  - 新規 interface: BackchannelAuthenticate (/bc-authorize)、token endpoint の urn:openid:params:grant-type:ciba。permission TokenGrantCiba。
- **go**:
  - /bc-authorize で auth_req_id を発行し、user の承認状態に応じて /token が authorization_pending / slow_down / 成功 / 失効 を fail-closed で返す。
  - 承認要求のユーザー向け提示 (binding_message・要求 scope / authorization_details)。
- **http**:
  - /bc-authorize エンドポイント、CIBA grant、discovery への backchannel メタデータ反映。
- **ui**:
  - end-user: 保留中の承認要求の一覧・承認 / 拒否画面。

## Out of Scope
- push 通知基盤そのものの構築 (まず poll + email 通知、push は将来)。
- FCM / APNs 等モバイル SDK の同梱。
- signed authentication request (CIBA の JWT request) は将来拡張。

## Verification
- `just test-go`
  - reason: authorization_pending / slow_down / 承認後成功 / 拒否・期限切れ拒否の状態遷移境界。
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just build-ui`
- 手動: エージェントが /bc-authorize → ユーザーが承認 UI で承認 → /token が成功。拒否時は token が出ないことを確認する。

## Risk Notes
非同期承認は状態遷移 (pending / approved / denied / expired) とポーリング制御を
正しく扱う必要があり、緩いと未承認のまま token が出る恐れがある。token は承認成立まで
必ず保留し、slow_down / 期限切れを厳密に扱う (fail-closed)。binding_message を提示して
別要求の取り違え承認を防ぐ。

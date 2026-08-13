---
depends_on: [wi-49-agent-identity-first-class-principal, wi-50-token-exchange-delegation-actor-chain, wi-51-rich-authorization-requests-agent-scopes]
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-06-22
change_kind: feature
initial_context:
  specification:
    - spec/contexts/oauth2/SPECIFICATION.md#REQ-OAUTH2-027
    - spec/contexts/oauth2/SPECIFICATION.md#REQ-OAUTH2-012
    - spec/contexts/oauth2/SPECIFICATION.md#REQ-OAUTH2-032
    - spec/contexts/oauth2/SPECIFICATION.md#RFC8628-POLLING
  typespec:
    - IdMagic.Contract.DeviceAuthorization
    - IdMagic.Contract.DeviceCodeState
    - IdMagic.Contract.TokenRequest
    - IdMagic.Contract.AccountConsentListResponse
    - IdMagic.OAuth2.Operations.DeviceAuthorization
    - IdMagic.OAuth2.Operations.ListMyConsents
  source:
    - backend/oauth2/device/domain/device_authorization.go
    - backend/oauth2/device/usecases/device_flow.go
    - backend/oauth2/device/ports/device_code_store.go
    - backend/oauth2/db_postgres/device_code_store.go
    - backend/oauth2/handlers_http/token_handler.go
    - backend/oauth2/handlers_http/routes.go
    - backend/shared/spec/device_code_machine.go
    - backend/shared/spec/enums.go
    - backend/shared/spec/discovery.go
    - backend/idmanagement/agent/domain/agents.go
    - backend/authentication/deps_http/account_helpers.go
    - backend/authentication/handlers_http/account_consents_handler.go
    - frontend/src/features/account/AccountApplicationsPage.tsx
  tests:
    - backend/oauth2/device/usecases/device_flow_test.go
    - backend/oauth2/handlers_http/device_handler_test.go
    - backend/oauth2/db_postgres/ephemeral_stores_test.go
  stop_before_reading:
    - backend/saml
    - backend/provisioning
    - backend/sourcing
affected_spec:
  - { path: spec/contexts/oauth2/SPECIFICATION.md, requirement: REQ-OAUTH2-041 }
  - { path: spec/contexts/oauth2/SPECIFICATION.md, requirement: REQ-OAUTH2-042 }
  - { path: spec/contexts/oauth2/SPECIFICATION.md, requirement: REQ-OAUTH2-043 }
  - { path: spec/contexts/oauth2/SPECIFICATION.md, requirement: CIBA-CORE-BACKCHANNEL-REQUEST }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.ApprovalRequest }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.ApprovalRequestState }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.BackchannelAuthenticationRequest }
  - { path: spec/contexts/oauth2/models.tsp, symbol: IdMagic.Contract.BackchannelAuthenticationResponse }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.BackchannelAuthenticate }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.ListMyApprovalRequests }
  - { path: spec/contexts/oauth2/main.tsp, symbol: IdMagic.OAuth2.Operations.DecideMyApprovalRequest }
  - { path: spec/contexts/tenancy/models.tsp, symbol: IdMagic.Contract.NotificationTemplateKey }
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
非同期承認フローを持たない。本 WI は CIBA Core の poll mode を実装し、エージェントが
行動前に人間承認を得る経路を提供する。RAR
([[wi-51-rich-authorization-requests-agent-scopes]]) と組み合わせ「何を承認するか」を
構造的に提示し、既存の通知 (email) と接続する。

**モデルと実行の乖離**: idmagic は `AgentKind` に `Supervised` (人間の監督下で実行する区分) を
持ち、管理者はエージェントをそう宣言できる。しかし**その監督を実行時に強制する仕組みが
一つも無い**ため、`Supervised` は現状ただのラベルである。本 WI は、宣言された区分に対して
初めて実際の監督経路 (人間の承認なしにはトークンが出ない grant) を与える。ただし
「Supervised なら全 grant で承認必須」という強制は本 WI では入れない (Design 参照)。

**依存の状態**: `depends_on` の 3 件 ([[wi-49-agent-identity-first-class-principal]] /
[[wi-50-token-exchange-delegation-actor-chain]] / [[wi-51-rich-authorization-requests-agent-scopes]])
はすべて完了済みで、本 WI はアンブロック済みである。
[[wi-59-agent-governance-guardrails-audit-inventory]] が本 WI を明示的に待っており
(閾値超過時の「承認へ昇格」経路)、それまでは承認要求を deny 扱いにせざるを得ない。
エージェントのガバナンス層はここが埋まるまで進まない。この優先度判断は
[[wi-369-agent-capability-survey-2026-08]] の棚卸しによる。

## Scope
- **spec (TypeSpec)**:
  - 新規 model: `ApprovalRequest` / `ApprovalRequestState` / `BackchannelAuthenticationRequest` /
    `BackchannelAuthenticationResponse` / `AccountApprovalRequest` /
    `AccountApprovalRequestListResponse` / `AccountApprovalDecisionRequest` /
    `UnknownUserIdError`。
  - 新規 event: `BackchannelAuthRequested` / `BackchannelAuthApproved` /
    `BackchannelAuthDenied` / `BackchannelAuthExpired`。
  - 新規 interface: `BackchannelAuthenticate` (`POST /bc-authorize`)、
    `ListMyApprovalRequests` / `DecideMyApprovalRequest` (account API)、
    token endpoint の `urn:openid:params:grant-type:ciba`。
- **spec (SPECIFICATION.md)**: CIBA Core の Standards、`ApprovalRequestLifecycle` の
  state transition table、Glossary、Design (承認能力としての CIBA、AARP 互換の一般形)、
  normative scenario REQ-OAUTH2-041 / 042 / 043。
- **go**: `oauth2/approval` slice (domain / ports / usecases / db_memory) と
  `db_postgres` adapter。`/bc-authorize` が auth_req_id を発行し、承認状態に応じて
  `/token` が authorization_pending / slow_down / 成功 / 失効 / 拒否を fail-closed で返す。
- **http**: `/bc-authorize` エンドポイント、CIBA grant、discovery への backchannel メタデータ反映。
- **notification**: 新規 template key `agent_action_approval_request` で承認要求を本人へ通知する。
- **ui**: end-user の保留中承認要求一覧と承認 / 拒否画面 (agent・actor・scope・
  authorization_details・binding_message を提示)。

## Out of Scope
- push / ping の token delivery mode (poll のみ。ping は adapter seam を残すだけ)。
- push 通知基盤そのものの構築。FCM / APNs 等モバイル SDK の同梱。
- signed authentication request (CIBA の JWT request) は将来拡張。
- `user_code` パラメータ (discovery で非対応と広告する。Design 参照)。
- 「Supervised agent は全 grant で承認必須」の強制 (Design 参照。
  [[wi-59-agent-governance-guardrails-audit-inventory]] が所有する)。

## Design

### CIBA は認証方式ではなく「承認能力」として OAuth2 に統合する
CIBA を独立した認証フローとしてではなく、client の token request を人間の判断へ昇格する
approval capability として実装する。同意 (Consent) の意味論は変えない — Consent は
`(subject, client_id)` の scope 付与であり長期的、承認要求は 1 回の行動に対する短命な判断である。
step-up は承認**操作**そのものを守る (承認画面で再認証を要求する) のであって、CIBA が
step-up を置き換えるわけではない。

### 決定の意味論と CIBA の輸送 bookkeeping を分離する
OpenID AuthZEN WG の Access Request and Approval Profile (AARP) は「認可がまだ下せない —
前提条件を先に満たす必要がある」状況を要求 → 追跡 → 充足 → 再評価の一般形として抽象化して
おり、CIBA はその最も普及した実装形にあたる。将来 AARP へ寄せる余地を残すため、決定を保持する
集約は `ApprovalRequest` という一般名で持ち、状態遷移と判断内容を
`auth_req_id`・poll interval・slow_down といった CIBA 固有の輸送 bookkeeping から分離する。

具体的には、`ApprovalRequest.ID` は UUID であり、CIBA の `auth_req_id` は adapter が生成する
32 バイトのランダム bearer secret で、store には SHA-256 ハッシュ (`AuthReqIDHash`) だけを
索引として持つ。account UI の URL は UUID を使い、bearer secret を人間の画面へ出さない。
polling 台帳 (`IntervalSeconds` / `LastPolledAt`) は同じ record 上に併置する。承認と poll の競合を
一つの store 境界で直列化できるようにするためであり、輸送 bookkeeping であることを型とコメントで
明示して、決定の意味論とは混ぜない。

### 状態機械は一方向で、承認済みは一度しかトークンにならない
`Pending → Approved / Denied / Expired`、`Approved → Consumed` の一方向遷移。Consumed は
Approved からのみ到達する終端で、トークン発行は store の CAS (`state='approved'` の行だけを
`consumed` にして返す) を通す。device code の `Exchange` と同じ形で、並行 poll が二重発行に
ならないことを保存層で担保する。

### poll のみ / user_code なし / 期限とポーリング間隔
token delivery mode は poll だけを実装し、discovery では
`backchannel_token_delivery_modes_supported: ["poll"]` と広告する。ping は「承認確定時に
client へ通知する」adapter を後から差せる形にとどめ、本 WI では配線しない。

`user_code` は非対応と広告する (`backchannel_user_code_parameter_supported: false`)。
idmagic の承認画面は本人の認証済み session + step-up の背後にあり、そこへ二つ目の、
より弱い共有秘密を足す理由がない。

`scope` は CIBA Core に従って必須かつ `openid` を含まなければならない。承認対象 hint は
`login_hint` / `id_token_hint` のちょうど一方だけを受理し、欠落・同時指定は
`invalid_request` とする。`binding_message` は 64 文字を上限とし、超過時は
`invalid_binding_message` で拒否する。

有効期限は既定 300 秒、client が正の `requested_expiry` で 600 秒まで指定できる。
非正または 600 秒超は `invalid_request` で拒否する。
ポーリング間隔は device code と同じ既定 5 秒 / `slow_down` ごとに +5 秒で、
`DefaultDeviceCodePolling()` を共有する (二つの異なるポーリング規約を持たない)。

### 承認対象ユーザーの解決と、承認操作の保護
`/bc-authorize` は `login_hint` (username / email) または `id_token_hint` で承認対象 User を
解決する。解決できない・非 active・別テナントはすべて `unknown_user_id` で fail-closed に
拒否し、User の存在有無を error の差で開示しない。

承認 / 拒否は account portal の本人 session + step-up + CSRF を必須にする。承認要求は解決済み
User に束縛され、他人の承認要求は一覧にも出ず、ID を直接指定しても 403 になる。画面は
binding_message だけで内容を代替せず、agent 名・client 名・要求 scope・
authorization_details を並べて表示する。

### 保存先は Postgres (Valkey ではない)
起票時の Plan は poll 状態を Valkey に置くと書いていたが、本リポジトリに Valkey adapter は
存在せず、揮発性の OAuth2 状態 (authorization request / authorization code / PAR / device code /
replay) はすべて `UNLOGGED` な Postgres テーブル + memory adapter という一つの規約で
扱われている。承認要求だけを別の保存技術へ出す理由がないため、既存規約に合わせて
`oauth2_approval_requests` (UNLOGGED) を足し、期限切れ行は既存の ephemeral sweep に載せる。

### Supervised の強制は本 WI では入れない
「Supervised agent は client_credentials では直接トークンを取れない」という強制は魅力的だが、
`normalizeAgentKind` の既定が `Supervised` であり、これまでに作られた Agent は明示指定が
無い限りすべて Supervised である。いま強制すると既存デプロイの agent が一斉にトークンを
失う。本 WI は「人間の承認なしにはトークンが出ない grant」を用意するところまでを担い、
どの agent にそれを義務付けるかの判定は閾値・ポリシーを持つ
[[wi-59-agent-governance-guardrails-audit-inventory]] に委ねる。

### 却下した代替案
- **device code flow の再利用**: 状態機械は似ているが、device flow は「ユーザーが手元の
  コードを入力する」ことが本質で、CIBA は「サーバがユーザーを特定して通知する」ことが本質
  である。user_code 索引・verification_uri を持つ record に承認対象 User と
  authorization_details を後付けすると、どちらのフローにとっても不自然な集約になる。
- **Consent への相乗り**: Consent は長期の scope 付与であり、失効 (revoke) の意味論も
  GDPR 由来の再同意期限も持つ。1 回の行動判断をそこへ載せると、両方の意味論が壊れる。

## Plan
1. TypeSpec (models / operations) と oauth2 `SPECIFICATION.md` (Glossary / Standards /
   State Transitions / Design / Scenarios) を先に変更し、`just check-spec` を通す。
2. Domain: `shared/spec` に `ApprovalRequestState` と遷移表、`GrantCiba`、discovery メタデータ。
   `oauth2/approval/domain` に集約と検証・期限判定。
3. Use Cases: `oauth2/approval/usecases` に Start / Poll(Exchange) / Approve / Deny / List。
   audit event は既存 `Emit` に載せる。
4. Adapters: memory / postgres store、`/bc-authorize` handler、token endpoint の ciba grant、
   account API handler、bootstrap 配線、schema。
5. UI: account portal に承認要求ページを足し、i18n 辞書と unit test を付ける。

## Tasks
- [x] T001 [Spec] TypeSpec に CIBA / 承認要求の model・operation・event・error を追加し、oauth2 `SPECIFICATION.md` に Glossary・CIBA Core Standards・`ApprovalRequestLifecycle`・Design・REQ-OAUTH2-041/042/043 を追加して `just check-spec` を通す。
- [x] T002 [Domain] `shared/spec` の `ApprovalRequestState`・遷移表・`GrantCiba`・discovery メタデータと、`oauth2/approval/domain` の集約・検証・期限判定を RED 先行で実装する (REQ-OAUTH2-042)。
- [x] T003 [Store/Usecases] memory / postgres の `ApprovalRequestStore` (一度きり消費の CAS 込み) と Start / Exchange / Approve / Deny / List use case、audit event、通知を実装する (REQ-OAUTH2-041)。
- [x] T004 [OAuth HTTP] client 認証付き `/bc-authorize` と token endpoint の ciba grant、discovery の backchannel メタデータ、bootstrap 配線を追加する (REQ-OAUTH2-041, REQ-OAUTH2-042)。
- [x] T005 [Account UI] step-up / CSRF 付きの account API と、agent・client・scope・authorization_details・binding_message を提示する承認一覧 / 承認 / 拒否画面を追加する (REQ-OAUTH2-043)。
- [x] T006 [Verify] 状態遷移境界・並行 poll・slow_down・replay・別ユーザー / 別テナント・承認後の token claim を自動テストで検証し、`just verify` を通す。

## Verification
- `just test-go`
  - reason: authorization_pending / slow_down / 承認後成功 / 拒否・期限切れ・二重消費拒否の状態遷移境界。
- `just lint-go`
- `just build-go`
- `just typecheck-ui`
- `just test-ui-unit`
- `just build-ui`
- `just check-spec`
- `just verify`

## Risk Notes
非同期承認は状態遷移 (pending / approved / denied / expired / consumed) とポーリング制御を
正しく扱う必要があり、緩いと未承認のまま token が出る恐れがある。token は承認成立まで
必ず保留し、slow_down / 期限切れを厳密に扱う (fail-closed)。二重発行は保存層の CAS で
止める。binding_message と要求内容の全項目を提示して、別要求の取り違え承認を防ぐ。
`auth_req_id` は bearer secret なのでハッシュのみを保存し、画面にも監査ログにも出さない。

## Completion
- **Completed At**: 2026-08-14
- **Summary**:
  CIBA Core poll mode による非同期承認を仕様・domain・use case・memory/Postgres adapter・OAuth HTTP・account API/UI まで一貫して追加した。client と tenant に束縛した承認要求を human-in-the-loop で承認または拒否でき、承認後だけ access token と ID token を一度だけ発行する。期限、poll 間隔、Agent 状態、step-up、CSRF、所有者、並行消費を fail-closed に検証する。
- **Verification Results**:
  - `just spec-diff main` - passed; REQ-OAUTH2-041/042/043、ApprovalRequestLifecycle、CIBA TypeSpec 宣言の意味差分を確認
  - `just verify` - passed

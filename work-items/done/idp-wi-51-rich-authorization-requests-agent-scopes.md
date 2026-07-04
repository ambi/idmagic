---
id: idp-wi-51-rich-authorization-requests-agent-scopes
title: "Rich Authorization Requests (RFC 9396) でエージェント権限を細粒度に束縛する"
created_at: 2026-06-22
authors: ["tn"]
status: completed
risk: medium
---

# Motivation
エージェントへの権限付与は粗い OAuth scope (例: `payments`) では危険で、
「口座 X から最大 $100 まで送金」「この 3 文書のみ閲覧」のように対象・上限・
条件を構造的に束縛したい。これを表現する標準が Rich Authorization Requests
(RFC 9396) の `authorization_details` で、最小権限 (least privilege) と
ダウンスコープを構造化 JSON で要求・同意・トークン反映できる。

idmagic は scope ベースの同意までで、構造化された細粒度権限を持たない。本 WI は
/authorize と /par、および token-exchange
([[wi-50-token-exchange-delegation-actor-chain]]) で `authorization_details` を
受理・検証・同意・トークンへ反映し、エージェントが「必要な分だけ」を要求できる
ようにする。これにより人間の関与 (CIBA、[[wi-52-ciba-async-human-approval]]) や
ガバナンス ([[wi-59-agent-governance-guardrails-audit-inventory]]) と組み合わせた
「上限つき委譲」が成立する。

# Scope
- **decision**:
  - 新規 ADR [[ADR-050]]: 対応する authorization_details type の定義方法 (tenant ごとに登録可能な type スキーマ)、同意 UI での提示方法、token への反映と検証点 (resource server が 何を信頼するか)、token-exchange 時の details ダウンスコープ規則を確定する。
- **scl**:
  - 新規 model: AuthorizationDetail / AuthorizationDetailType / AuthorizationDetailsSchema。 AuthorizationRequest / Consent / AccessTokenClaims に authorization_details を追加する。
  - 新規 event: AuthorizationDetailsRequested / AuthorizationDetailsConsented / AuthorizationDetailsRejected。
  - 新規 permission: AdminAuthorizationDetailTypesManage。
- **go**:
  - authorization_details の構文・登録済み type 適合・ダウンスコープ (要求は 同意済みの部分集合か) を Zog schema で fail-closed に検証する。
  - token-exchange で交換後 details を元 details の部分集合に限定する。
- **http**:
  - /authorize・/par・/token が authorization_details を受理し、discovery に対応 type を広告する。
- **ui**:
  - end-user 同意 UI で authorization_details を人間が読める形で提示する。

# Out of Scope
- 個別業務ドメインの type スキーマ (口座・送金等) の網羅的定義。
- resource server 側の details 解釈実装 (idmagic は発行と検証点の提供まで)。

# Verification
- `go test ./...` (in: idmagic)
  - reason: details の構文検証、type 適合、同意との部分集合判定、交換時ダウンスコープの境界。
- `golangci-lint run ./...` (in: idmagic)
- `go build ./...` (in: idmagic)
- `bun --cwd idmagic/ui typecheck`
- `bun --cwd idmagic/ui build`
- 手動: authorization_details つきで認可 → 同意 UI で内容確認 → token に反映 → 同意外の details 要求が拒否されることを確認する。

# Risk Notes
authorization_details は自由度が高く、検証が緩いと過大な権限付与や同意逸脱を招く。
受理する type を事前登録スキーマに限定し、同意との部分集合判定・交換時のダウンスコープを
fail-closed で強制する。粗い scope との重複は ADR で優先順位を定める。

# Completion
- **Completed At**: 2026-06-24
- **Summary**:
  ADR-050 に基づき RFC 9396 Rich Authorization Requests を spec-first で実装した。
  テナントが事前登録した AuthorizationDetailType の検証スキーマ (フィールドごとに
  set / at_most / enum / exact の半順序) に対し、各 authorization_details を
  構文・必須・許可値・未登録フィールド拒否で fail-closed に検証する (ValidateAgainstType)。
  /authorize・/par・/token で受理し、検証済み詳細を AuthorizationRequest → 同意 Consent →
  AuthorizationCodeRecord → access token の authorization_details claim まで貫通させる。
  構造化詳細があるときは粗い scope の過去同意で自動スキップせず明示同意を必須とする。
  token-exchange (RFC 8693) では要求詳細を subject_token の詳細の部分集合に限定し
  (DetailsSubsetOf)、詳細なしの交換は元の詳細を保持のみ・決して拡張しない。introspection は
  検証済み JWT から詳細を抽出して返し (RS の検証点)、discovery は Enabled な type を
  authorization_details_types_supported として広告する。type 定義は新 permission
  AdminAuthorizationDetailTypesManage 配下の admin API / 管理 UI で CRUD でき、同意画面は
  登録 type の表示テンプレートで詳細を人間可読に提示する。
- **Verification Results**:
  - `go build ./...` (in: idmagic)
    - result: ok
  - `go test ./...` (in: idmagic)
    - result: ok, no failures (検証の拒否境界・同意との部分集合・交換ダウンスコープ・token claim・admin API の各テストを含む)
  - `golangci-lint run ./...` (in: idmagic)
    - result: ok, 0 issues
  - `bun --cwd idmagic/ui typecheck`
    - result: ok
  - `bun --cwd idmagic/ui build`
    - result: ok
  - `bun run yaml-check:scl` (in: tools)
    - result: ok
- **Affected Guarantees State**:
  - 最小権限: pass。エージェントは登録 type の検証スキーマで対象・上限を束縛した詳細しか得られず、未登録 type・スキーマ不適合は拒否する。
  - 同意の一致: pass。発行トークンの詳細は同意済み詳細に基づき、構造化詳細があるときは明示同意を必須化して過去 scope 同意での自動承認を許さない。
  - ダウンスコープ: pass。token-exchange は要求詳細を subject_token の部分集合に限定し、詳細なしの交換は保持のみで拡張しない。
  - 監査: pass。AuthorizationDetailsRequested / Consented / Rejected を emit する。

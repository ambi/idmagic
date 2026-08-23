---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-16
priority: p0
depends_on: [wi-52-ciba-async-human-approval]
change_kind: feature
initial_context:
  specification:
    - docs/contexts/oauth2/scenarios.md#REQ-OAUTH2-041
    - docs/contexts/oauth2/scenarios.md#REQ-OAUTH2-042
    - docs/contexts/oauth2/scenarios.md#REQ-OAUTH2-046
    - docs/contexts/identity-management/scenarios.md#REQ-IDMANAGEMENT-009
  typespec:
    - IdMagic.Contract.AgentKind
    - IdMagic.Contract.AgentRegisterRequest
    - IdMagic.Contract.AccessTokenClaims
  source:
    - backend/oauth2/usecases/agent_issuance.go
    - backend/oauth2/handlers_http/token_handler.go
    - backend/oauth2/token/usecases/exchange_token.go
    - backend/oauth2/approval/usecases/approval_flow.go
    - backend/idmanagement/agent/usecases/admin_agents.go
    - backend/idmanagement/domain/enums.go
  tests:
    - backend/oauth2/usecases
    - backend/oauth2/token/usecases
    - backend/oauth2/handlers_http
    - backend/idmanagement/agent/usecases
  stop_before_reading: [frontend]
affected_spec:
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-050 }
  - { path: docs/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-041 }
  - { path: docs/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-009 }
  - { path: spec/contexts/identity-management/models.tsp, symbol: AgentKind }
  - { path: spec/contexts/identity-management/models.tsp, symbol: Agent }
  - { path: spec/contexts/oauth2/models.tsp, symbol: AccessTokenClaims }
---

# `AgentKind.Supervised` のエージェントには人間の承認なしにトークンを発行しない

## Motivation

`AgentKind` は `Autonomous` と `Supervised` を区別し、仕様は後者を「Agent が人間の監督下で実行する区分」と定義している。しかし監督を実行時に強制する経路は一つも存在しない。`Supervised` は Go 全体で `backend/idmanagement/domain/enums.go` の宣言と一件のテストにしか現れず、`backend/oauth2` のトークン発行経路は一度も参照しない。`Autonomous` として登録しても `Supervised` として登録しても、得られるトークンは完全に同一である。

つまり `Supervised` は現在ただのラベルである。管理者が監督下だと宣言したエージェントが、誰の承認も経ずに自律実行する。宣言と挙動が食い違っている以上、この区分を根拠にした運用判断はすべて誤りうる。エージェントの区分を監査画面で見て安心する運用者を作ってしまう点で、区分が無い状態より悪い。

この穴は [[wi-52-ciba-async-human-approval]] の完了 (2026-08-14) によって埋められる状態になった。CIBA の `ApprovalRequest` は既に `agent_id` を持ち、承認が成立するまでトークンを発行しない (REQ-OAUTH2-041)。必要なのは新しい承認機構ではなく、`AgentKind` と既存の承認機構を結ぶ一本の線である。

OAuth2 の仕様は「どのエージェントが承認を要するか」の判断をガバナンス層へ明示的に委ねている。ただし [[wi-59-agent-governance-guardrails-audit-inventory]] が扱うのは閾値超過時に承認へ降格するという条件付きの規則であり、本 work item が扱うのは `Supervised` なら常に承認を要するという無条件の規則である。後者は前者を待つ必要がなく、前者よりはるかに安い。

## Scope

- `docs/contexts/oauth2/scenarios.md` に、`Supervised` な Agent へのトークン発行が人間の承認を経ることを求める規範シナリオを追加する。対象は Agent が主体となるすべての発行経路 — `client_credentials`、トークン交換、ワークロード ID 連携による交換 — とする。
- 承認が成立していない場合の応答を確定する。CIBA の承認要求を暗黙に生成するのか、承認要求の提示を要求して拒否するのかを `## Design` で決め、フェイルクローズを守る。
- `spec/contexts/identity-management/models.tsp` に `model Agent` 集約を追加する。現在この集約はリクエスト / レスポンス / イベントとしてしか仕様に存在せず、`AgentKind` が実行時の意味を持つ本変更が [[wi-369-agent-capability-survey-2026-08]] の設定した再評価条件 (「エージェント集約に仕様変更が入る時に併せて解消する」) を発火させる。
- `spec/contexts/oauth2/models.tsp` の `AccessTokenClaims` に `agent_id` クレームを宣言する。Go (`backend/shared/security/tokens_jose/jwt_signer.go`) では既に発行しているが仕様に無く、本変更が参照するため同時に解消する。
- Go 側でトークン発行境界に `AgentKind` の判定を追加する。`Agent` の状態ゲート (`Active` / `Disabled` / `Killed`) と同じ位置で、同じフェイルクローズの規則に従う。
- 拒否と承認をイベントとして残し、監査から `AgentKind` を根拠にした判断が追えるようにする。
- `AgentRegisterRequest.kind` を必須にする。区分が実行時の意味を持つ以上、省略時の既定値をどちらに倒しても誤りになるため。

## Out of Scope

- 閾値・予算・レート超過を根拠にした条件付きの承認要求。[[wi-59-agent-governance-guardrails-audit-inventory]] が持つ。
- 承認者の決定規則の高度化 (承認者の委任、複数承認、エスカレーション)。現行どおり対象ユーザー本人のステップアップ認証済みセッションに限る (REQ-OAUTH2-043)。
- CIBA の `ping` / `push` モード。wi-52 が poll のみに限定した判断を維持する。
- **AAuth / Agent Authorization Grant** (`draft-…-aauth`、リダイレクトを使えない経路でエージェントがユーザー代行トークンを得る OAuth 2.1 拡張) の実装。解こうとしている問題は CIBA が既に解いており、個人 draft の段階で二つ目の非リダイレクト承認経路を持つと、承認記録が二重化して監査が分岐する。IETF OAuth WG に採択され、かつ CIBA と異なる要求が現れた時点で再評価する。
- 既存 `Supervised` 行のデータ移行。未リリースのため移行対象の行が存在しない。`infra/schema/data-migrations` にスクリプトは置かない。
- トークン交換が発行するトークンへ、subject 側 Agent の `agent_id` を引き継ぐこと。現在 `ExchangeToken` は workload 経路でしか `agent_id` を刻印しない。本 work item は発行の可否だけを扱い、刻印の欠落は別途扱う。

## Design

### 承認が無い状態の応答 — 案 B (拒否)

`Supervised` な Agent は、人間の承認が記録された発行経路からしかトークンを得ない。現在それは CIBA だけである (REQ-OAUTH2-041 / REQ-OAUTH2-042)。ほかの経路 — `client_credentials`、トークン交換、ワークロード ID 連携による交換 — は、その Agent が `Supervised` である限り常に拒否する。

案 A (承認要求の暗黙生成) を採らない決め手は、`client_credentials` に `auth_req_id` に相当するパラメータが無いことである。案 A は「承認が成立するまで `authorization_pending` を返し続け、承認後に同じ `client_credentials` リクエストが通る」という、RFC 6749 のどこにも無いポーリング意味論を `client_credentials` に新設することになる。承認要求の重複生成を防ぐ dedupe も別途必要になる。拒否なら既存の Agent 状態ゲートと同じ位置・同じ形に収まり、クライアントが取るべき行動 (CIBA へ切り替える) も一意に決まる。

拒否は OAuth error code `unauthorized_client` とする。RFC 6749 の定義 (認証済みクライアントがその grant type を使う権限を持たない) が事実そのものであり、Agent 自身の状態や所有者を理由とする既存の `invalid_client` (REQ-OAUTH2-046) と監査上も区別できる。

判定は区分の否定形で書く。`Kind == Supervised` ではなく `Kind != Autonomous` を承認必須とする。列挙に未知の値が入った場合 (データ破損、将来の区分追加) に、承認不要側へ倒れないためである。

承認済みトークンからの派生も拒否する。`Supervised` な Agent が CIBA で得たトークンを `subject_token` としてトークン交換に持ち込む経路も通さない。一つの承認は一つのトークンに対応させる。承認を継承させると、audience と scope が hop ごとに変わりうる派生の連鎖へ一度の承認が無限に伸びる。

### `kind` を必須にする

`AgentRegisterRequest.kind` を必須にし、既定値を持たせない。省略は `AgentKindRequiredError` で拒否する。

現在 API と DB の既定は `supervised` (`normalizeAgentKind`、`agents.kind DEFAULT 'supervised'`)、管理 UI の既定は `autonomous` で食い違っている。本変更で `supervised` が実行時の意味を持つと、この食い違いは「`kind` を省いて登録したエージェントが、登録応答に現れない理由で一切トークンを得られない」という壊れ方になる。逆に既定を `autonomous` へ反転すると、フェイルクローズを目的とする変更で省略時の既定を安全でない側へ倒すことになり、監査からは意図的な宣言と単なる省略を区別できなくなる。必須にすればどちらも起きず、区分は常に意図的な宣言になる。

`normalizeAgentKind` が不正な列挙値を黙って `supervised` へ丸めている点も、同時に明示的な拒否へ変える。未リリースのため既存行の移行は不要で、DB 側の `DEFAULT 'supervised'` は取り除く (アプリケーションが常に値を与える)。

### 判定の位置

`ResolveIssuableAgent` (`backend/oauth2/usecases/agent_issuance.go`) の隣に、区分だけを見る述語を置く。`ResolveIssuableAgent` 自体には入れない。CIBA の `StartApproval` も同じ関数を通っており、そこは `Supervised` を通すべき経路だからである。既存の `AgentOwnerIsActive` を分けてある理由 (呼び出し段ごとに適した OAuth error code を選ぶ) と同じ形をとる。

適用点は 3 つ。`client_credentials` (`token_handler.go`)、トークン交換の workload 経路と self-issued 経路、そして交換を行うクライアント自身に束縛された Agent (`ExchangeToken`) である。交換に関与するどの Agent が `Supervised` でも拒否する。

### 監査

拒否側に `AgentApprovalRequired` を新設し、テナント・Agent・クライアント・区分・grant type を残す。承認側は既存の `BackchannelAuthApproved` に `agentId` を足す。これで「なぜこの Agent はトークンを得られなかったか」と「なぜこの Agent はトークンを得られたか」の両方が、区分を根拠として監査から追える。

## Plan

- 仕様を先に変え、規範シナリオと `model Agent` を含めて再生成する。
- トークン発行境界の判定は、既存の Agent 状態ゲートと同じ関数の近傍に置く。エージェント固有の関心事を複数箇所へ散らさない。
- 拒否経路のテストを先に書き、承認済み経路が通ることを後から確認する。

## Tasks

- [x] T001 [Design] 承認が無い場合の応答と既存エージェントへの適用方法を確定し、`## Design` に記録する。案 B (拒否) と `kind` 必須化を採用。
- [x] T002 [Spec] `Supervised` の承認必須シナリオ (REQ-OAUTH2-050)、`model Agent` 集約、`AccessTokenClaims.agent_id`、`AgentRegisterRequest.kind` の必須化、`AgentApprovalRequired` と `BackchannelAuthApproved.agentId` を追加して再生成する。
- [x] T003 [App] トークン発行境界に `AgentKind` 判定をフェイルクローズで実装し、拒否と承認をイベントへ出す。
  - test: `TestAgentRequiresHumanApproval` / `TestResolveIssuableAgentWithoutApproval` (backend/oauth2/usecases) — REQ-OAUTH2-050
  - test: `TestTokenClientCredentials_supervisedAgent_rejected` (backend/oauth2/handlers_http) — REQ-OAUTH2-050
  - test: `TestExchangeTokenRejectsSupervisedWorkloadAgent` / `…SupervisedSubjectAgent` / `…SupervisedActingClient` / `TestExchangeTokenAllowsAutonomousAgents` (backend/oauth2/token/usecases) — REQ-OAUTH2-050
  - test: `TestApprovalRecordsTheAgentItPermitted` (backend/oauth2/approval/usecases) — REQ-OAUTH2-050
- [x] T004 [App] `kind` を必須にし、不正な列挙値を黙って丸めるのをやめる。
  - test: `TestRegisterAgentRequiresAKind` / `TestUpdateAgentRejectsAnUnknownKind` (backend/idmanagement/agent/usecases) — REQ-IDMANAGEMENT-009
- [x] T005 [Verify] 全発行経路 (`client_credentials` / トークン交換 / ワークロード ID 交換) で `Supervised` が承認なしに発行されないこと、`Autonomous` が影響を受けないこと、判定不能時に拒否側へ倒れることを検証する。

## Verification

- `mise run verify-spec`
- `mise run test-go`
  - reason: 3 つの発行経路それぞれで `Supervised` の拒否と承認後の発行、`Autonomous` の非退行、判定不能時のフェイルクローズを確認する。
- `mise run verify`
- 手動: `Supervised` のエージェントを登録し、承認なしにトークンが得られないこと、アカウントポータルで承認するとトークンが得られること、監査に区分を根拠とした判断が残ることを確認する。

## Risk Notes

リスクは medium。未リリースのため既存エージェントの移行は問題にならないが、判定を緩めに入れると本 work item の目的そのものが失われる。判定は常にフェイルクローズとし、区分が `Autonomous` であると確認できた場合にだけ承認を省く。`kind` を必須にする API 変更は破壊的だが、未リリースであるため影響は管理 UI と seed に閉じる。

## Completion

- **Completed At**: 2026-08-22
- **Summary**:
  `AgentKind.Supervised` が実行時の意味を持つようになった。それまで宣言でしかなかった区分が、いまはトークン発行の可否を決める。人間の承認を記録しない 3 つの発行経路 — `client_credentials`、トークン交換 (self-issued / workload / 交換クライアント自身の 3 者すべて)、ワークロード ID 連携 — は、関与する Agent が `Autonomous` であると確認できない限り `unauthorized_client` で拒否する (REQ-OAUTH2-050)。`Supervised` な Agent が通れるのは、承認が `ApprovalRequest` として記録される CIBA だけになった。判定は `Kind != Autonomous` の否定形で書いてあり、列挙に無い値も承認が必要な側へ倒れる。拒否は `AgentApprovalRequired` として、承認は `BackchannelAuthApproved.agentId` として監査に残り、どちらの向きの判断も区分から追える。

  `AgentRegisterRequest.kind` は必須になった。区分が実行時の意味を持つ以上、省略時の既定値はどちらへ倒しても誤る — `supervised` なら「登録は成功したのに一切トークンを得られない Agent」が黙って生まれ、`autonomous` ならフェイルクローズを目的とする変更で省略が安全でない側へ倒れる。あわせて、不正な列挙値を黙って `supervised` へ丸めていた `normalizeAgentKind` を撤去し、`AgentKindRequiredError` と `InvalidAgentKindError` で明示的に拒否するようにした。この丸め込みが実際に何を隠していたかは既存テストが示していて、`kind: "daemon"` で登録していた 3 件のテストが今回はじめて失敗した。

  仕様側では、リクエスト / レスポンス / イベントとしてしか存在しなかった `Agent` 集約を `model Agent` として宣言し ([[wi-369-agent-capability-survey-2026-08]] の再評価条件を解消)、Go では発行済みだが仕様に無かった `AccessTokenClaims.agent_id` を宣言した。`model Agent` の追加により SharedSignals 側の published-language stub `Agent` が衝突したため、実体が同じ名前空間に来たことで不要になった stub を削除した。DB では `agents.kind` の `DEFAULT 'supervised'` を取り除いた。未リリースのため既存行のデータ移行は行っていない。

  意味差分 (`mise run spec-diff`): 追加シナリオ `REQ-OAUTH2-050`、変更シナリオ `REQ-IDMANAGEMENT-009`、追加 TypeSpec 宣言 `Agent` / `AgentKindRequiredError` / `InvalidAgentKindError` / `AgentApprovalRequired`、削除 `sharedsignals:Agent`。
- **Verification Results**:
  - `mise run check-spec` - passed
  - `mise run test-go` - passed
  - `mise run lint-go` - passed (0 issues)
  - `mise run verify` - passed
  - `mise run update-api-baseline` - `POST /api/admin/v1/agents` の `kind` 必須化は意図した破壊であり、未リリースのためベースラインを再凍結した
  - 手動確認は未実施 (残作業)

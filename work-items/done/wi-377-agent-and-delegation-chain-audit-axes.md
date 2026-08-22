---
status: completed
authors: [tn]
risk: medium
created_at: 2026-08-16
priority: p0
depends_on: [wi-50-token-exchange-delegation-actor-chain, wi-368-delegation-depth-policy-and-delegation-mode-claim]
change_kind: feature
initial_context:
  specification:
    - spec/contexts/audit/scenarios.md#REQ-AUDIT-003
    - spec/contexts/oauth2/scenarios.md#REQ-OAUTH2-049
  typespec:
    - IdMagic.Contract.AuditEventSearchAttribute
    - IdMagic.Contract.AuditEventQuery
    - IdMagic.Contract.TokenExchanged
    - IdMagic.Contract.FgaCheckEvaluated
    - IdMagic.Contract.DelegationMode
  source:
    - backend/audit/ports/audit_search_attribute.go
    - backend/audit/ports/audit_event_repository.go
    - backend/audit/usecases/audit_search_extractor.go
    - backend/audit/db_postgres/audit_events.go
    - backend/audit/db_memory/audit_event_store.go
    - backend/oauth2/domain/delegation_mode.go
    - backend/oauth2/token/usecases/exchange_token.go
    - backend/authorization/usecases/check_access.go
    - frontend/src/features/admin-audit-events/AdminAuditEventsPage.tsx
    - infra/schema/postgres.sql
  tests:
    - backend/audit/usecases
    - backend/audit/db_memory
    - backend/audit/db_postgres
    - frontend/src/features/admin-audit-events
  stop_before_reading:
    - backend/idgovernance
    - backend/provisioning
affected_spec:
  - { path: spec/contexts/audit/scenarios.md, requirement: REQ-AUDIT-003 }
  - { path: spec/contexts/audit/scenarios.md, requirement: REQ-AUDIT-005 }
  - { path: spec/contexts/audit/scenarios.md, requirement: REQ-AUDIT-006 }
  - { path: spec/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-049 }
---

# 監査の検索軸にエージェントと委譲チェーンを加え、代行操作を本人の操作と区別できるようにする

## Motivation

監査の検索軸は `backend/audit/ports/audit_search_attribute.go` の閉じた許可リストで決まる。現在の軸は `event.type` / `outcome` / `actor.id` / `actor.username` / `target.id` / `client.id` / `client.ip` / `session.id` / `transaction.id` / `correlation.id` / `request.id` / `workflow.id` / `workflow_run.id` / `workflow_step.id` である。**エージェントを指す軸も、主体の種別を表す軸も、委譲チェーンを表す軸も無い。**

さらに `backend/audit/usecases/audit_search_extractor.go` は `payload.actorUserId` を `payload.userId` へフォールバックする。結果として、エージェントがユーザーの代わりに行った操作は、そのユーザー本人が行った操作と**検索上まったく区別できない**。許可リストの外はフィルタ解析の時点で弾かれ SQL に届かないため、これは運用の工夫で回避できる制限ではない。

委譲の情報がまったく無いわけではない。`TokenExchanged` は `actorUserId` / `subjectUserId` / `delegationDepth` / `delegationMode` を持つ。しかしそれはイベントのペイロード内部にあるだけで検索軸ではなく、`FgaCheckEvaluated` に至っては `actorChainDepth` という整数しか持たない。「エージェント X がユーザー Y の代わりに何をしたか」という、エージェントを第一級の主体として扱う製品が最初に答えるべき問いに、現在の監査は答えられない。

これはガードレールより先に来る。上限を設けて拒否したところで、その判断が誰に対するものだったかを後から辿れなければ、ガードレールは検証できない主張にしかならない。`draft-klrc-aiagent-auth` が監査に必須と挙げる項目 — 認証されたエージェント識別子、委譲された主体、対象リソース、要求された操作、認可判断、時刻、判断に影響したアテステーションまたはリスク状態 — の前半は、まさにここで欠けているものである。

本 work item は [[wi-59-agent-governance-guardrails-audit-inventory]] の監査部分を分離したものである。wi-59 は決定の記録・ガードレール評価器・監査・インベントリ・UI を一つに束ねた大物で、依存が解消された今も着手しにくい。監査軸だけを先に出せば、ガードレールは「判断が追える状態」の上に載せられる。

## Scope

- `spec/contexts/audit/scenarios.md` に、エージェントを主体とする監査イベントの検索と、委譲チェーンによる相関を求める規範シナリオを追加する。既存の REQ-AUDIT-003 (`workflow_id` / `run_id` による検索) と同じ形の軸拡張とする。
- 追加する検索軸を確定する。少なくとも次を含む。
  - `actor.type` — 主体が `user` か `agent` かを区別する。既存の `actor.id` の意味を変えずに種別を分ける。
  - `agent.id` — エージェント自身の識別子。
  - 委譲チェーンの相関軸 — `act` チェーンに現れる主体を横断検索できるようにする。チェーン全体を一つの軸で表すか、チェーン ID と参加者の 2 軸に分けるかは `## Design` で決める。
  - `delegation.depth` と `delegation.mode` — 後者は REQ-OAUTH2-049 が定める `direct` / `autonomous` / `on_behalf_of` の導出規則をそのまま使い、監査側で二つ目の導出を作らない。
- `ExtractSearchAttributes` を、エージェントが主体のイベントで `actor.id` をユーザーへフォールバックしないよう改める。既存イベントの後方互換の扱いを `## Design` で決める。
- 委譲チェーンをペイロードへ載せる範囲を確定する。現状は深さの整数だけの箇所があり、参加者を辿れない。
- 管理コンソールの監査検索に、追加した軸での絞り込みと、委譲チェーンを展開して表示するビューを加える。
- `spec/contexts/identity-management/models.tsp` に `model Agent` 集約を追加する ([[wi-369-agent-capability-survey-2026-08]] の再評価条件による。wi-376 が先に着手していれば重複させない)。

## Out of Scope

- ガードレールの評価器と上限の強制。[[wi-59-agent-governance-guardrails-audit-inventory]] が持つ。
- エージェントのインベントリ画面と統制ダッシュボード。同上。
- 外部への監査ログ配信。[[wi-286-outbound-event-hooks-and-audit-log-streaming]] と [[wi-287-tamper-evident-audit-log-integrity]] が持つ。
- 監査イベントの保持期間と削除。現行の追記専用 7 年保持を変えない。
- リスクスコアやアテステーション状態の記録。`draft-klrc-aiagent-auth` が挙げる監査項目のうち、この 2 つは記録すべき値がまだ存在しない ([[wi-150-risk-based-authentication-and-adaptive-sign-in]] と [[wi-151-managed-device-inventory-and-posture-access-conditions]] が先)。軸の設計が将来の追加を妨げないことだけ確認する。

## Design

### 追加する検索軸

| 軸 | 値の出どころ | 演算子 | 多値 |
|---|---|---|---|
| `actor.type` | `agent` (Agent が行為者として振る舞う経路のイベント) / `user` (それ以外で行為者が定まる) | `eq` `in` | 単値 |
| `agent.id` | `payload.agentId` | `eq` `in` | 単値 |
| `delegation.actor` | `act` チェーンの参加者 (現在の行為者、入れ子の `act.sub`、subject) | `eq` `in` | 多値 |
| `delegation.depth` | `payload.delegationDepth` または `payload.actorChainDepth` | `eq` `in` | 単値 |
| `delegation.mode` | `payload.delegationMode` | `eq` `in` | 単値 |

`delegation.mode` は REQ-OAUTH2-049 が定める `DeriveDelegationMode` の出力をペイロードから読むだけで、監査側で導出し直さない。二つ目の導出を置けば、イントロスペクションの応答と監査の記録が食い違いうる。

### 1. 委譲チェーンの表現 — 案 A (参加者の多値軸) を採る

`delegation.actor` を多値の軸とし、1 つのイベントの `act` チェーンに現れる主体をそれぞれ 1 つの値として sidecar に書く。`delegation.actor=X` は、X がチェーンのどの段にいても引ける。

案 B (チェーンに ID を振る) は採らない。ID は新しい永続状態であり、最初の交換で採番したうえで以降のトークン、リフレッシュ、イントロスペクション、認可判定のすべてへ伝播させる必要がある。`DeriveDelegationMode` は「`act` から導出することで第二の真実を作らない」という判断で置かれたものであり、チェーン ID はまさにその第二の真実を持ち込む。参加者は既に `act` に載っているので、案 A は状態を増やさずに同じ相関を与える。

代償は sidecar の主キーである。`audit_event_search_attributes` は主キーが `(event_id, attr_name)` で、1 イベント 1 属性あたり 1 値しか持てない。主キーを `(event_id, attr_name, attr_value)` へ広げて属性を多値にする。検索側の `EXISTS` 述語は「いずれかの行が一致すれば真」の意味を既に持つため、単値の軸の意味論は変わらない。sidecar はペイロードから再生成できる派生データであり、監査の正本である `audit_events` は触らない。

### 2. 既存イベントの後方互換 — 読み出し時に補わず、投影も作り直さない

`actor.id` のフォールバックを止める範囲は、Agent を行為者とするイベントに限る (どのイベントがそれにあたるかは 4 節で決める)。そこでは `actor.id` を Agent 側の識別子とし、`userId` へのフォールバックを行わない。それ以外のイベントのフォールバックは現状のまま残す。したがって既存の検索は結果を変えず、変わるのはこれまで種別を持たなかったイベントの見え方だけである。

過去に記録済みのイベントは新しい軸の行を持たない。これを読み出し時に `unknown` で埋めると、「軸が無い」ことと「種別が unknown である」ことが区別できなくなり、`unknown` で引いた結果に古いイベントが混ざる。軸を持たないイベントは軸による絞り込みに一致しない、という `EXISTS` の既存の意味論をそのまま使い、追加以降のイベントにのみ軸が付くことを UI 側の注記で明示する。投影の作り直しは追記専用の記録への書き戻しであり、7 年保持の監査に対して後から属性を足す操作は、記録の不変性の主張と両立しない。

### 3. PII の扱い — チェーン参加者は生の識別子のまま保存する

チェーンの参加者は `user_id` / `client_id` / `agent_id` の不透明な識別子であり、`actor.id` と `target.id` が既に生で保存しているものと同じ種類である。`ResolvableUserEventPayloadPolicy` が構造的に禁じているのはユーザー名であって識別子ではない。`FgaCheckEvaluated` がリソース ID を要約へ落とすのは、リソース ID が業務上の値を埋め込みうるからであり、主体の識別子はその対象にあたらない。よって `delegation.actor` は `TransformNone` とし、チェーンにユーザー名相当の値を載せないことをイベントモデル側で担保する。

### 4. `agentId` は行為者を指すとは限らない

`agentId` を持つイベントの多くでは、Agent は行為者ではなく操作の**対象**である。`AgentRegistered` や `AgentDisabled` は管理者が Agent に対して行う操作であり、利用者が承認を与える `BackchannelAuthApproved` も同様である。`agentId` の有無から種別を推測すると、これらが Agent 自身の操作として記録される。

そこで `actor.type` は、Agent が自ら振る舞う経路のイベント型を列挙して決める (`TokenExchanged` / `WorkloadTokenExchanged` / `AgentApprovalRequired` / `BackchannelAuthRequested`)。`outcome` の分類が既に同じ形をとっており、推測ではなく列挙で決める点も同じである。列挙に無いイベントの `agentId` は対象を指すものとして `agent.id` 軸にだけ載せる。`agent.id` は「この Agent に関するイベントか」、`actor.type` は「その Agent が行為者だったか」という別の問いに答える。両者を併せて指定すれば、Agent 自身の操作だけが引ける。

`actor.id` のフォールバックは、Agent が行為者のイベントでのみ利用者ではなく Agent の識別子へ向ける。同時に、代行された利用者は `target.id` に残す。行為者と対象の区別 (`AuditActorVsTarget`) をそのまま委譲へ延長したものである。

### 5. チェーンの参加者は資格情報の水準の識別子である

`delegation.actor` に並ぶのは `act` チェーンの `sub` と subject、すなわちトークンの主体である。Agent の識別子は載せない。交換の時点で解決できるのは現在の行為者に束縛された Agent だけで、チェーンの前の段の Agent は `act` から復元できない。片側だけを載せれば、チェーンのどこに Agent がいたのかを誤って読ませることになる。Agent の水準の問いには `agent.id` と `actor.type` が答える。

### ペイロードへ載せる範囲

- `TokenExchanged` に `agentId` と `actorChain` を足す。深さと両端 (`actorUserId` / `subjectUserId`) だけでは中間の参加者を辿れない。
- `FgaCheckEvaluated` と `FgaResourcesEnumerated` は据え置く。Motivation は `FgaCheckEvaluated` が深さの整数しか持たない点を欠落として挙げているが、Authorization には「タプルの内容と主体識別子は監査へ複製しない」という既存の判断があり (`spec/contexts/authorization/internals.md` の Audit 節)、リソース識別子をダイジェストへ落とすのも同じ判断に基づく。参加者を載せることはこの判断の撤回にあたり、監査軸の追加の副産物として行ってよい変更ではない。`FgaCheckEvaluated` は識別子を含まない `actorChainDepth` を `delegation.depth` へ供給するにとどめ、チェーンの相関はそれを成立させた OAuth2 の `TokenExchanged` 側で取る。判断そのものを見直す必要があれば別の work item で扱う。

### 索引

`audit_event_search_attributes_lookup_idx` は `(tenant_id, attr_name, attr_value, occurred_at DESC)` なので、追加する軸にも既存の索引がそのまま当たる。軸ごとの索引は足さない。主キーの拡張は `attr_value` を鍵の末尾に足すだけで、既存の鍵順序を変えない。

### `model Agent` について

Scope に挙げた `spec/contexts/identity-management/models.tsp` への `model Agent` の追加は、[[wi-376-supervised-agent-mandatory-approval]] が先に完了させている。重複させない。

## Plan

- 仕様の軸定義を先に確定し、`AuditSearchRegistry` はその写しとして実装する。許可リストが二つの正本を持たないようにする。
- `delegation.mode` は REQ-OAUTH2-049 が定める共有の導出関数を呼ぶ。監査側で独自に導出しない。
- 抽出器の変更は、エージェント主体のイベントを含む固定データに対する RED から始める。

## Tasks

- [x] T001 [Design] 委譲チェーンの表現、既存イベントの後方互換、PII の扱いを確定し `## Design` に記録する。
- [x] T002 [Spec] 監査の検索軸とエージェント相関の規範シナリオを追加し、必要なイベントのペイロードに委譲チェーンを載せて再生成する。
  - REQ-AUDIT-005 / REQ-AUDIT-006 を `spec/contexts/audit/scenarios.md` に追加。`AuditEventSearchAttribute.multi_valued` と `AuditEventSearchOptionsResponse.actor_types` / `delegation_modes` を追加。`TokenExchanged` に `agentId` / `actorChain` を追加。`AuditDelegationChain` / `AuditActorType` を用語集へ、多値の検索属性を `internals.md` へ。
- [x] T003 [App] `AuditSearchRegistry` と `ExtractSearchAttributes` を更新し、エージェント主体のフォールバックを止める。
  - RED: `TestActorChainParticipants` (`backend/oauth2/domain`, REQ-AUDIT-006) → `domain.ActorChainParticipants` を実装。
  - RED: `TestExtractSearchAttributesAgentActorDoesNotFallBackToTheUser` / `TestExtractSearchAttributesDelegationChainIsMultiValued` / `TestExtractSearchAttributesLeavesDelegationAxesAbsent` (`backend/audit/usecases`, REQ-AUDIT-005 / REQ-AUDIT-006) → 抽出器を多値化し、委譲の軸を追加。
  - `TestExtractSearchAttributesAgentAsTargetKeepsTheHumanActor` / `TestExtractSearchAttributesAgentActorWithoutAnActorSubUsesTheAgent` (REQ-AUDIT-005) で、`agentId` が対象を指す場合との区別を固定。
  - `TestAuditSearchRegistryHasDelegationAxes` (`backend/audit/ports`) で軸の宣言と多値の境界を固定。
- [x] T004 [UI] 管理コンソールの監査検索に新しい軸の絞り込みと委譲チェーンの展開表示を追加する。
  - `AdminAuditEventsPage` に 5 つの軸の行、`actor.type` と `delegation.mode` の選択式、委譲チェーンの展開表示 (参加者から `delegation.actor` で引き直す) を追加。過去のイベントには軸が付かないことを注記で示す。
- [x] T005 [Verify] エージェント代行の操作が本人の操作と区別して引けること、チェーン参加者から横断検索できること、許可リスト外が従来どおり弾かれること、PII が漏れないことを検証する。
  - `TestAuditEventRepositoryDelegationAxes` (`backend/audit/db_postgres`, REQ-AUDIT-005 / REQ-AUDIT-006) が実 DB で、エージェントの操作と本人の操作の分離、チェーン中間の参加者からの横断検索、チェーン外の主体が一致しないこと、多値行の冪等を確認。
  - `TestAuditEventStoreMatchesAnyValueOfAMultiValuedAttribute` (`backend/audit/db_memory`) が memory store と PostgreSQL の意味論の一致を固定。
  - 許可リスト外の拒否は既存の `TestParseAuditFilterRejects` が引き続き担保。PII は `TestAuditSearchRegistryHasDelegationAxes` で全軸が生の識別子であること、チェーンにユーザー名相当の値が入らないことをイベントモデル側で担保。

## Verification

- `mise run verify-spec`
- `mise run test-go`
  - reason: 検索軸の許可リスト、抽出器のフォールバック除去、委譲チェーンによる相関、テナント境界、PII の非露出。
- `mise run verify-ui`
- `mise run verify`
- 手動: エージェントにユーザーを代行させて操作し、`actor.type=agent` と `agent.id` で引けること、同じ操作がユーザー本人の検索結果に混ざらないこと、チェーンを展開して代行の連なりが読めることを確認する。

## Risk Notes

リスクは medium。`actor.id` のフォールバックを止める変更は、既存の監査検索の結果を変える。運用者が慣れた検索が黙って別の結果を返すのは、機能が無いことより危険なので、後方互換の扱いを `## Design` で明示し、過去イベントの見え方を検証項目に含める。

軸を増やすことによる SQL の劣化にも注意する。許可リスト方式は任意の列での絞り込みを許さない設計なので、追加する軸それぞれに索引の要否を判断する。

## Completion

- **Completed At**: 2026-08-22
- **Summary**:
  監査の検索軸に `actor.type` / `agent.id` / `delegation.actor` / `delegation.depth` / `delegation.mode` の 5 つを加え、エージェントが利用者を代行した操作を、利用者本人の操作と区別して引けるようにした。規範シナリオは REQ-AUDIT-005 (代行の区別) と REQ-AUDIT-006 (チェーン参加者からの横断検索) の 2 つを追加した。

  意味の差は 3 つある。第一に、検索属性の副表が多値になった。主キーを `(event_id, attr_name)` から `(event_id, attr_name, attr_value)` へ広げ、`delegation.actor` が 1 イベントにつきチェーンの参加者ぶんの値を持てるようにした。既存の `EXISTS` 照合は「いずれかの行が一致すれば真」の意味を既に持つため、単値の軸の意味論は変わらない。第二に、`TokenExchanged` が委譲チェーンの参加者 (`actorChain`) と交換を行った Agent (`agentId`) を残すようになった。これまでは深さの整数と両端しか無く、中間の行為者を後から辿れなかった。第三に、`actor.id` のフォールバックが Agent を行為者とするイベントで止まり、代行された利用者は `target.id` に移った。行為者と対象の区別を委譲へ延長したものである。

  `agentId` の解釈には注意を要する判断があった。`AgentRegistered` や `BackchannelAuthApproved` のように、`agentId` が行為者ではなく操作の対象を指すイベントが多数あるため、種別を `agentId` の有無から推測すると管理者の操作が Agent の操作として記録される。Agent が自ら振る舞う経路のイベント型を列挙して決める形にした。

  Motivation が挙げていた `FgaCheckEvaluated` への参加者の追加は行っていない。Authorization には「タプルの内容と主体識別子は監査へ複製しない」という既存の判断があり、参加者を載せることはその撤回にあたる。識別子を含まない `actorChainDepth` を `delegation.depth` へ供給するにとどめた。詳細は `## Design` に記した。

  仕様が得たものは規範シナリオ 2 件 (REQ-AUDIT-005 / REQ-AUDIT-006)、用語 2 件 (`AuditDelegationChain` / `AuditActorType`)、TypeSpec のフィールド 5 件 (`AuditEventSearchAttribute.multi_valued`、`AuditEventSearchOptionsResponse.actor_types` / `delegation_modes`、`TokenExchanged.agentId` / `actorChain`)、`internals.md` の節 1 件である。失ったものはない。

- **Verification Results**:
  - `mise run verify` - passed
  - `mise run test-go` - passed (`backend/audit/...` と `backend/oauth2/...` を含む全パッケージ)
  - `mise run check-schema` - passed (副表の主キー変更が収束する)
  - `mise run check-api-compat` - passed (破壊的変更なし)
  - `mise run test-ui-unit` - passed (657 件)
  - `mise run spec-diff` - `added scenarios: REQ-AUDIT-005, REQ-AUDIT-006`
  - 手動確認は未実施。実環境でエージェントに利用者を代行させる経路を用意していないため、同じ確認を `TestAuditEventRepositoryDelegationAxes` が実 PostgreSQL に対して行っている。

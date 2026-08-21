---
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-16
priority: p0
depends_on: [wi-50-token-exchange-delegation-actor-chain, wi-368-delegation-depth-policy-and-delegation-mode-claim]
change_kind: feature
affected_spec:
  - { path: spec/contexts/audit/scenarios.md, requirement: REQ-AUDIT-003 }
  - { path: spec/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-049 }
---

# 監査の検索軸にエージェントと委譲チェーンを加え、代行操作を本人の操作と区別できるようにする

## Motivation

監査の検索軸は `backend/audit/ports/audit_search_attribute.go` の閉じた許可リストで決まる。現在の軸は `event.type` / `outcome` / `actor.id` / `actor.username` / `target.id` / `client.id` / `client.ip` / `session.id` / `transaction.id` / `correlation.id` / `request.id` / `workflow.id` / `workflow_run.id` / `workflow_step.id` である。**エージェントを指す軸も、主体の種別を表す軸も、委譲チェーンを表す軸も無い。**

さらに `backend/audit/usecases/audit_search_extractor.go` は `payload.actorUserId` を `payload.userId` へフォールバックする。結果として、エージェントがユーザーの代わりに行った操作は、そのユーザー本人が行った操作と**検索上まったく区別できない**。許可リストの外はフィルタ解析の時点で弾かれ SQL に届かないため、これは運用の工夫で回避できる制限ではない。

委譲の情報がまったく無いわけではない。`TokenExchanged` は `actorUserId` / `subjectUserId` / `delegationDepth` / `delegationMode` を持つ。しかしそれはイベントのペイロード内部にあるだけで検索軸ではなく、`FgaCheckEvaluated` に至っては `actorChainDepth` という整数しか持たない。「エージェント X がユーザー Y の代わりに何をしたか」という、エージェントを第一級の主体として扱う製品が最初に答えるべき問いに、現在の監査は答えられない。

これはガードレールより先に来る。上限を設けて拒否したところで、その判断が誰に対するものだったかを後から辿れなければ、ガードレールは検証できない主張にしかならない。`draft-klrc-aiagent-auth` が監査に必須と挙げる項目 — 認証されたエージェント識別子、委譲された主体、対象リソース、要求された操作、認可判断、時刻、判断に影響したアテステーションまたはリスク状態 — の前半は、まさにここで欠けているものである。

本 work item は [[wi-59-agent-governance-guardrails-audit-inventory]] の監査部分を分離したものである。wi-59 は ADR・ガードレール評価器・監査・インベントリ・UI を一つに束ねた大物で、依存が解消された今も着手しにくい。監査軸だけを先に出せば、ガードレールは「判断が追える状態」の上に載せられる。

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

未定。着手時に次の 3 点を確定して本節に記録する。

1. **委譲チェーンの表現。** 案 A: チェーンの参加者を多値の軸として持ち、いずれかに一致すれば引ける。案 B: チェーンに ID を振り、ID による相関と参加者の一覧を分ける。案 B は相関が正確だが、ID の生成と伝播を全経路へ通す必要がある。
2. **既存イベントの後方互換。** `actor.id` のフォールバックを止めると、過去に記録済みのイベントは種別を持たないまま残る。読み出し時に `unknown` として扱うか、投影を作り直すかを決める。
3. **PII の扱い。** 監査は `ResolvableUserEventPayloadPolicy` でユーザー名の混入を構造的に禁じ、`FgaCheckEvaluated` はリソース ID を 16 桁の要約へ落としている。委譲チェーンの参加者にも同じ規則を適用するかを決める。

## Plan

- 仕様の軸定義を先に確定し、`AuditSearchRegistry` はその写しとして実装する。許可リストが二つの正本を持たないようにする。
- `delegation.mode` は REQ-OAUTH2-049 が定める共有の導出関数を呼ぶ。監査側で独自に導出しない。
- 抽出器の変更は、エージェント主体のイベントを含む固定データに対する RED から始める。

## Tasks

- [ ] T001 [Design] 委譲チェーンの表現、既存イベントの後方互換、PII の扱いを確定し `## Design` に記録する。
- [ ] T002 [Spec] 監査の検索軸とエージェント相関の規範シナリオを追加し、必要なイベントのペイロードに委譲チェーンを載せて再生成する。
- [ ] T003 [App] `AuditSearchRegistry` と `ExtractSearchAttributes` を更新し、エージェント主体のフォールバックを止める。
- [ ] T004 [UI] 管理コンソールの監査検索に新しい軸の絞り込みと委譲チェーンの展開表示を追加する。
- [ ] T005 [Verify] エージェント代行の操作が本人の操作と区別して引けること、チェーン参加者から横断検索できること、許可リスト外が従来どおり弾かれること、PII が漏れないことを検証する。

## Verification

- `just verify-spec`
- `just test-go`
  - reason: 検索軸の許可リスト、抽出器のフォールバック除去、委譲チェーンによる相関、テナント境界、PII の非露出。
- `just verify-ui`
- `just verify`
- 手動: エージェントにユーザーを代行させて操作し、`actor.type=agent` と `agent.id` で引けること、同じ操作がユーザー本人の検索結果に混ざらないこと、チェーンを展開して代行の連なりが読めることを確認する。

## Risk Notes

リスクは medium。`actor.id` のフォールバックを止める変更は、既存の監査検索の結果を変える。運用者が慣れた検索が黙って別の結果を返すのは、機能が無いことより危険なので、後方互換の扱いを `## Design` で明示し、過去イベントの見え方を検証項目に含める。

軸を増やすことによる SQL の劣化にも注意する。許可リスト方式は任意の列での絞り込みを許さない設計なので、追加する軸それぞれに索引の要否を判断する。

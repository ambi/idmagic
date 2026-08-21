---
status: pending
authors: [tn]
risk: medium
created_at: 2026-08-16
priority: p0
depends_on: [wi-52-ciba-async-human-approval]
change_kind: feature
affected_spec:
  - { path: spec/contexts/oauth2/scenarios.md, requirement: REQ-OAUTH2-041 }
  - { path: spec/contexts/identity-management/models.tsp, symbol: AgentKind }
---

# `AgentKind.Supervised` のエージェントには人間の承認なしにトークンを発行しない

## Motivation

`AgentKind` は `Autonomous` と `Supervised` を区別し、仕様は後者を「Agent が人間の監督下で実行する区分」と定義している。しかし監督を実行時に強制する経路は一つも存在しない。`Supervised` は Go 全体で `backend/idmanagement/domain/enums.go` の宣言と一件のテストにしか現れず、`backend/oauth2` のトークン発行経路は一度も参照しない。`Autonomous` として登録しても `Supervised` として登録しても、得られるトークンは完全に同一である。

つまり `Supervised` は現在ただのラベルである。管理者が監督下だと宣言したエージェントが、誰の承認も経ずに自律実行する。宣言と挙動が食い違っている以上、この区分を根拠にした運用判断はすべて誤りうる。エージェントの区分を監査画面で見て安心する運用者を作ってしまう点で、区分が無い状態より悪い。

この穴は [[wi-52-ciba-async-human-approval]] の完了 (2026-08-14) によって埋められる状態になった。CIBA の `ApprovalRequest` は既に `agent_id` を持ち、承認が成立するまでトークンを発行しない (REQ-OAUTH2-041)。必要なのは新しい承認機構ではなく、`AgentKind` と既存の承認機構を結ぶ一本の線である。

OAuth2 の仕様は「どのエージェントが承認を要するか」の判断をガバナンス層へ明示的に委ねている。ただし [[wi-59-agent-governance-guardrails-audit-inventory]] が扱うのは閾値超過時に承認へ降格するという条件付きの規則であり、本 work item が扱うのは `Supervised` なら常に承認を要するという無条件の規則である。後者は前者を待つ必要がなく、前者よりはるかに安い。

## Scope

- `spec/contexts/oauth2/scenarios.md` に、`Supervised` な Agent へのトークン発行が人間の承認を経ることを求める規範シナリオを追加する。対象は Agent が主体となるすべての発行経路 — `client_credentials`、トークン交換、ワークロード ID 連携による交換 — とする。
- 承認が成立していない場合の応答を確定する。CIBA の承認要求を暗黙に生成するのか、承認要求の提示を要求して拒否するのかを `## Design` で決め、フェイルクローズを守る。
- `spec/contexts/identity-management/models.tsp` に `model Agent` 集約を追加する。現在この集約はリクエスト / レスポンス / イベントとしてしか仕様に存在せず、`AgentKind` が実行時の意味を持つ本変更が [[wi-369-agent-capability-survey-2026-08]] の設定した再評価条件 (「エージェント集約に仕様変更が入る時に併せて解消する」) を発火させる。
- `spec/contexts/oauth2/models.tsp` の `AccessTokenClaims` に `agent_id` クレームを宣言する。Go (`backend/shared/security/tokens_jose/jwt_signer.go`) では既に発行しているが仕様に無く、本変更が参照するため同時に解消する。
- Go 側でトークン発行境界に `AgentKind` の判定を追加する。`Agent` の状態ゲート (`Active` / `Disabled` / `Killed`) と同じ位置で、同じフェイルクローズの規則に従う。
- 拒否と承認をイベントとして残し、監査から `AgentKind` を根拠にした判断が追えるようにする。

## Out of Scope

- 閾値・予算・レート超過を根拠にした条件付きの承認要求。[[wi-59-agent-governance-guardrails-audit-inventory]] が持つ。
- 承認者の決定規則の高度化 (承認者の委任、複数承認、エスカレーション)。現行どおり対象ユーザー本人のステップアップ認証済みセッションに限る (REQ-OAUTH2-043)。
- CIBA の `ping` / `push` モード。wi-52 が poll のみに限定した判断を維持する。
- **AAuth / Agent Authorization Grant** (`draft-…-aauth`、リダイレクトを使えない経路でエージェントがユーザー代行トークンを得る OAuth 2.1 拡張) の実装。解こうとしている問題は CIBA が既に解いており、個人 draft の段階で二つ目の非リダイレクト承認経路を持つと、承認記録が二重化して監査が分岐する。IETF OAuth WG に採択され、かつ CIBA と異なる要求が現れた時点で再評価する。

## Design

未定。着手時に次の 2 点を確定して本節に記録する。

1. **承認が無い状態の応答。** 案 A: 認可サーバーが CIBA の `ApprovalRequest` を暗黙に生成し、クライアントへ `authorization_pending` 相当を返して待たせる。案 B: `auth_req_id` の提示を要求し、無ければ拒否する。案 A はクライアント実装が楽だが、承認要求が意図せず大量生成されうる。案 B は明示的だが、既存の `client_credentials` クライアントを壊す。
2. **既存の `Supervised` エージェントへの適用方法。** 本変更は挙動を破壊的に変える。テナント単位の移行猶予を設けるか、`Supervised` の登録自体を新しい意味に切り替えるかを決める。

## Plan

- 仕様を先に変え、規範シナリオと `model Agent` を含めて再生成する。
- トークン発行境界の判定は、既存の Agent 状態ゲートと同じ関数の近傍に置く。エージェント固有の関心事を複数箇所へ散らさない。
- 拒否経路のテストを先に書き、承認済み経路が通ることを後から確認する。

## Tasks

- [ ] T001 [Design] 承認が無い場合の応答と既存エージェントへの適用方法を確定し、`## Design` に記録する。
- [ ] T002 [Spec] `Supervised` の承認必須シナリオ、`model Agent` 集約、`AccessTokenClaims.agent_id` を追加して再生成する。
- [ ] T003 [App] トークン発行境界に `AgentKind` 判定をフェイルクローズで実装し、拒否と承認をイベントへ出す。
- [ ] T004 [Verify] 全発行経路 (`client_credentials` / トークン交換 / ワークロード ID 交換) で `Supervised` が承認なしに発行されないこと、`Autonomous` が影響を受けないこと、判定不能時に拒否側へ倒れることを検証する。

## Verification

- `just verify-spec`
- `just test-go`
  - reason: 3 つの発行経路それぞれで `Supervised` の拒否と承認後の発行、`Autonomous` の非退行、判定不能時のフェイルクローズを確認する。
- `just verify`
- 手動: `Supervised` のエージェントを登録し、承認なしにトークンが得られないこと、アカウントポータルで承認するとトークンが得られること、監査に区分を根拠とした判断が残ることを確認する。

## Risk Notes

リスクは medium。既存の `Supervised` エージェントの挙動を破壊的に変えるため、移行方法を `## Design` で明示しないまま実装すると、稼働中のエージェントが一斉に停止しうる。一方で判定を緩めに入れると本 work item の目的そのものが失われるので、緩和は移行猶予で行い、判定自体は常にフェイルクローズとする。

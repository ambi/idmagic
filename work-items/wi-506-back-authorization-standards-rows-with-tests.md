---
depends_on: []
status: pending
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-09-06
priority: p2
change_kind: maintenance
spec_impact: { kind: none, reason: "宣言済みの標準行に、その id を名指しするテストを対応付ける作業である。standards.md の行そのものも製品の振る舞いも変えない。テストが書けない行が見つかった場合、それは製品が宣言した採用を満たしていないということなので、欠陥として個別の work item に切り出す。" }
---

# Authorization が宣言する標準 5 行にテストを対応付け、標準の被覆台帳から外す

## Motivation

[[wi-495-burn-down-the-standards-coverage-debt]] は標準の被覆台帳へ受入集合を入れ、消化の単位を所有文書と決めた。本項目はそのうち `docs/contexts/authorization/standards.md` の 5 行を引き取る。**この文書は 5 行すべてが名指しを持たない。** WsFederation と並んで、台帳の中で全滅している 2 文書のうちの 1 つである。

5 行が扱うのは認可の判定そのものである。`AUTHZEN-FGA-FAIL-CLOSED` は、判定できないときに拒否する側へ倒れることを宣言している。この行が守られていなければ、他の 4 行が正しくても、判定器が答えられない場面で通ってしまう。全滅している 5 行の中で最も先に観測すべき行である。

## Scope

- 次の 5 行を消化する。

| ID | Adoption |
|---|---|
| `AUTHZEN-FGA-EVALUATION` | required |
| `AUTHZEN-FGA-FAIL-CLOSED` | required |
| `AUTHZEN-FGA-ACTOR-CHAIN` | required |
| `RFC8693-FGA-ACTOR-AND` | required |
| `AUTHZEN-FGA-SEARCH` | optional |

- 1 行ごとに、その行の `Statement` を区別できる入力と観測を持つテストを対応付け、`// <ID>: <この行の何を固定しているか>` の注記を足す。
- 観測の形は行の `Adoption` に従う。型は [[wi-495-burn-down-the-standards-coverage-debt]] の Design が定める。
- 消化した id を `tools/check/standards-coverage-debt.json` から外す。

## Out of Scope

- 他の文書が宣言する標準行。文書ごとに別の work item が持つ。OAuth2 が持つ `RFC8693-*` の他の行（委譲、なりすまし、subject token）は [[wi-499-back-oauth2-standards-rows-with-tests]] が持つ。
- `standards.md` の行の追加、削除、`Adoption` および `Strength` の変更。
- 見つかった実装の欠陥の修正。欠陥として切り出した先で扱う。
- 受入集合の導入と、台帳ファイルそのものの削除。親の [[wi-495-burn-down-the-standards-coverage-debt]] が持つ。
- 具体例の被覆負債。[[wi-496-burn-down-the-example-coverage-debt]] が持つ。
- 認可モデルそのものの変更。[[wi-371-authorization-rebac-admin-ui]] が別の側面を持つ。
- id を名指しする文字列があるだけを根拠にした台帳からの削除。

## Design

`AUTHZEN-FGA-FAIL-CLOSED` の観測は、判定器が答えを返せない状況を実際に作ることである。到達できない外部依存、壊れた関係データ、時間切れなど、答えが出ない入力を与えて、結果が「拒否」になることを読む。**答えが出る入力で「拒否が返る」ことを観測しても、この行は区別できない。** それは `AUTHZEN-FGA-EVALUATION` が扱う普通の否認であり、フェイルクローズではない。この 2 行を同じテストで名指すと、片方が失われても気づけない。

`AUTHZEN-FGA-ACTOR-CHAIN` と `RFC8693-FGA-ACTOR-AND` は、どちらも代理の連鎖を扱う。前者は連鎖そのものの解釈、後者は連鎖に含まれる主体の権限が**積で効く**こと、つまり連鎖のどの主体も持っていない権限は結果として持てないことを宣言している。したがって `RFC8693-FGA-ACTOR-AND` の観測は、連鎖の 1 つの主体だけが持つ権限が、連鎖全体としては通らないことである。連鎖が解釈されることだけを観測しても、和で効く実装と区別できない。

`AUTHZEN-FGA-SEARCH` は `optional` である。提供しているならその振る舞いを観測する。提供していないなら行の `Adoption` が誤っているので規範の変更として切り出す。

## Plan

1. `AUTHZEN-FGA-EVALUATION` から着手し、判定の入口と `required` の観測の型を決める。
2. `AUTHZEN-FGA-FAIL-CLOSED` を、判定器が答えを返せない入力で消化する。普通の否認とは別のテストにする。
3. `AUTHZEN-FGA-ACTOR-CHAIN` を消化する。
4. `RFC8693-FGA-ACTOR-AND` を、連鎖の 1 主体だけが持つ権限が通らないことで消化する。
5. `AUTHZEN-FGA-SEARCH` の実装の有無を確かめ、観測するか切り出すかを決める。
6. 解決した id を台帳から外す。

## Tasks

- [ ] T001 [Acceptance] `AUTHZEN-FGA-EVALUATION` を消化し、判定の入口と `required` の型を決める。
- [ ] T002 [Acceptance] `AUTHZEN-FGA-FAIL-CLOSED` を、答えを返せない入力で消化する。
- [ ] T003 [Acceptance] `AUTHZEN-FGA-ACTOR-CHAIN` を消化する。
- [ ] T004 [Acceptance] `RFC8693-FGA-ACTOR-AND` を、連鎖の 1 主体だけの権限が通らないことで消化する。
- [ ] T005 [Inventory] `AUTHZEN-FGA-SEARCH` の実装の有無を確かめ、観測するか切り出すかを決める。
- [ ] T006 [Ledger] 解決した id を `standards-coverage-debt.json` から外す。
- [ ] T007 [Verify] `mise run verify`。

## Verification

- 本項目が持つ 5 件が `tools/check/standards-coverage-debt.json` から 1 件残らず消えている。
- 注記を足した各テストについて、対応する production の判断を崩すとそのテストが落ちる。
- `mise run check-spec`
- `mise run verify`

## Risk Notes

- **フェイルクローズを普通の否認で観測する。** 拒否が返ることは両方に共通なので、`AUTHZEN-FGA-FAIL-CLOSED` と `AUTHZEN-FGA-EVALUATION` を同じテストが名指しやすい。前者は判定器が答えを返せない入力でしか区別できない。別のテストにする。
- **積を和で観測する。** `RFC8693-FGA-ACTOR-AND` は連鎖が解釈されることではなく、連鎖のどの主体も持たない権限が通らないことである。連鎖の 1 主体だけが持つ権限を入力にする。
- **台帳の同時編集。** 自分が持つ id のエントリーだけを削除し、コメント配列と他の id には触れない。

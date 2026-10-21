---
status: in_progress
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-10-21
priority: p2
depends_on: []
change_kind: bugfix
affected_spec:
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-081, impact: conforms }
  - { path: docs/modules/identity-management/features/agent.md, requirement: REQ-IDMANAGEMENT-077, impact: conforms }
primary_use_cases:
  - id: agent-edit-keeps-owner-offboarding
    requirement: REQ-IDMANAGEMENT-081
    observable_result: 管理者が Agent を編集している間に所有者の停止で Agent が無効化されても、編集の後も Agent は Disabled のまま残り、編集した項目だけが変わる。
    boundary: acceptance
    test: { path: backend/idmanagement/internal/agent/usecases/concurrent_agent_writes_test.go, name: TestAgentWritesKeepAStateChangeMadeMeanwhile, task: test-go-race }
    fault_model: 管理者の編集が先に読んだ Agent を丸ごと保存し、その間の無効化を書き戻す。
  - id: agent-edit-keeps-kill
    requirement: REQ-IDMANAGEMENT-077
    observable_result: 管理者が Agent を編集している間に Agent が停止されると、編集は agent_killed で拒否され、Agent は Killed のまま残る。
    boundary: acceptance
    test: { path: backend/idmanagement/internal/agent/usecases/concurrent_agent_writes_test.go, name: TestAgentWritesKeepAStateChangeMadeMeanwhile, task: test-go-race }
    fault_model: 管理者の編集が先に読んだ Active の Agent を丸ごと保存し、その間の停止を取り消す。
---

# Agent を変える書き込みを、行を固定して読み直した最新の Agent に当てる

## 動機

REQ-IDMANAGEMENT-081 は、所有者の User が止まると、その User が所有する Agent を無効化すると定める。
`backend/idmanagement/internal/agent/usecases/owner_offboarding.go` は Agent を読み、状態を変えて丸ごと保存する。
管理者の Agent の編集（`admin_agents.go`）も Agent を読んで丸ごと保存する。

管理者が Agent を編集している間に所有者が停止すると、編集の保存が読んだ時点の `Active` を書き戻し、所有者の止まった Agent が有効のまま残り得る。
無効化は自動の書き手なので、管理者は書き戻しに気づけない。

## 対象範囲

- 所有者の停止に伴う無効化と、管理者の更新、無効化と再有効化、停止を、行を固定して読み直した最新の Agent に変更を当てる操作へ置き換える。メモリと PostgreSQL の実装と共有の契約テストを揃える。
- 修正後に、[同時の更新と排他](../../docs/design/data/database.md#同時の更新と排他)の適用状況と、IdManagement のデータとリスクの文書から、Agent の丸ごとの保存を除く。

## 対象外

- 管理者どうしの同じ項目の編集の競合（[wi-605](wi-605-detect-lost-updates-with-etag-and-if-match.md)）。
- 資格情報の束縛と解除。別の表に書くので Agent の行を書き戻さない。

## 設計

### 着手時の調査

起票時は所有者の停止に伴う無効化だけを想定していたが、管理者の操作も同じ形で書いている。

| 書き手 | 先に読んだ Agent を丸ごと保存したときに書き戻すもの |
| --- | --- |
| 所有者の停止に伴う無効化（`owner_offboarding.go`） | 同時の管理者の更新（名前、説明、ロールなど） |
| 管理者の更新（`UpdateAgent`） | 同時の無効化と停止。停止は一方向の終端なので、書き戻すと停止が取り消される |
| 管理者の無効化と再有効化（`SetAgentDisabled`） | 同時の停止と更新 |
| 管理者の停止（`KillAgent`） | 同時の更新 |

### 型と操作

wi-98381 の `UserRepository.Update` と同じ形にする。

```go
// package agentports
type AgentRepository interface {
    // ...既存の操作
    // Update は Agent の行を固定して読み直し、change をその最新の値に当てて保存し、保存した
    // Agent を返す。Agent がなければ change を呼ばずに nil を返す。
    Update(ctx context.Context, tenantID, id string, change AgentChange) (*agentdomain.Agent, error)
}

// AgentChange は Update が最新の Agent に当てる変更である。false を返すと保存しない。
type AgentChange func(agent *agentdomain.Agent) (bool, error)
```

`Killed` の拒否、変わった項目の判定、ロールの割り当ての検証は `change` の中で最新の Agent に対して行う。
名前の重複と所有者の存在の確認はほかの Agent と User を読むので、行を固定する前に行う。
条件付きの UPDATE 文を書く案は、状態の遷移と変わった項目の判定が SQL と Go に分かれるので採らない。

## 計画

1. 管理者の編集の読み取りと保存の間に所有者の停止を挟むテストで、故障を RED にする。
2. `AgentRepository.Update` を実装し、4 つの書き手を置き換えて GREEN にする。

## タスク

- [x] T001 [Acceptance] 管理者の更新が間の無効化と停止を書き戻す故障と、所有者の停止に伴う無効化が間の更新を書き戻す故障を RED にする。
- [x] T002 [App] `AgentRepository.Update` を実装し、4 つの書き手を置き換える。
- [x] T003 [Verify] 変更を検証する。

## 検証

- メモリと PostgreSQL の共有の契約テストで、`Update` が最新の Agent に変更を当てることを確かめる。PostgreSQL の検査はサンドボックスの外で実行する。
- `mise run test-go-changed`、`mise run lint-go`。

## リスク

無効化のイベントを更新した行から作るため、更新と発行の間の失敗の扱いを既存の無効化と揃える。

## 要約

規範仕様の差分はない。
起票時は所有者の停止に伴う無効化だけを想定していたが、管理者の更新、無効化と再有効化、停止も、先に読んだ Agent を丸ごと保存していた。
受け入れテストで、更新が同時の無効化を書き戻すこと、停止の後に届いた更新と再有効化が停止を取り消すこと、停止と所有者の停止に伴う無効化が同時の更新を書き戻すことを RED にしてから直した。

wi-98381 の `UserRepository.Update` と同じ形の `AgentRepository.Update` を設け、4 つの書き手を、行を固定して読み直した最新の Agent に変更を当てる形にした。
停止の拒否、変わった項目の判定、ロールの割り当ての検証は最新の Agent に対して行い、名前の重複と所有者の存在の確認は行を固定する前に行う。
所有者の停止に伴う無効化は、最新の Agent がまだその所有者の Active であるときだけ無効化してイベントを発行する。
条件付きの UPDATE 文を書く起票時の案は、状態の遷移と変わった項目の判定が SQL と Go に分かれるので採らなかった。

`Update` の意味はメモリと PostgreSQL の共有の契約テストで確かめた。
ミューテーションで見つかった、所有者の停止が複数の Agent と途中で停止された Agent を扱う経路の観測の不足は、テストを足して塞いだ。
データ設計の適用状況と IdManagement のデータとリスクの文書から、Agent の丸ごとの保存を除いた。
管理者どうしの同じ項目の更新の競合は wi-605 に残る。

---
status: completed
authors: [tn]
risk: low
reversibility: reversible
created_at: 2026-10-09
priority: p3
depends_on: []
change_kind: maintenance
evidence_policy: risk-based-v4
documentation_impact:
  level: none
  reason: "未公開であり、イベントはどの経路でも発行されていないので、既存データと実利用の契約に影響しない。"
  references: []
initial_context:
  specification:
    - docs/modules/provisioning/task/README.md
    - docs/modules/provisioning/README.md
  typespec: [IdMagic.Contract.GroupPushed]
  source:
    - backend/provisioning/internal/domain/task.go
    - backend/provisioning/internal/domain/events.go
    - backend/provisioning/internal/usecases/execute_task.go
    - tools/check/event-emission-debt.json
  tests:
    - backend/provisioning/internal/domain/task_test.go
    - backend/provisioning/internal/domain/events_test.go
  stop_before_reading: [frontend]
affected_spec:
  - { path: spec/modules/provisioning/models.tsp, symbol: IdMagic.Contract.GroupPushed }
---

# 発行されない GroupMembershipPushed のイベントと遷移を消す

## 動機

`tools/check/event-emission-debt.json` は、宣言はあるが本番のコードのどこも値を作らないドメインイベントを列挙する。
4 件のうち、`FederationLinked` と `FederationUnlinked` は wi-45298、`SigningKeyRetired` は wi-96851 が扱う。
残る `GroupMembershipPushed` を扱う work item はない。

Group のメンバーシップは、Group のプロビジョニングタスクの中で、Group のリソースを送った後に `members` への増分の `add` として送っている（`backend/provisioning/usecases/execute_task.go` の `pushGroupMembers`）。
そのタスクは成功すると `GroupPushed` を発行する。
メンバーシップだけを送るタスクは存在しないので、`GroupMembershipPushed` を発行する経路はない。

それでも、タスクの状態遷移の表（`docs/modules/provisioning/task/README.md`）と `backend/provisioning/domain/task.go` には `in_flight` から `GroupMembershipPushed` で `succeeded` へ進む遷移があり、TypeSpec もイベントを宣言している。
監査の記録を読む側は、起きないイベントに備えることになる。

## 対象範囲

- `GroupMembershipPushed` を、TypeSpec、ドメインイベントの型、タスクの状態遷移の表とコード、Provisioning の README のイベント一覧から消す。
- `tools/check/event-emission-debt.json` から `GroupMembershipPushed` を消す。

## 対象外

- メンバーシップの除去の配信。
  `RFC7643-OUT-GROUP-RESOURCES` が送らない理由を定めている。

## 設計

採らない案は、メンバーシップの送信を別のタスクに分け、その成功で `GroupMembershipPushed` を発行することである。
メンバーシップの送信は Group のリソースの送信で得た下流の識別子を使うので、別のタスクにすると、Group の送信の完了を待つ順序の制御が新たに要る。
いまの一つのタスクで送る形は、`GroupPushed` が Group とメンバーシップの両方の送信の完了を表しており、監査の記録の利用者がメンバーシップの送信だけを区別する要件もない。

## タスク

- [x] T001 [Contract] TypeSpec からイベントを消す。
- [x] T002 [Spec] タスクの状態遷移の表と README のイベント一覧から消す。
- [x] T003 [App] ドメインイベントの型と遷移を消す。
- [x] T004 [Tooling] `tools/check/event-emission-debt.json` から消す。
- [x] T005 [Verify] 変更を検証する。

## 検証

- `mise run check-event-contract`
- `mise run check-api-compat`
- `mise run verify`

## リスク

- イベントの宣言を消すと、OpenAPI の互換性の検査が後退として報告する。
  未リリースなので、基準を更新して受け入れる。

## 完了
- **完了日**: 2026-10-15
- **要約**:
  `mise run spec-diff` は、`ProvisioningTaskLifecycle` の状態遷移の変更、`spec/modules/provisioning/models.tsp:GroupMembershipPushed` の削除、`GroupPushed` の変更を示す。
  `GroupMembershipPushed` を、TypeSpec、Go のイベント型と遷移、タスクの状態遷移の表、Provisioning の README のイベント一覧、`tools/check/event-emission-debt.json` から消した。
  `GroupPushed` の説明に、メンバーシップの追加が下流に届いたことも含むと明記した。
  消した宣言は `affected_spec` から解決できないので、記録の参照先を、メンバーシップの送信の成功を表すようになった `GroupPushed` に改めた。
- **受け入れ RED の証拠**:
  - **テスト**: `mise run check-event-contract`
  - **要件**: N/A: 消したイベントを名指す規範要件はない。契約の変更は `IdMagic.Contract.GroupPushed` で追う。
  - **観測した失敗**: 負債台帳から消した状態で Go のイベント型を戻すと、`fail  backend: event GroupMembershipPushed is declared but no production code emits it` で失敗した。
  - **検出できる理由**: 発行経路のないイベントの宣言が残ると、この検査が必ず報告する。TypeSpec のモデルだけを戻した場合は検出しなかった。この検査は Go の宣言を読み、TypeSpec の宣言は読まないためである。
- **単体 RED の証拠**:
  - **テスト**: `backend/provisioning/internal/domain/task_test.go` の `TestTransitionProvisioningTaskLifecycle_RejectsMembershipOnlySuccess`
  - **要件**: N/A: 状態遷移の表の行を名指す規範要件はない。
  - **観測した失敗**: 実装前は `GroupMembershipPushed must not complete a task` で失敗した。
  - **検出できる理由**: 遷移の表に `in_flight` から `GroupMembershipPushed` で `succeeded` へ進む行が残ると、遷移が成功してこのテストが失敗する。
- **変更耐性の結果**:
  手で 2 つの故障を注入した。Go のイベント型を戻すと `check-event-contract` が検出した。TypeSpec のモデルだけを戻した場合は検出されなかったので、その限界を受け入れ RED の欄に記した。遷移の行は単体 RED が検出する。
- **検証結果**:
  - `mise run test-go-package -- ./backend/provisioning/...`: 302 件成功。
  - `mise run lint-go`、`mise run check-spec`、`mise run check-event-contract`、`mise run check-repository`: 成功。
  - 最新の main へ rebase した後に `mise run check` と `mise run verify` が成功した。

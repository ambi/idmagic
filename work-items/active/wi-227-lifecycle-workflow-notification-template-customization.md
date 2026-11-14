---
status: pending
authors: ["tn"]
risk: medium
created_at: 2026-07-16
priority: p3
group: lifecycle-workflow
change_kind: feature
affected_spec:
  - { path: docs/modules/identity-governance/features/workflow-run.md, requirement: REQ-IDGOVERNANCE-008 }
  - { path: docs/modules/tenancy/features/notification-template.md, requirement: REQ-TENANCY-016 }
depends_on: [wi-218-lifecycle-workflow-action-execution-and-audit, wi-6-real-email-sender-adapter]
---

# ライフサイクルワークフローの送信ごとに、テナントが別の文面を選べるようにする

## 動機

起票時は、ワークフローの `send_email` の文面をテナントが変えられなかった。
その後、[通知テンプレート](../../docs/modules/tenancy/features/notification-template.md)が、テナントによる上書き、プレビュー、試し送り、許可した差し込み変数の検査を持ち、ワークフローの通知も `lifecycle_workflow_notification` という一つのテンプレートとして上書きできるようになった（2026-11-14 に確認）。

残るのは、すべてのワークフローの送信が同じ一つのテンプレートを使うことである。
`send_email` の操作が持つ `template_key` は、本文に差し込むラベルにすぎず、文面を選ばない。
入社の歓迎と退職の案内のように、ワークフローごとに違う文面を送りたいテナントは、一つのテンプレートの中で書き分けられない。

通知の上書きの中心は実装済みで、残りは文面の選び方の拡張なので、優先度は p3 とする。

## 対象範囲

- ワークフローの通知に、テナントが名前を付けた複数の文面を持てるようにし、`send_email` の操作がそのどれかを選べるようにする。選ばない操作は、今の `lifecycle_workflow_notification` を使う。
- 差し込み変数は、今の通知テンプレートと同じく許可した集合に限り、保存の時点で検査する。
- 管理コンソールで、ワークフローの通知の文面の一覧、編集、プレビューを扱う。

## 対象外

- 任意の HTML のエディターとリッチテキスト。
- メール以外の通知の経路（SMS、プッシュ）。
- ワークフロー以外の通知の文面を複数にすること。通知のテンプレートのキーを固定の列挙とする判断を変えない。

## 設計

通知テンプレートは `template_key` を仕様で定める固定の列挙とし、テナントは追加できないと決めている。
ワークフローの文面を複数にするとき、この判断を変えずに、`lifecycle_workflow_notification` の中の名前付きの変種として持つか、IdGovernance が文面を持つかを着手時に決める。
前者は通知の検査と描画を再利用でき、後者はテンプレートのカタログを固定に保てる。

## 検証

- 選んだ文面で送られ、選ばない操作は既存のテンプレートで送られることを観測するテスト。
- 許可していない差し込み変数を含む文面の保存が拒否されるテスト。
- `mise run verify-ui`
- `mise run verify`

## リスク

文面を選ぶ操作が、削除した文面を指したまま残ると、送信の時点で失敗する。
文面の削除を、参照するワークフローがある間は拒否する。

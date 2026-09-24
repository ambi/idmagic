---
status: completed
authors: [tn]
risk: medium
reversibility: reversible
created_at: 2026-09-24
priority: p2
depends_on: []
change_kind: feature
evidence_policy: risk-based-v3
documentation_impact:
  level: removal_notice
  reason: 管理 API の操作 `ListProvisioningDeliveries`、`GetProvisioningDelivery`、`RetryProvisioningDelivery` とそのパスを取り除き、`ProvisioningTask` の名前の操作へ置き換える。契約は未配布だが、旧パスを呼ぶクライアントには移行が必要になる。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-68415-rename-provisioning-delivery-to-provisioning-task.md }
    - { kind: upgrade_note, path: docs/releases/upgrades/wi-68415-rename-provisioning-delivery-to-provisioning-task.md }
initial_context:
  specification:
    - docs/domain/provisioning/glossary.md
    - docs/domain/provisioning/scenarios.feature.md#REQ-PROVISIONING-015
    - docs/domain/scenarios.feature.md#REQ-PLATFORM-003
  typespec:
    - IdMagic.Provisioning.Operations.ListProvisioningTasks
    - IdMagic.Provisioning.Operations.GetProvisioningTask
    - IdMagic.Provisioning.Operations.RetryProvisioningTask
  source:
    - backend/provisioning
    - frontend/src/features/admin-applications
  tests:
    - backend/shared/http/server_http/provisioning_tenant_isolation_test.go
    - backend/provisioning/handlers_http/admin_task_list_pagination_test.go
  stop_before_reading: []
primary_use_cases:
  - id: read-provisioning-task-at-renamed-path
    requirement: REQ-PROVISIONING-015
    observable_result: 管理者は `GET /api/admin/v1/applications/{id}/provisioning/tasks/{task_id}` で自テナントのプロビジョニングタスクを読め、他テナントの管理者には存在しない id と同じ 404 が返る。
    unit_test: { path: backend/provisioning/handlers_http/admin_task_list_pagination_test.go, name: TestAdminTaskListRejectsCursorFromAnotherTenant, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/provisioning_tenant_isolation_test.go, name: TestForeignTenantAdminSeesProvisioningAsMissing, task: test-go-race }
    unit_fault_model: Provisioning のルート登録が旧パス `/provisioning/deliveries` のまま残り、新しいパスがハンドラーへ到達しない。
    e2e_fault_model: 本番の組み立てで旧パスだけが配線され、所有テナントの管理者が新しいパスでプロビジョニングタスクを読めない。
affected_spec:
  - { path: docs/domain/provisioning/scenarios.feature.md, requirement: REQ-PROVISIONING-015 }
  - { path: spec/contexts/provisioning/main.tsp, symbol: IdMagic.Provisioning.Operations.ListProvisioningTasks }
  - { path: spec/contexts/provisioning/main.tsp, symbol: IdMagic.Provisioning.Operations.GetProvisioningTask }
  - { path: spec/contexts/provisioning/main.tsp, symbol: IdMagic.Provisioning.Operations.RetryProvisioningTask }
---

# ProvisioningDelivery を ProvisioningTask へ改名し、日本語では「プロビジョニングタスク」と呼ぶ

## 動機

`ProvisioningDelivery` は、User や Group の変更をきっかけに作られる行である。
この行は、下流の SCIM サービスプロバイダーへ作成、更新、無効化、削除のいずれか 1 件を送るためのもので、`worker` が自動で実行する。
文書と仕様はこれを「配信」、管理画面の ja ロケールは「配送」と呼んでいる。
どちらも、何を届けるのか、誰が行うのかが読み取れない。
一つの概念に訳語が二つあることも、文書と画面の対応を分かりにくくしている。

英語の delivery も同じ問題を抱えている。
delivery は、GitHub の webhook deliveries のように、できあがったメッセージを届けることを指す語である。
この行が表すのはメッセージではなく、`operation` とステータスを伴う実行待ちの仕事である。

## 対象範囲

- 英語の名前を `ProvisioningDelivery` から `ProvisioningTask` へ改める。
  - TypeSpec のモデル、列挙、エラー、操作（`ListProvisioningTasks`、`GetProvisioningTask`、`RetryProvisioningTask`）
  - 管理 API のパス（`/api/admin/v1/applications/{id}/provisioning/tasks`、`/tasks/{task_id}`、`/tasks/{task_id}/retry`）と一覧の応答のフィールド `tasks`
  - ドメインイベント（`ProvisioningTaskStarted`）と、イベントのペイロードのフィールド `taskId`
  - ジョブの種類 `provisioning_task`
  - テーブル `provisioning_tasks`、`provisioning_scheduled_deprovisions.task_id` 列、制約と索引の名前
  - Go の型、関数、変数、ファイル名、テスト名。フロントエンドの型、API クライアント、コンポーネント
- 日本語の「配信」と「配送」を「プロビジョニングタスク」へ改める。行為を指す「配信する」は「プロビジョニングする」、または文脈に合う動詞（「作る」「実行する」「送る」）へ改める。
  - Provisioning の現行文書、`REQ-PLATFORM-003`、`docs/domain/structure.md`、`docs/design/` の API ガイドライン、データベース設計、脅威モデル
  - スキーマ、TypeSpec、Go のコメントとテストの説明文、`tools/check/example-coverage-debt.json`、`tools/render-docs` の説明文
  - 管理画面の ja と en の文言
  - 未完了の work item
- 生成物（sqlc、OpenAPI、実行時のルート情報、ルート優先度の参照文書）を作り直し、リリースのベースラインを凍結し直す。

## 対象外

- 別の意味の delivery、「配信」、「配送」。Shared Signals の `SecurityEventDelivery`、ログアウト通知の配送、メールの配送、Jobs の「配信不能」と「少なくとも 1 回の配送」、静的ファイルとアセットの配信、監査の「配信点」は変えない。これらは実際にメッセージを届けている。
- 完了済みの work item とリリース文書。当時の記録として残す。
- Go のコメントに残る、存在しない `spec/contexts/provisioning.yaml` の節名（`§配送・信頼性`）の参照。参照先の整理は改名とは別の問題である。
- work item のファイル名。wi-92540 のファイル名は他の記録から参照されているので変えない。

## 設計

### 呼び方の選択

| 候補 | 判断 |
| --- | --- |
| プロビジョニングタスク（`ProvisioningTask`） | 採用する。タスクは「これから片付ける仕事 1 件」を指し、`pending` から `succeeded` へ進むステータスと自然につながる。Jobs の `Job` が実行するという関係も「タスクをジョブが実行する」と素直に書ける。ドメインの語としての「タスク」はまだ使われていない |
| 配信、配送（`ProvisioningDelivery`） | 採らない。何を届けるのか、誰が行うのかが読み取れない。「ユーザーの作成を配送する」のように、届けるものが変更の指示であるときに日本語として不自然になる |
| 下流操作、プロビジョニング操作 | 採らない。管理者やユーザーが行う操作と読まれる |
| プロビジョニング処理 | 採らない。自動処理であることは伝わるが、1 件ずつ積まれて実行される単位であることが伝わりにくい |
| プロビジョニングイベント | 採らない。Microsoft Entra ID はこう呼ぶが、このリポジトリの「イベント」はドメインイベントを指し、`ProvisioningTaskStarted` などと紛れる |

英語の名前も改めるのは、訳語と英語の名前が食い違うと、文書を読んでからコードを探すたびに対応をつなぎ直す必要があるためである。
製品は未リリースなので、テーブルとジョブの種類の改名にデータ移行は要らない。
API のパスの改名は、未配布の契約を整える変更として、ベースラインを凍結し直して扱う。

### 書き換えの形

| 元の表現 | 改めた表現 |
| --- | --- |
| 配信、配信行、配送 | プロビジョニングタスク、プロビジョニングタスクの行 |
| 配信のステータス | プロビジョニングタスクのステータス |
| 下流へ配信される | 下流へプロビジョニングされる |
| 配信捕捉のポート | プロビジョニングタスクを捕捉するポート |
| 配送履歴（ja の画面） | プロビジョニングタスクの履歴 |
| Delivery history（en の画面） | Provisioning task history |
| delivery engine（Go のコメント） | provisioning engine |

## 計画

1. 受け入れ RED として、HTTP の境界のテスト 2 件のパスだけを `/provisioning/tasks` へ変え、404 で失敗することを確かめる。
2. 英語の識別子、ファイル名、パス、テーブル、ジョブの種類を改め、生成物を作り直す。
3. 日本語の文書、コメント、画面の文言を改める。
4. `rg '配信|配送|[Dd]eliver'` で残りを点検し、対象外の意味だけが残ることを確かめる。
5. リリースのベースラインを凍結し直し、リリースノートを書く。

## タスク

- [x] T001 [Acceptance] 新しいパスへの参照が 404 になることを、E2E と HTTP ハンドラーのテストで RED として観測する。
- [x] T002 [App] TypeSpec、Go、スキーマ、フロントエンドの識別子を改め、生成物を作り直す。
- [x] T003 [Docs] 用語集、仕様、設計文書、コメント、未完了の work item の日本語を改める。
- [x] T004 [UI] ja と en の画面の文言を改める。
- [x] T005 [Docs] ベースラインを凍結し直し、リリースノートを書く。
- [x] T006 [Verify] 変更を検証する。

## 検証

- `mise run test-go-test -- ./backend/shared/http/server_http TestForeignTenantAdminSeesProvisioningAsMissing`
- `mise run test-go-test -- ./backend/provisioning/handlers_http TestAdminTaskListSetsLinkHeaderWhenMorePagesExist`
- `mise run check-spec`
- `mise run check-work-items`
- `mise run verify`
- `mise run test-ui-e2e`

## リスク

- **別の意味の delivery まで書き換える。** 一括置換は Provisioning のファイルに限り、ほかのファイルでは `ProvisioningDelivery` のような Provisioning 固有の識別子だけを置換する。置換後に Shared Signals とログアウト通知の名前が残っていることを確かめる。
- **生成物との食い違い。** sqlc、OpenAPI、実行時のルート情報、ルート優先度の参照文書を作り直し、`check-generated-contract` と `check-contract-drift` で確かめる。

## 完了

- **Completed At**: 2026-09-25
- **Summary**:
  `mise run spec-diff` は、`REQ-PLATFORM-003` と `REQ-PROVISIONING-001`、`003` から `010`、`012` から `018` のシナリオ、`RFC7643-OUT-CORE-RESOURCES`、`RFC7643-OUT-GROUP-RESOURCES`、`RFC7644-OUT-BULK` の標準行、状態遷移 `ProvisioningDeliveryLifecycle` から `ProvisioningTaskLifecycle` への改名を変更として挙げた。
  どのシナリオと標準行も、「配信」を「プロビジョニングタスク」または「プロビジョニングする」へ改めただけで、条件と要求する結果は変わらない。
  TypeSpec では、`ListProvisioningDeliveries`、`GetProvisioningDelivery`、`RetryProvisioningDelivery`、`ProvisioningDelivery`、`ProvisioningDeliveryStatus`、`ProvisioningDeliveryStarted` と二つのエラーを取り除き、同じ内容の `ProvisioningTask` の名前の宣言を加えた。
  管理 API のパスは `/provisioning/tasks` と `task_id` に、一覧の応答は `tasks` に、ジョブの種類は `provisioning_task` に、テーブルは `provisioning_tasks` に変わった。
  管理画面の ja の「配送」と en の delivery も、プロビジョニングタスクへ改めた。
- **Primary Use Case Evidence**:
  - id: read-provisioning-task-at-renamed-path
    unit_red: パスを `/provisioning/tasks` へ変えた一覧のテストが、ルートが旧パスのままのため status=404 で失敗した（TestAdminTaskListSetsLinkHeaderWhenMorePagesExist で観測）。
    e2e_red: パスを `/provisioning/tasks/{task_id}` へ変えた TestForeignTenantAdminSeesProvisioningAsMissing が、本番の組み立てに新しいパスがないため所有テナントの参照が status=404 になり、前提の確認で失敗した。
    unit_fault_injection: 改名後に routes.go のパスだけを `/provisioning/deliveries` へ戻すと、TestAdminTaskListRejectsCursorFromAnotherTenant が 400 ではなく 404 を受けて失敗した。
    e2e_fault_injection: 本番の組み立てに旧パスだけを配線する誤実装は e2e_red の状態そのものであり、TestForeignTenantAdminSeesProvisioningAsMissing が所有テナントの 404 で失敗した。
- **Change-Resistance Results**:
  振る舞いを変えない改名なので、変異テストは実施していない。
  代わりに、改名で壊れうる箇所を次の検査で確かめた。
  旧パスを残す誤りは上の 2 件のテストが検出した。
  生成物との食い違いは `check-generated-contract`、`check-contract-drift`、`check-route-reference` が検出する。
  別の意味の delivery を誤って置換した箇所（bootstrap の Shared Signals の配線、`backchannel_logout_delivery`、Shared Signals のテスト名）は、ビルドの失敗と `check-work-items` の参照検査で見つかり、元に戻した。
- **Verification Results**:
  - `mise run test-go-changed` - 成功
  - `mise run check-spec` - 成功
  - `mise run test-ui-e2e` - 成功（39 件）
  - `mise run verify` - 成功（コミット前は、完了済みの記録 5 件と進行中の wi-496 に `removal_notice` を求めて失敗した。変更を `main` にコミットした後は成功した）

---
status: completed
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-09-03
change_kind: feature
priority: p1
depends_on: [wi-460-cross-tenant-health-control-plane-membership]
evidence_policy: risk-based-v3
documentation_impact:
  level: upgrade_note
  reason: テナント管理 API から `all_tenants` を除き、横断操作を別経路のシステム API へ移すため、既存の `system_admin` 向け自動化は呼び先を変える必要がある。
  references:
    - { kind: release_note, path: docs/releases/changes/wi-462.md }
    - { kind: upgrade_note, path: docs/releases/upgrades/wi-462.md }
initial_context:
  specification:
    - docs/contexts/audit/scenarios.feature.md#REQ-AUDIT-001
    - docs/contexts/audit/decisions.md
    - docs/contexts/jobs/scenarios.feature.md#REQ-JOBS-012
    - docs/contexts/jobs/scenarios.feature.md#REQ-JOBS-013
    - docs/contexts/system/scenarios.feature.md#REQ-SYSTEM-018
    - docs/contexts/system/decisions.md
    - docs/structure.md
  typespec:
    - IdMagic.Audit.Operations.ListAdminAuditEvents
    - IdMagic.Audit.Operations.GetAdminAuditEvent
    - IdMagic.Audit.Operations.ExportAdminAuditEvents
    - IdMagic.Jobs.Operations.ListJobs
    - IdMagic.Jobs.Operations.GetJob
    - IdMagic.Jobs.Operations.CancelJob
  source:
    - backend/audit/handlers_http
    - backend/jobs/handlers_http
    - backend/jobs/usecases/admin.go
    - backend/shared/http/support_http/auth.go
    - backend/shared/http/server_http/priority_class.go
    - frontend/src/api/admin.ts
    - frontend/src/features/admin-audit-events
    - frontend/src/features/admin-jobs
    - frontend/src/lib/systemNav.ts
    - frontend/src/routes/admin/audit_events.tsx
    - frontend/src/routes/admin/jobs.tsx
    - frontend/src/routes/system
  tests:
    - backend/shared/http/server_http/control_plane_boundary_test.go
    - backend/audit/handlers_http/admin_audit_event_handler_test.go
    - backend/jobs/handlers_http/admin_job_handler_test.go
    - frontend/src/features/admin-audit-events/AdminAuditEventsPage.test.tsx
    - frontend/src/features/admin-jobs/AdminJobsPage.test.tsx
    - frontend/tests/e2e/ui-scenario-smoke.spec.ts
  stop_before_reading:
    - backend/audit/db_postgres
    - backend/jobs/db_postgres
    - spec/generated
affected_spec:
  - { path: docs/contexts/audit/scenarios.feature.md, requirement: REQ-AUDIT-001 }
  - { path: docs/contexts/audit/scenarios.feature.md, requirement: REQ-AUDIT-007 }
  - { path: docs/contexts/jobs/scenarios.feature.md, requirement: REQ-JOBS-012 }
  - { path: docs/contexts/jobs/scenarios.feature.md, requirement: REQ-JOBS-013 }
  - { path: docs/contexts/jobs/scenarios.feature.md, requirement: REQ-JOBS-015 }
  - { path: docs/contexts/system/scenarios.feature.md, requirement: REQ-SYSTEM-020 }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ListAdminAuditEvents }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.GetAdminAuditEvent }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ExportAdminAuditEvents }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ListSystemAuditEvents }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.GetSystemAuditEvent }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ExportSystemAuditEvents }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.ListJobs }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.GetJob }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.CancelJob }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.ListSystemJobs }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.GetSystemJob }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.CancelSystemJob }
primary_use_cases:
  - id: system-audit-cross-tenant-search
    requirement: REQ-AUDIT-007
    observable_result: 制御面主体がシステム経路で検索すると、全テナントの監査イベントが返り、同じ条件のエクスポートも同じ範囲を返す。
    unit_test: { path: backend/audit/handlers_http/admin_audit_event_handler_test.go, name: TestListSystemAuditEventsSpansEveryTenant, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/control_plane_boundary_test.go, name: TestSystemAuditEventsSpanEveryTenantForControlPlaneActor, task: test-go-race }
    unit_fault_model: システム経路のハンドラーが範囲を要求元テナントへ閉じ、他テナントのイベントを落とす。
    e2e_fault_model: システム経路が登録されず、または制御面判定を通さないまま登録され、横断参照へ到達できない。
  - id: system-jobs-cross-tenant-oversight
    requirement: REQ-JOBS-015
    observable_result: 制御面主体がシステム経路で他テナントの Job を一覧し、1 件を参照し、取り消すと状態が `canceled` になる。
    unit_test: { path: backend/jobs/handlers_http/admin_job_handler_test.go, name: TestSystemJobHandlersSpanEveryTenant, task: test-go-race }
    e2e_test: { path: backend/shared/http/server_http/control_plane_boundary_test.go, name: TestControlPlaneJobOversightNeedsNoAdminRole, task: test-go-race }
    unit_fault_model: システム経路の取消しが範囲を確かめる前に Cancel を呼び、範囲外の Job を止めてから存在しないと答える。
    e2e_fault_model: システム経路の取消しが CSRF と Origin の検証を経ずに登録され、ブラウザー経路の変更操作が保護されない。
  - id: tenant-console-stays-in-tenant
    requirement: REQ-SYSTEM-020
    observable_result: テナント管理 API は `all_tenants` を付けても要求先テナントに閉じ、テナント管理コンソールに横断切替の UI が存在しない。
    unit_test: { path: frontend/src/features/admin-jobs/AdminJobsPage.test.tsx, name: does not offer a cross-tenant toggle to a control-plane operator, task: test-ui-unit }
    e2e_test: { path: frontend/tests/e2e/ui-scenario-smoke.spec.ts, name: system console cross-tenant surfaces are reachable and the admin console has no cross-tenant toggle, task: test-ui-e2e }
    unit_fault_model: ページが操作者のロールと realm から横断切替を復活させる。
    e2e_fault_model: テナント管理コンソールが横断能力を持つシステム API を呼ぶ配線に戻る。
---

# 監査とジョブのテナント横断 API および画面をシステムコンソールへ移す

## Motivation

システムコンソール（`/system`）は、テナント一覧、署名鍵ヘルス、DEK ヘルスを持つ `system_admin` 専用の制御面 UI として、テナント管理コンソール（`/admin`）から別の経路、シェル、ナビゲーション、配色で分離されている。

しかし監査イベントとジョブは、テナント管理コンソールの画面が操作者のロールと realm を見て、テナント内表示とテナント横断表示を切り替えている。

監査画面は横断検索と横断エクスポートを行い、ジョブ画面は横断一覧から別テナントのジョブを選択して詳細参照と取消しまで行える。

同じ画面がテナント管理と制御面の権限文脈を切り替えるため、利用者は現在どちらの範囲を操作しているかを画面のチェックボックスから判断しなければならず、システムコンソールを分けた意図が成立していない。

画面だけを移しても認可境界は分かれない。

現在の監査とジョブの管理 API は、同じ経路で `all_tenants` または操作者のロールから範囲を切り替えるため、`system_admin` はテナント管理コンソールの API を直接呼んで横断検索、別テナントの詳細参照、取消しを実行できる。

テナント管理コンソールをテナント内へ閉じるには、画面だけでなく API の経路と入力から横断能力を除く必要がある。

仕様にも欠落がある。

Audit の決定と TypeSpec は制御面テナントの `system_admin` による横断検索を説明するが、`docs/contexts/audit/scenarios.feature.md` に成功シナリオがない。

Jobs は `ListJobs` の横断一覧を TypeSpec とテストで説明する一方、`GetJob` と `CancelJob` が制御面主体に限って別テナントへ届く現在の挙動を TypeSpec とシナリオに記述していない。

## Scope

- 既存の `/api/admin/v1/audit_events` とその詳細およびエクスポートを、要求先テナントだけを扱う API に変更し、`all_tenants` 入力を契約と実装から削除する。
- `/api/admin/v1/system/audit_events` とその詳細およびエクスポートを追加し、制御面主体だけが全テナントを扱えるようにする。
- 既存の `/api/admin/v1/jobs` とその詳細および取消しを、要求先テナントだけを扱う API に変更し、`all_tenants` 入力を契約と実装から削除する。
- `/api/admin/v1/system/jobs` とその詳細および取消しを追加し、制御面主体だけが全テナントを扱えるようにする。
- `/admin/audit_events` と `/admin/jobs` を常に要求先テナントへ閉じ、`all_tenants` の URL 状態と切替 UI を削除する。
- `/system/audit-events` と `/system/jobs` を追加し、専用のシステム API だけを呼び出す。
- システムコンソールのジョブ取消し確認に、対象の `tenant_id`、ジョブ種別、ジョブ ID を表示する。
- Audit に制御面主体の横断検索と横断エクスポートを表す新しい規範シナリオを追加する。
- Jobs に制御面主体の横断一覧、詳細参照、取消しを表す規範シナリオを追加し、テナント管理者の操作範囲と分ける。
- System に、テナント横断 UI はシステムコンソールだけに置き、テナント管理コンソールは操作者のロールにかかわらず要求先テナントへ閉じるシナリオと決定を追加する。
- Audit と Jobs の TypeSpec にテナント内操作とシステム操作を別の操作記号として定義し、同じハンドラーへ暗黙に合流させない。
- `docs/structure.md` の「制御面のテナント管理だけを専用経路へ登録する」という現在の実装と異なる説明を修正する。
- 既存の `system_admin` 向け横断呼出しが新しいシステム API へ移ることをアップグレードノートへ記載する。

## Out of Scope

- テナント横断ヘルスの認可修正と `requireSystemAccount` の条件統一。
  依存する [[wi-460-cross-tenant-health-control-plane-membership]] が先に扱う。
- 制御面操作へ到達できる資格情報の種類。
  [[wi-461-control-plane-credential-boundary]] が扱う。
- 一般利用者向けの公開入口から制御面 API を到達不能にすること、または管理コンソールを別ホストへ移すこと。
  [[wi-459-api-process-plane-separation-decision]] の再検討条件に従う。
- 容量、リリース、バックアップ、機能レジストリなど、現在 HTTP 管理 API を持たない運用機能をシステムコンソールへ追加すること。

## Design

### UI の不変条件

**テナント境界を越える操作はシステム API とシステムコンソールだけに置き、テナント管理 API とテナント管理コンソールは操作者が `system_admin` であっても要求先テナントへ閉じる。**

この不変条件は、画面を経由しない直接の API 呼出しにも適用する。

システムコンソールの各経路は `requireSystemAccount` を通り、システム API は [[wi-460-cross-tenant-health-control-plane-membership]] の制御面主体判定を行う。

### 権限文脈を経路で固定する

テナント側の画面と API は `all_tenants` を受け付けず、API クライアントへも送らない。

古い画面のブックマークに `allTenants=true` が残っていても、検証済み検索状態から削除してテナント内表示へ戻す。

システム側は利用者が切り替えるチェックボックスを持たず、範囲が全テナントに固定された専用 API を呼ぶ。

システム API は `all_tenants` を入力として受け取らない。

範囲を利用者入力で切り替えず、どの操作記号と経路を呼んだかで固定する。

権限文脈を URL 経路とシェルで固定すると、再読込、ページング、絞り込み変更のたびにロールから表示範囲を再推定せずに済む。

### 画面構成

テナント側とシステム側は別のページ境界を持ち、ページ境界が `AdminShell` または `SystemShell`、API の範囲、取消し確認の文言を決める。

表、絞り込み入力、状態バッジなど、権限判断を持たない表示部品は共有してよい。

同じページコンポーネントへ `canCrossTenant` を渡して権限文脈を切り替える形は残さない。

監査のシステムページは検索、ページング、エクスポートを専用のシステム API で提供する。

ジョブのシステムページは一覧だけでなく、現在サーバーが許可している別テナントの詳細参照と取消しも扱う。

取消しは変更操作であるため、確認ダイアログに `tenant_id`、ジョブ種別、ジョブ ID を表示し、対象テナントを一覧の列にも常時表示する。

### 規範の分割

Audit の横断検索は TenantAdministrator の `REQ-AUDIT-001` と主体も範囲も異なるため、新しい SystemAdministrator のシナリオにする。

Jobs も TenantAdministrator の `REQ-JOBS-012` と `REQ-JOBS-013` へ制御面の成功経路を `ALT` として詰め込まず、新しい SystemAdministrator のシナリオで横断一覧、詳細参照、取消しを一続きに記述する。

System の新しいシナリオは、テナント管理 API とシステム API の範囲が経路によって固定され、対応するコンソール以外から横断能力へ到達できないことを所有する。

### 解決した問い

**テナント管理 API の受け入れ条件**。テナント側のハンドラーは、要求先テナントに所属し `admin` ロールを持つ操作者だけを受け入れ、制御面主体の特例を持たない。

範囲だけをテナント内へ固定して受け入れの分岐を残すと、経路が認可を表すという不変条件を保ったまま、ロールを見る分岐がハンドラーに残ってしまう。

この特例を外しても到達できなくなる能力はない。制御面主体はシステム API から全テナント（制御面テナント自身を含む）へ届く。

Jobs では、この結果として `requireJobAdministrator` が `ResolveAdminActor` と同じ判定になるため、専用のヘルパーを廃止する。

Audit の `RequireAuditReader` は認証イベントバケットの参照とも共有するため受け入れ条件を変えず、テナント側の範囲を `all_tenants` に依らず要求元の所属テナントへ固定するだけに留める。

**監査の検索選択肢**。`/api/admin/v1/audit_events/search_options` はシステム側の双子を作らず、テナント側の 1 本を両コンソールが呼ぶ。

返すのは `event.type`、`outcome`、`actor.type`、`delegation.mode` の語彙だけで、テナントの記録を一切含まないため、経路が範囲を表すという不変条件の対象にならない。

**カーソルの束縛**。監査のカーソルは絞り込みの指紋に範囲の識別子を混ぜ、テナント側で発行したカーソルをシステム側へ持ち込んでも続きにならないようにする。

制御面主体のテナント側呼出しでは所属テナントが `default` であり、範囲を混ぜなければ両経路の指紋が一致してしまう。

ジョブのカーソルは既存の指紋が `TenantScope` の両フィールドを含むため、範囲が変わると指紋も変わる。

**優先度クラス**。システム側の監査一覧とエクスポートは、テナント側と同じ理由で走査になるため `management_bulk` へ分類する。

システム側のジョブは `/api/admin/v1` の既定である `management` に属する。

### 却下した選択肢

- **テナント管理コンソールに横断切替を残し、システムコンソールにも同じ画面を置く。** 2つの入口が残り、コンソール分離の不変条件を満たさない。
- **画面だけを分け、両方から同じ API を呼ぶ。** 直接の API 呼出しではテナント管理側から横断できるため、経路が認可範囲を表さない。
- **システムコンソールを廃止し、すべてをテナント管理コンソールのロール分岐へ統合する。** 横断能力が各画面の条件分岐へ散らばり、運用者が制御面にいることをシェルと経路で確認できなくなる。
- **監査とジョブの一覧だけを移し、エクスポート、詳細参照、取消しをテナント側へ残す。** 一覧で見つけた別テナントの対象を操作するためにコンソールを行き来することになり、横断操作の入口が一つにならない。
- **`docs/structure.md` の説明に合わせて制御面 API を `/realms/default` だけへ登録する。** 経路登録の変更は公開入口と配備境界の判断を伴い、画面配置の変更より広い。

## Plan

1. Audit、Jobs、System の規範シナリオと決定を更新し、テナント操作とシステム操作を別の TypeSpec 操作として定義する。
2. テナント管理 API から `system_admin` が横断できる現在の挙動を HTTP 境界で観測し、RED を確認する。
3. テナント管理 API を要求先テナントへ固定し、制御面主体だけを受け入れるシステム API を別経路で登録する。
4. 監査とジョブから権限判断を持たない表示部品を抽出し、テナント側とシステム側に別のページ境界を作る。
5. テナント側から `all_tenants` の URL 状態、切替 UI、ロールと realm による分岐を削除する。
6. システム側の監査検索とエクスポート、ジョブ一覧、詳細参照、取消しを専用 API へ接続し、ナビゲーションと日英辞書を更新する。
7. システム側の取消し確認に対象テナントとジョブ識別情報を表示する。
8. テナント API が自テナントへ閉じ、システム API が制御面主体に限って全テナントを扱うことを HTTP、UI 単体、E2E の各境界で確認する。
9. アップグレードノートを作成し、仕様生成物を再生成して検査を通す。

## Tasks

検証の刻みは各タスクに書いた recipe を使う。赤緑ごとに選び直さない。

- [x] T001 [Spec] Audit、Jobs、System の規範シナリオ、決定、TypeSpec 操作、`docs/structure.md` を更新する。
  `REQ-AUDIT-007`、`REQ-JOBS-015`、`REQ-SYSTEM-020` を割り当てる。recipe: `mise run check-spec`。
- [x] T002 [Acceptance] テナント管理 API から横断できる現在の挙動を HTTP 境界で観測し、RED を確認する。
  `TestSystemAuditEventsSpanEveryTenantForControlPlaneActor` と `TestControlPlaneJobOversightNeedsNoAdminRole`
  (`REQ-AUDIT-007` / `REQ-JOBS-015`)。recipe:
  `mise run test-go-package -- ./backend/shared/http/server_http`。
- [x] T003 [App] テナント管理 API を要求先テナントへ固定し、制御面主体だけを受け入れるシステム API を別経路で登録する。
  `TestListSystemAuditEventsSpansEveryTenant` (`REQ-AUDIT-007`) と `TestSystemJobHandlersSpanEveryTenant`
  (`REQ-JOBS-015`)。recipe: `mise run test-go-test -- ./backend/audit/handlers_http <test>`、
  `mise run test-go-test -- ./backend/jobs/handlers_http <test>`、GREEN 後に `mise run lint-go`。
- [x] T004 [Acceptance] テナント管理 API へ `all_tenants` を付けても横断せず、別テナントの識別子を指定した詳細参照と取消しが対象を変更しないことを確認する。
  `TestTenantAdminApisStayInsideRequestTenant` (`REQ-SYSTEM-020` / `REQ-JOBS-012` / `REQ-JOBS-013`)。recipe:
  `mise run test-go-package -- ./backend/shared/http/server_http`。
- [x] T005 [App] 監査とジョブの表示部品を、権限判断を持たない単位へ分ける。
  recipe: `mise run test-ui-unit-file -- <file>`。
- [x] T006 [App] テナント側の `all_tenants` 状態、切替 UI、権限分岐を削除する。
  `AdminJobsPage.test.tsx` の `does not offer a cross-tenant toggle to a control-plane operator`
  (`REQ-SYSTEM-020`)。recipe: `mise run test-ui-unit-file -- src/features/admin-jobs/AdminJobsPage.test.tsx`。
- [x] T007 [App] システム側に監査の検索とエクスポート、ジョブの一覧、詳細参照、取消しと対象テナントを含む確認を追加する。
  recipe: `mise run test-ui-unit-file -- <file>`。
- [x] T008 [App] `systemNav`、シェル、日英辞書を更新する。
  recipe: `mise run test-ui-unit-file -- src/lib/systemNav.test.ts`。
- [x] T009 [Acceptance] テナント側の非横断とシステム側の横断を UI 単体テストと E2E テストで確認する。
  `ui-scenario-smoke.spec.ts` の
  `system console cross-tenant surfaces are reachable and the admin console has no cross-tenant toggle`
  (`REQ-SYSTEM-020`)。recipe: `mise run test-ui-e2e-file -- tests/e2e/ui-scenario-smoke.spec.ts`。
- [x] T010 [Docs] API の移行方法をアップグレードノートへ記載する。
  recipe: `mise run check-work-items`、`mise run check-links`。
- [x] T011 [Verify] 仕様生成物を再生成し、互換性検査を含む検査を通す。
  recipe: `mise run spec-render`、`mise run check-api-compat`、`mise run check-contract-drift`、
  `mise run check-route-reference`、`mise run verify`。

## Verification

- `mise run test-ui-unit`
- `mise run test-ui-e2e`
- `mise run test-go-race`
- `mise run check-api-compat`
- `mise run check-spec`
- `mise run check-ids`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

リスクは high とする。

公開 API の経路と範囲を変更するため、既存の `system_admin` 向け自動化がテナント管理 API へ `all_tenants` を付けている場合は新しいシステム API への移行が必要になる。

画面移設または経路分離を誤ると、システム運用者が横断監査やジョブ操作へ到達できなくなるか、テナント管理 API に横断能力が残る。

監査では検索条件、双方向ページング、エクスポートが同じ `all_tenants` 範囲を保つことを確認する。

カーソルはテナント範囲と絞り込み条件へ束縛されるため、テナント側とシステム側の URL または状態を共有しない。

ジョブの取消しは別テナントの実行へ影響するため、システムページで対象の `tenant_id` を常時表示し、確認時にも再掲する。

新しい Audit、Jobs、System の `REQ` 番号と公開 API 操作を割り当てるため、`reversibility` は irreversible とする。

画面の配置自体は戻せるが、割り当てた規範 ID は再利用しない。

## Completion

- **Completed At**: 2026-09-12
- **Summary**:
  `mise run spec-diff` が示す規範上の差分は、追加が `REQ-AUDIT-007`、`REQ-JOBS-015`、`REQ-SYSTEM-020` の 3 つ、変更が `REQ-JOBS-012` と `REQ-JOBS-013` の 2 つ、そして TypeSpec 宣言の追加が `ListSystemAuditEvents`、`ExportSystemAuditEvents`、`GetSystemAuditEvent`、`ListSystemJobs`、`GetSystemJob`、`CancelSystemJob` の 6 つである。
  意味の差は、テナント境界を越える範囲を利用者入力ではなく経路が決めるようになったことである。
  `REQ-AUDIT-007` と `REQ-JOBS-015` は制御面主体が専用のシステム経路で全テナントを検索・参照・エクスポート・取り消せることを新たに所有し、`REQ-SYSTEM-020` は横断の入口をシステムコンソールだけに置くことを所有する。
  `REQ-JOBS-012` と `REQ-JOBS-013` からは制御面の成功経路が抜け、代わりにテナント管理経路が横断を求める入力を添えられても要求先テナントへ閉じることを述べる例に変わった。
  `AdminJobQuery` から `all_tenants` を削除したため、テナント管理 API には範囲を動かす入力がもう存在しない。
- **Primary Use Case Evidence**:
  - id: system-audit-cross-tenant-search
    unit_red: "`TestListSystemAuditEventsSpansEveryTenant` は `/api/admin/v1/system/audit_events` が未登録のため 404 で失敗した。"
    e2e_red: "`TestSystemAuditEventsSpanEveryTenantForControlPlaneActor` は組み立て済みルーターに同経路が無く、検索・エクスポート・1 件参照のいずれも 404 で失敗した。"
    unit_fault_injection: "`systemAuditScope()` を `auditScope{}` へ変えて範囲を要求元テナントへ閉じると `TestListSystemAuditEventsSpansEveryTenant` が落ちた。`exportAuditEvents` の範囲をテナント側へ差し替えた変異も落ちた。"
    e2e_fault_injection: "システム経路の `RequireControlPlaneUser` を `RequireAuditReader` へ緩めると、一覧と 1 件参照のいずれの変異も HTTP 境界のテストが落とした。"
  - id: system-jobs-cross-tenant-oversight
    unit_red: "`TestSystemJobHandlersSpanEveryTenant` は `/api/admin/v1/system/jobs` が未登録のため 404 で失敗した。"
    e2e_red: "`TestControlPlaneJobOversightNeedsNoAdminRole` をシステム経路へ向けた時点で、一覧・詳細・取消しのすべてが 404 で失敗した。"
    unit_fault_injection: "システム経路の `RequireControlPlaneUser` を `ResolveAdminActor` へ緩める変異は、一覧・1 件参照・取消しのいずれも落ちた。"
    e2e_fault_injection: "`handleCancelSystemJob` から `VerifyBrowserRequest` を外す変異は、Jobs パッケージのテストでは生き残り (常に正しい CSRF を送るため)、HTTP 境界の `TestSystemJobCancelRefusesWithoutBrowserProof` が落とした。効果まで読んでいるので、拒否を書いたうえで取り消すハンドラーも通らない。"
  - id: tenant-console-stays-in-tenant
    unit_red: "`does not offer a cross-tenant toggle to a control-plane operator` は、テナント管理画面に横断切替が残っている状態で失敗した。"
    e2e_red: "`system console cross-tenant surfaces are reachable and the admin console has no cross-tenant toggle` は `/system/audit-events` と `/system/jobs` が存在しない状態で失敗した。"
    unit_fault_injection: "`JobsBrowser` へ横断のチェックボックスを戻すと単体テストが落ちた。`AdminJobsPage` の `listAdminJobs` を `listSystemJobs` へ差し替える変異も落ちた。"
    e2e_fault_injection: "同じチェックボックスを戻した状態で E2E を走らせると、`/admin/jobs` が依然として横断切替を出しているという失敗で落ちた。`validateAuditEventsSearch` へ `allTenants` を戻す変異は経路のテストが落とした。"
- **Change-Resistance Results**:
  変更した純粋なロジックはテナントの範囲を決める判定であり、そこへ系統的に変異を当てた。17 件中 16 件を初回で殺し、生存した 1 件はテストを足して殺した。残る 1 件はパッケージ単位では生存し、上位の境界で殺している。
  - killed: テナント範囲を横断へ変える変異 (`tenantAuditScope` に `allTenants`、ジョブの一覧・1 件参照・取消しの `TenantScope` を `AllTenants` へ)。
  - killed: システム範囲をテナント内へ狭める変異 (`systemAuditScope`、システム側エクスポート)。
  - killed: 1 件参照の可視判定 `auditScope.includes` を常に真にする変異。
  - killed: カーソルの指紋から範囲を落とす変異 (`auditEventQueryHash`)。テナント経路のカーソルがシステム経路の続きとして読めてしまう取り違えを、`TestTenantAuditCursorDoesNotContinueTheSystemSearch` が捕らえた。
  - killed: システム経路の制御面判定を `RequireAuditReader` / `ResolveAdminActor` へ緩める 4 件の変異。
  - killed: テナント経路の受け入れを `RequireAdmin` から `ResolveAdminActor` へ戻す変異。これは実装中に実際に踏んだ誤りで、`ResolveAdminActor` は解決するだけで認可しない。テストが先に捕らえた。
  - killed: `parseListJobsQuery` へ `all_tenants` の解釈を戻す変異。
  - killed: UI の 3 変異 (横断チェックボックスの復活、テナント画面からシステム API の呼出し、`allTenants` の URL 状態の復活)。うち 2 つは E2E でも落ちる。
  - 当初 survived → テストを追加して killed: テナント側エクスポートの範囲をシステム側へ差し替える変異。一覧だけを閉じてエクスポートを開いたままにする誤りを見るテストが無かったため、`TestAdminAuditEventsIgnoreAnyCrossTenantInput` へエクスポートの表明を追加した。
  - パッケージでは survived / 境界で killed: `handleCancelSystemJob` から `VerifyBrowserRequest` を外す変異。Jobs のパッケージテストは常に正しい CSRF を送るので区別できない。配線ごと通る `backend/shared/http/server_http` の検査が落とす。この分担は意図したもので、パッケージ側に同じ検査を重ねてはいない。
  - 手法の限界: 変異はテナント範囲の判定に限って当てており、ページングの境界計算、絞り込みの解析、UI の描画には当てていない。いずれもこの作業で意味を変えていない部分である。
- **Verification Results**:
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - passed (28 tests)
  - `mise run check-spec` - passed
  - `mise run check-api-compat` - passed (破壊的変更なし)
  - `mise run check-contract-drift` / `check-status-drift` / `check-admin-scopes` / `check-route-reference` / `check-security-controls` - passed
  - `mise run check-work-items` - passed

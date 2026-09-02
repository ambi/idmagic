---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-09-03
change_kind: feature
priority: p1
depends_on: [wi-460-cross-tenant-health-control-plane-membership]
affected_spec:
  - { path: docs/contexts/audit/scenarios.md, requirement: REQ-AUDIT-001 }
  - { path: docs/contexts/jobs/scenarios.md, requirement: REQ-JOBS-012 }
  - { path: docs/contexts/jobs/scenarios.md, requirement: REQ-JOBS-013 }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ListAdminAuditEvents }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.GetAdminAuditEvent }
  - { path: spec/contexts/audit/main.tsp, symbol: IdMagic.Audit.Operations.ExportAdminAuditEvents }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.ListJobs }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.GetJob }
  - { path: spec/contexts/jobs/main.tsp, symbol: IdMagic.Jobs.Operations.CancelJob }
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

Audit の決定と TypeSpec は制御面テナントの `system_admin` による横断検索を説明するが、`docs/contexts/audit/scenarios.md` に成功シナリオがない。

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

- [ ] T001 [Spec] Audit、Jobs、System の規範シナリオ、決定、TypeSpec 操作、`docs/structure.md` を更新する。
- [ ] T002 [Acceptance] テナント管理 API から横断できる現在の挙動を HTTP 境界で観測し、RED を確認する。
- [ ] T003 [App] テナント管理 API を要求先テナントへ固定し、制御面主体だけを受け入れるシステム API を別経路で登録する。
- [ ] T004 [Acceptance] テナント管理 API へ `all_tenants` を付けても横断せず、別テナントの識別子を指定した詳細参照と取消しが対象を変更しないことを確認する。
- [ ] T005 [App] 監査とジョブの表示部品を、権限判断を持たない単位へ分ける。
- [ ] T006 [App] テナント側の `all_tenants` 状態、切替 UI、権限分岐を削除する。
- [ ] T007 [App] システム側に監査の検索とエクスポート、ジョブの一覧、詳細参照、取消しと対象テナントを含む確認を追加する。
- [ ] T008 [App] `systemNav`、シェル、日英辞書を更新する。
- [ ] T009 [Acceptance] テナント側の非横断とシステム側の横断を UI 単体テストと E2E テストで確認する。
- [ ] T010 [Docs] API の移行方法をアップグレードノートへ記載する。
- [ ] T011 [Verify] 仕様生成物を再生成し、互換性検査を含む検査を通す。

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

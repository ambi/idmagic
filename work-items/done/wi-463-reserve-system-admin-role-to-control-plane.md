---
status: completed
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-09-03
change_kind: bugfix
priority: p1
depends_on: [wi-460-cross-tenant-health-control-plane-membership]
evidence_policy: risk-based-v3
documentation_impact:
  level: upgrade_note
  reason: "通常テナントの User と Group、およびすべての Agent への `system_admin` の新規割当てを、公開 API と CSV が拒否するようになる。既存の割当ては自動削除しないため、運用者は検出方法と手動での除去手順を知る必要がある。"
  references:
    - { kind: release_note, path: docs/releases/changes/wi-463.md }
    - { kind: upgrade_note, path: docs/releases/upgrades/wi-463.md }
initial_context:
  specification:
    - docs/domain/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-004
    - docs/domain/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-014
    - docs/domain/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-015
    - docs/domain/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-026
    - docs/domain/identity-management/scenarios.feature.md#REQ-IDMANAGEMENT-032
    - docs/domain/identity-management/decisions.md
    - docs/domain/identity-management/glossary.md
    - docs/design/security/authorization.md
  typespec:
    - IdMagic.Contract.InvalidRoleError
    - IdMagic.Contract.UserImportRowError
    - IdMagic.Contract.GroupImportRowError
    - IdMagic.IdentityManagement.Operations.CreateAdminUser
    - IdMagic.IdentityManagement.Operations.UpdateAdminUser
    - IdMagic.IdentityManagement.Operations.CreateGroup
    - IdMagic.IdentityManagement.Operations.UpdateGroup
    - IdMagic.IdentityManagement.Operations.RegisterAgent
    - IdMagic.IdentityManagement.Operations.UpdateAgent
  source:
    - backend/idmanagement/usecases/helpers.go
    - backend/idmanagement/user/usecases/admin_users.go
    - backend/idmanagement/user/usecases/user_import_planner.go
    - backend/idmanagement/user/usecases/federated_user.go
    - backend/idmanagement/group/usecases/admin_groups.go
    - backend/idmanagement/group/usecases/group_import_planner.go
    - backend/idmanagement/agent/usecases/admin_agents.go
    - backend/idmanagement/user/handlers_http/admin_user_handler.go
    - backend/idmanagement/group/handlers_http/admin_group_handler.go
    - backend/idmanagement/agent/handlers_http/admin_agent_handler.go
    - backend/idmanagement/handlers_http/routes.go
    - backend/shared/http/support_http/auth.go
    - backend/sourcing/scim/usecases/users.go
    - backend/sourcing/scim/usecases/groups.go
    - backend/cmd/internal/bootstrap/seed.go
  tests:
    - backend/idmanagement/handlers_http/refusal_effects_test.go
    - backend/idmanagement/user/usecases/user_import_planner_test.go
    - backend/idmanagement/group/usecases/group_import_planner_test.go
  stop_before_reading: [frontend, infra, docs/runbooks, backend/provisioning]
primary_use_cases:
  - id: control-plane-group-grants-system-admin
    requirement: REQ-IDMANAGEMENT-032
    observable_result: 制御面テナントの Group へ `system_admin` を付与すると、所属する制御面 User の有効ロールに `system_admin` が現れる。
    unit_test:
      path: backend/idmanagement/group/usecases/admin_groups_test.go
      name: TestCreateGroupKeepsTheReservedRoleInsideTheControlPlane
      task: test-go-race
    e2e_test:
      path: backend/idmanagement/handlers_http/reserved_role_test.go
      name: TestControlPlaneGroupGrantsSystemAdminToItsMembers
      task: test-go-race
    unit_fault_model: 予約ロール検査が対象の所属テナントを見ず、制御面テナントの Group への `system_admin` まで拒否する。
    e2e_fault_model: Group の作成経路が共通検査へ対象種別を `RoleTargetAgent` として渡し、制御面 Group の作成が 422 になって所属 User の有効ロールへ何も乗らない。
affected_spec:
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-004 }
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-032 }
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-014 }
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-015 }
  - { path: docs/domain/identity-management/scenarios.feature.md, requirement: REQ-IDMANAGEMENT-026 }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.CreateAdminUser }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.UpdateAdminUser }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.CreateGroup }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.UpdateGroup }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.RegisterAgent }
  - { path: spec/contexts/identity-management/main.tsp, symbol: IdMagic.IdentityManagement.Operations.UpdateAgent }
---

# `system_admin` を制御面 User のための予約ロールにする

## Motivation

`system_admin` は、制御面テナントに所属する人間のシステム運用者を表す製品ロールである。

しかし現在の `NormalizeRoles` は空白除去、空要素の拒否、重複除去、整列だけを行い、ロール名の意味を検証しない。

そのため任意のテナントの管理者は、User または Group の JSON API と CSV インポートから `system_admin` を保存できる。

Agent の登録と更新も同じ正規化処理を使うため、どのテナントの Agent にも `system_admin` を保存できる。

[[wi-460-cross-tenant-health-control-plane-membership]] が制御面操作に所属テナントを必須とするため、制御面テナント外に保存された `system_admin` は同作業項目の完了後にテナント横断能力を持たない。

それでも同じロール名を通常テナントの任意文字列として使える状態を残すと、管理 UI、監査、将来の認可処理が `system_admin` を製品ロールとして扱うのか、テナント固有ロールとして扱うのかを判別できない。

## Scope

- `system_admin` は制御面テナントの User に直接付与するか、制御面テナントの Group から User へ付与する場合だけ有効な予約ロールであると、IdentityManagement の決定と用語集へ記録する。
- 通常テナントの User と Group、すべての Agent への `system_admin` の新規割当てを拒否する。
- User と Group の作成、更新、CSV プレビュー、CSV 適用、および Agent の登録と更新を同じ予約ロール検査へ通す。
- 制御面テナントの User と Group には、従来どおり `system_admin` を割り当てられるようにする。
- 予約ロールの許可と拒否を表す新しい `REQ-IDMANAGEMENT-NNN` シナリオを追加し、関連する TypeSpec 文書コメントへ制約を記述する。
- JIT、SCIM、外部プロビジョニングなど、roles を書き込まない User 作成または更新経路が予約ロールを導入しないことを棚卸しする。
- 既存データの検出方法と手動での除去手順をアップグレードノートに記載する。

## Out of Scope

- `admin` やテナント固有ロールを予約語にすること。
- ロール語彙全体を閉じた enum にすること。
- 「操作者が自分の持つロールだけを付与できる」という一般的な権限委譲規則を導入すること。
- 制御面テナント外に保存済みの `system_admin` をデータベース移行で自動削除すること。
- 制御面操作の認可条件。
  依存する [[wi-460-cross-tenant-health-control-plane-membership]] が先に扱う。

## Design

### 予約ロールの規則

`system_admin` の保存規則は対象の種類と所属テナントで決め、操作者が持つロールでは決めない。

| 対象 | `default` テナント | その他のテナント |
| --- | --- | --- |
| User | 許可する | 拒否する |
| Group | 許可する | 拒否する |
| Agent | 拒否する | 拒否する |

Group は自身がシステム運用者になるのではなく、制御面 User の有効ロールへ `system_admin` を付与する束として許可する。

Agent は `docs/design/security/authorization.md` でシステム運用者とは異なる主体として定義され、制御面の管理 API は User と ApiToken だけを受け入れるため、`system_admin` を割り当てない。

通常テナントで `catalog:read` など任意のロール名を使う現在のモデルは維持し、予約する文字列は `system_admin` だけに限定する。

### 検査の配置

`NormalizeRoles` は Agent を含む複数の対象で使われ、対象種別と tenant ID を受け取らないため、予約ロールの判断を加えない。

字句の正規化と製品上の割当て規則を分け、`ValidateRoleAssignment` を IdentityManagement のユースケース境界 (`backend/idmanagement/usecases`) に置く。

```go
type RoleAssignmentTarget string

const (
    RoleTargetUser  RoleAssignmentTarget = "user"
    RoleTargetGroup RoleAssignmentTarget = "group"
    RoleTargetAgent RoleAssignmentTarget = "agent"
)

// ReservedRoleSystemAdmin は制御面 User のための予約ロール名。
const ReservedRoleSystemAdmin = "system_admin"

// ErrReservedRole は予約ロールを割り当てられない対象へ新しく割り当てようとした場合に返る。
var ErrReservedRole = errors.New("role is reserved for control plane users")

func ValidateRoleAssignment(
    target RoleAssignmentTarget, tenantID string, current, next []string,
) error
```

`tenantID` は対象が所属するテナントであり、要求元のテナントではない。管理 API は同一テナント内の対象だけを扱うため両者は一致するが、判定の根拠は対象側に置く。

HTTP ハンドラーだけで検査すると CSV 適用と内部の再計画経路が抜けるため、User、Group、Agent の各書き込みユースケースと CSV 計画器が保存前に共通検査を呼ぶ。

CSV はプレビュー時に対象行を安定した `invalid_roles` で拒否し、適用時にも保存済みプレビュー結果を信頼せず、再計画で同じ検査を行う。

### roles を書き込む経路の棚卸し

| 経路 | 実装 | 扱い |
| --- | --- | --- |
| User の作成と更新 | `backend/idmanagement/user/usecases/admin_users.go` | 共通検査へ接続する |
| Group の作成と更新 | `backend/idmanagement/group/usecases/admin_groups.go` | 共通検査へ接続する |
| Agent の登録と更新 | `backend/idmanagement/agent/usecases/admin_agents.go` | 共通検査へ接続する |
| User CSV の計画器 | `backend/idmanagement/user/usecases/user_import_planner.go` | 共通検査へ接続する |
| Group CSV の計画器 | `backend/idmanagement/group/usecases/group_import_planner.go` | 共通検査へ接続する |
| JIT の連携ユーザー作成 | `backend/idmanagement/user/usecases/federated_user.go` | `Roles: []string{}` を書くだけで、要求からロールを受け取らない |
| SCIM の User と Group 作成 | `backend/sourcing/scim/usecases/users.go`、`groups.go` | `Roles: []string{}` を書くだけで、SCIM 属性からロールへ写像しない |
| 外部プロビジョニング | `backend/provisioning` | 外向きの配信だけを行い、`User` と `Group` へ書き戻さない |
| 開発用 seed | `backend/cmd/internal/bootstrap/seed.go` | 書き込み先が `tenancydomain.DefaultTenantID` に固定されており、規則と矛盾しない |

前半の 5 つだけが要求由来のロール文字列を受け取る。残りは予約ロールを導入する経路ではないため、検査を差し込まない。

### 判定は絶対集合ではなく差分で行う

`ValidateRoleAssignment` は保存後のロール集合そのものではなく、その書き込みが新しく加える予約ロールだけを拒否する。`current` に `system_admin` が既にあれば、`next` の同じ値は通す。

絶対集合で判定すると、通常テナントで `system_admin` を独自ロール名として使ってきた環境では、無編集の User CSV エクスポートを再適用するだけで該当行が `rejected` になる。[設計判断](../../docs/domain/identity-management/decisions.md)が置く「無編集のエクスポートを適用すると全行 `unchanged` になる」という往復不変条件と、この記録の「既存の不正な割当ては読出し時に保持する」が同時に成り立たなくなる。

差分で判定すると、新規割当ての拒否、既存値の保持、`roles` を明示して `system_admin` を外す除去経路の 3 つがどれも成立する。

### 拒否の表し方

JSON API は新しいエラーコードを導入せず、既存の 422 `invalid_role` を返し、`detail` で予約ロールであることを述べる。

コードを増やさないのは、`invalid_role` が既に「サーバーが受け付けないロール名が `roles` にある」を表す位置にあり、拒否の理由を機械可読な語彙へ足しても呼び出し元の分岐が増えるだけで、回復操作は「その名前を外す」の 1 つしかないからである。

CSV も既存の `invalid_roles` を使う。行ごとの安定コードの語彙を広げず、`column` が `roles` を指す。

### 既存データ

自動移行でロール文字列を削除しない。

テナント固有の文字列として既に使っている環境から値を無断で失わせず、[[wi-460-cross-tenant-health-control-plane-membership]] によって制御面能力は先に無効化されるためである。

既存の不正な割当ては読出し時に保持するが、新しい書込みでは追加を拒否し、更新要求が roles を省略した場合は既存値をそのまま保存する。

管理者は roles を明示的に更新して `system_admin` を除去できる。

アップグレードノートには User、Group、Agent の検出対象と、管理 API または CSV を使った除去手順を記載する。

### 規範シナリオ

既存の `REQ-IDMANAGEMENT-014` は管理 API を呼び出す側のロールを扱い、`REQ-IDMANAGEMENT-015` は Group 由来の有効ロールを扱うため、予約ロールの保存規則をどちらにも追加しない。

新しいシナリオは `REQ-IDMANAGEMENT-032` とし、制御面 Group へ `system_admin` を割り当てて制御面 User の有効ロールへ反映される成功経路を `EX-IDMANAGEMENT-032-01` に置く。

| 例 | 固定する振る舞い |
| --- | --- |
| EX-IDMANAGEMENT-032-01 | 制御面 Group への付与が所属する制御面 User の有効ロールへ乗る |
| EX-IDMANAGEMENT-032-02 | 制御面テナントの User への直接付与が通る |
| EX-IDMANAGEMENT-032-03 | 通常テナントの User への直接付与が拒否され、roles と同じ要求に含めた別項目が変わらない |
| EX-IDMANAGEMENT-032-04 | 通常テナントの Group への付与が拒否され、Group が変わらない |
| EX-IDMANAGEMENT-032-05 | 制御面テナントであっても Agent への付与が拒否され、Agent が変わらない |
| EX-IDMANAGEMENT-032-06 | 通常テナントの User CSV の行が `invalid_roles` で `rejected` となり、User が変わらない |
| EX-IDMANAGEMENT-032-07 | 通常テナントの Group CSV の行が `invalid_roles` で `rejected` となり、Group が変わらない |
| EX-IDMANAGEMENT-032-08 | 既に `system_admin` を持つ通常テナントの User の CSV 行が同じ値を再送すると `unchanged` になる |

`EX-IDMANAGEMENT-032-08` は差分判定を規範の側へ固定する。これが無いと、絶対集合で判定する実装が拒否のシナリオだけを満たしたまま往復不変条件を壊せる。

## Plan

1. IdentityManagement の決定と用語集へ予約ロールの規則を追加し、新しい規範シナリオと TypeSpec 文書コメントを作る。
2. JSON API、CSV、Agent API の全書き込み経路を一覧化し、現在は通常テナントまたは Agentへ `system_admin` を保存できることを受け入れ境界で観測する。
3. `ValidateRoleAssignment` の Unit RED を確認し、対象種別と tenant ID に基づく規則を実装する。
4. User と Group の作成、更新、CSV プレビュー、CSV 適用、および Agent の登録と更新を共通検査へ接続する。
5. JIT、SCIM、外部プロビジョニング、開発用 seed が予約ロール規則と矛盾しないことを確認する。
6. 既存データの検出対象と手動除去手順をアップグレードノートへ記載する。
7. 拒否された書き込みが roles と他の項目を一切変更しないことを確認し、検査を通す。

## Tasks

- [x] T001 [Spec] 予約ロールの決定、用語、新しい規範シナリオ `REQ-IDMANAGEMENT-032`、TypeSpec 文書コメントを追加する。
      `EX-IDMANAGEMENT-032-01` から `-08` を引用するテストが無いことを `mise run check-spec` が報告する状態を仕様側の基準線とする。`mise run check-api-compat` は 6 操作の 422 union が既に `InvalidRoleError` を含むため破壊的変更なしで通る。
- [x] T002 [Inventory] User、Group、Agent、CSV、JIT、SCIM、外部プロビジョニング、seed の roles 書き込み経路を確定する。
      結果は設計の「roles を書き込む経路の棚卸し」が持つ。要求由来のロール文字列を受け取るのは 5 経路だけである。
- [x] T003 [Acceptance] 通常テナントまたは Agent へ `system_admin` を保存できる現在の挙動を観測し、RED を確認する。
      `mise run test-go-test -- ./backend/idmanagement/handlers_http 'TestReservedRole.*'`
      規範 ID: `EX-IDMANAGEMENT-032-03`、`-04`、`-05`
- [x] T004 [App] `ValidateRoleAssignment` の Unit RED を確認してから予約ロール規則を実装する。
      `mise run test-go-test -- ./backend/idmanagement/usecases 'TestValidateRoleAssignment.*'`
- [x] T005 [App] User と Group の JSON API、CSV プレビュー、CSV 適用を共通検査へ接続する。
      `mise run test-go-test -- ./backend/idmanagement/user/usecases TestPlanUserImportRefusesTheReservedRoleButKeepsAStoredOne`
      `mise run test-go-test -- ./backend/idmanagement/group/usecases TestGroupImportPlannerRefusesTheReservedRole`
      規範 ID: `EX-IDMANAGEMENT-032-02`、`-06`、`-07`、`-08`
- [x] T006 [App] Agent の登録と更新を共通検査へ接続する。
      `mise run test-go-test -- ./backend/idmanagement/agent/usecases TestAgentReservedRoleRefusedEvenInsideTheControlPlane`
      規範 ID: `EX-IDMANAGEMENT-032-05`
- [x] T007 [Acceptance] 拒否時に roles と同時更新項目が変わらず、制御面 User と Group への正規割当ては成功することを確認する。
      `mise run test-go-changed`
      規範 ID: `EX-IDMANAGEMENT-032-01`、`-02`、`-03`、`-04`、`-05`
- [x] T008 [Docs] 既存データの検出対象と手動除去手順をアップグレードノートへ記載する。
- [x] T009 [Verify] 予約ロール検査の変異を `mise run test-go-mutation -- backend/idmanagement/usecases` で測り、仕様生成物を再生成して検査を通す。

## Verification

- `mise run test-go-race`
- `mise run test-ui-e2e`
- `mise run check-api-compat`
- `mise run check-spec`
- `mise run check-ids`
- `mise run check-work-items`
- `mise run verify`

## Risk Notes

リスクは high とする。

公開 API が以前は受理したロール文字列を拒否するため、通常テナントで `system_admin` を独自ロール名として使う既存の自動化または CSV 往復が失敗する可能性がある。

既存値を自動削除せず、roles を省略した更新は通し、明示的な除去経路を残すことで移行時のデータ消失を避ける。

CSV プレビューだけを直して適用時の再計画を直さない場合、保存済みプレビューまたは並行変更から予約ロールが入りうるため、両方を同じ検査へ通す。

拒否時に名前、メール、属性など同じ要求に含まれた別項目だけが更新されると部分適用になるため、集約保存の前に検査し、状態が一切変わらないことを確認する。

`reversibility` は irreversible とする。

既存データを自動変更しないため検査自体は外せるが、予約ロールのために割り当てた新しい規範 ID は再利用しない。

## 完了

- **Completed At**: 2026-09-19
- **Summary**:
  `mise run spec-diff` の規範差分は `added scenarios: REQ-IDMANAGEMENT-032` の 1 件である。
  `system_admin` が予約ロールになり、新しく割り当てられる対象は制御面テナントの `User` と `Group` だけになった。`Agent` はどのテナントでも割り当てられない。
  判定は `backend/idmanagement/usecases/role_assignment.go` の `ValidateRoleAssignment` が持ち、`User` と `Group` の作成と更新、`Agent` の登録と更新、`User` CSV と `Group` CSV の計画器の 8 箇所が保存前に呼ぶ。CSV の適用も同じ計画器を通るため、保存済みプレビューから予約ロールが入る経路は残らない。
  判定は保存後のロール集合ではなく書き込みが新しく加える予約ロールだけを見るため、既存値の再送、無編集エクスポートの再適用、`roles` を明示した除去のいずれも従来どおり通る。
  公開契約は増えていない。6 操作の 422 union は既に `InvalidRoleError` を含んでおり、拒否は既存の `invalid_role` と CSV の `invalid_roles` を使う。`mise run check-api-compat` は破壊的変更なしで通った。
- **Primary Use Case Evidence**:
  - id: control-plane-group-grants-system-admin
    unit_red: >-
      実装前に RED は観測していない。この作業項目は許可を広げるのではなく過剰な許可を狭める修正であり、
      制御面 Group への `system_admin` 付与は変更前から成功していたため、主要ユースケースの成功経路には
      失敗する時点が存在しない。同じ単体境界で実装前に観測した RED は、
      `TestValidateRoleAssignmentDecidesByTargetKindAndTenant` ほか 4 本が
      `undefined` シンボル 4 種 (`ValidateRoleAssignment`、`RoleAssignmentTarget`、
      `ReservedRoleSystemAdmin`、`ErrReservedRole`) でビルド失敗したことである。
    e2e_red: >-
      同じ理由で `TestControlPlaneGroupGrantsSystemAdminToItsMembers` は実装前から GREEN だった。
      同一ファイルの拒否側で実装前に観測した RED は、
      `TestReservedRoleRefusedOutsideTheControlPlaneChangesNoUser` が `roles` に `system_admin` を
      載せた status=200 を返し、`TestReservedRoleRefusedOutsideTheControlPlaneCreatesNoGroup` が
      status=201 で `acme` テナントに `system_admin` を持つ Group を作り、
      `TestReservedRoleRefusedForAgentsInEveryTenant` が両テナントの更新で status=200、
      制御面テナントの登録で status=201 を返したことである。
    unit_fault_injection: >-
      `ValidateRoleAssignment` から `tenantID` と `DefaultTenantID` を比べる分岐を落とし、
      予約ロールの新規付与を全テナントで拒否させると、
      `TestCreateGroupKeepsTheReservedRoleInsideTheControlPlane` を含む
      `backend/idmanagement/group/usecases` と `backend/idmanagement/usecases` が FAIL した。
    e2e_fault_injection: >-
      `CreateGroup` が共通検査へ渡す対象種別を `RoleTargetGroup` から `RoleTargetAgent` へ
      差し替えると、`TestControlPlaneGroupGrantsSystemAdminToItsMembers` が制御面 Group の作成で
      422 を受けて FAIL し、`backend/idmanagement/handlers_http` と
      `backend/idmanagement/group/usecases` が FAIL した。
- **Acceptance RED Evidence**:
  - **Test**: `mise run test-go-test -- ./backend/idmanagement/handlers_http 'TestReservedRole.*'`
  - **Requirement**: REQ-IDMANAGEMENT-032
  - **Observed Failure**: `acme` テナントの User 更新が `status=200 body={... "roles":["system_admin"] ...}`、`acme` テナントの Group 作成が `status=201 body={... "tenant_id":"acme","roles":["system_admin"] ...}`、制御面テナントと `acme` テナントの Agent 更新が `status=200`、制御面テナントの Agent 登録が `status=201`。いずれも want 422。
  - **Detection Reason**: 対象の例は `EX-IDMANAGEMENT-032-03`、`-04`、`-05`、`-09`、`-10` である。判定は HTTP 応答だけでなく、拒否後に保存層から対象を読み直した `roles`、同じ要求に相乗りさせた表示名または説明、`updated_at`、および `GroupCreated` / `AgentUpdated` / `UserCreated` の不在を見る。422 を書いてから保存する実装と、`roles` だけを弾いて別項目を部分適用する実装のどちらもここで落ちる。各テストは予約ロール以外の名前で同じ要求が通る対照を持つため、テナントごと書き込みが不能な構成と区別できる。
- **Unit RED Evidence**:
  - **Test**: `mise run test-go-test -- ./backend/idmanagement/usecases 'TestValidateRoleAssignment.*'`
  - **Requirement**: REQ-IDMANAGEMENT-032
  - **Observed Failure**: `undefined: RoleAssignmentTarget`、`undefined: ValidateRoleAssignment`、`undefined: ReservedRoleSystemAdmin`、`undefined: ErrReservedRole` でビルドが失敗した。
  - **Detection Reason**: 対象の例は差分判定の `EX-IDMANAGEMENT-032-08` を含む。表が対象種別 3 種とテナント 2 種の 6 行すべてを持つため、テナントを見ない実装と対象種別を見ない実装のどちらも 1 行では通れない。差分判定の表は既存値の再送、追加、明示的除去、新規付与を分けており、絶対集合で判定する実装は「既存値の再送」の行で落ちる。予約ロール以外を全対象で通す表があるため、語彙を閉じた列挙にする実装も落ちる。
- **Change-Resistance Results**:
  `mise run test-go-mutation -- backend/idmanagement/usecases` は 71 変異を発見し、54 を試験して 45 killed / 9 lived (efficacy 83.33%)。
  変更した `role_assignment.go` に対して変異器が作れたのは条件反転 2 件 (`==` → `!=` 48 行目、`!=` → `==` 51 行目) だけで、どちらも killed。生存 9 件はすべて未変更の `data_export.go` にあり、この作業項目が触っていない算術と条件境界である。
  変異器がこの関数から 2 件しか作れないのは、判定の骨格が `slices.Contains` の真偽値と早期 return であり、書き換えられる演算子トークンがほとんど無いからである。これが方法の限界であり、「分岐を落とす」「配線を外す」「対象種別を取り違える」種類は手で注入した。
  13 件の注入とその結果:

  | 注入 | 検出したテスト |
  | --- | --- |
  | 差分の分岐 (`slices.Contains(current, …)`) を落とす | `backend/idmanagement/usecases` |
  | 予約ロール以外の早期 return を落とす | `backend/idmanagement/handlers_http`、`backend/idmanagement/usecases` |
  | `target == RoleTargetAgent` の分岐を落とす | `backend/idmanagement/handlers_http`、`backend/idmanagement/usecases` |
  | `tenantID != DefaultTenantID` の分岐を落とす | `backend/idmanagement/handlers_http`、`backend/idmanagement/usecases` |
  | `CreateUser` の配線を外す | `backend/idmanagement/handlers_http` |
  | `UpdateUser` の配線を外す | `backend/idmanagement/handlers_http` |
  | `CreateGroup` の配線を外す | `backend/idmanagement/handlers_http`、`backend/idmanagement/group/usecases` |
  | `UpdateGroup` の配線を外す | `backend/idmanagement/handlers_http` |
  | `RegisterAgent` の配線を外す | `backend/idmanagement/handlers_http`、`backend/idmanagement/agent/usecases` |
  | `UpdateAgent` の配線を外す | `backend/idmanagement/handlers_http`、`backend/idmanagement/agent/usecases` |
  | User CSV 計画器の配線を外す | `backend/idmanagement/user/usecases` |
  | Group CSV 計画器の配線を外す | `backend/idmanagement/group/usecases` |
  | `CreateGroup` が対象種別を `RoleTargetAgent` として渡す | `backend/idmanagement/handlers_http`、`backend/idmanagement/group/usecases` |

  最初の実行では `CreateUser` と `UpdateGroup` の配線を外した 2 件が生存した。通常テナントでの User の作成と Group の更新を観測するテストが無かったためである。
  この 2 経路は対象範囲が挙げる「User と Group の作成、更新」に含まれるので、規範側へ `EX-IDMANAGEMENT-032-09` と `EX-IDMANAGEMENT-032-10` を追加し、`TestReservedRoleRefusedOutsideTheControlPlaneCreatesNoUser` と `TestReservedRoleRefusedOutsideTheControlPlaneChangesNoGroup` を足した。再実行で 13 件すべてが検出された。
- **Verification Results**:
  - `mise run lint-go` - passed
  - `mise run check-api-compat` - passed (破壊的変更なし)
  - `mise run check-spec` - passed
  - `mise run check-work-items` - passed
  - `mise run verify` - passed
  - `mise run test-ui-e2e` - passed (28 tests, 6 files)

`mise run test-go-test` は名前を `^…$` で囲むため、接頭辞で複数本を選ぶときは末尾に `.*` を付ける。

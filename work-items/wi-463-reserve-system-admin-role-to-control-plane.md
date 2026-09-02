---
status: pending
authors: [tn]
risk: high
reversibility: irreversible
created_at: 2026-09-03
change_kind: bugfix
priority: p1
depends_on: [wi-460-cross-tenant-health-control-plane-membership]
affected_spec:
  - { path: docs/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-004 }
  - { path: docs/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-014 }
  - { path: docs/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-015 }
  - { path: docs/contexts/identity-management/scenarios.md, requirement: REQ-IDMANAGEMENT-026 }
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

Agent は `docs/authorization.md` でシステム運用者とは異なる主体として定義され、制御面の管理 API は User と ApiToken だけを受け入れるため、`system_admin` を割り当てない。

通常テナントで `catalog:read` など任意のロール名を使う現在のモデルは維持し、予約する文字列は `system_admin` だけに限定する。

### 検査の配置

`NormalizeRoles` は Agent を含む複数の対象で使われ、対象種別と tenant ID を受け取らないため、予約ロールの判断を加えない。

字句の正規化と製品上の割当て規則を分け、正規化後のロール集合、対象種別、tenant ID を受け取る `ValidateRoleAssignment` を IdentityManagement のユースケース境界に置く。

HTTP ハンドラーだけで検査すると CSV 適用と内部の再計画経路が抜けるため、User、Group、Agent の各書き込みユースケースと CSV 計画器が保存前に共通検査を呼ぶ。

CSV はプレビュー時に対象行を安定した `invalid_roles` で拒否し、適用時にも保存済みプレビュー結果を信頼せず、再計画で同じ検査を行う。

### 既存データ

自動移行でロール文字列を削除しない。

テナント固有の文字列として既に使っている環境から値を無断で失わせず、[[wi-460-cross-tenant-health-control-plane-membership]] によって制御面能力は先に無効化されるためである。

既存の不正な割当ては読出し時に保持するが、新しい書込みでは追加を拒否し、更新要求が roles を省略した場合は既存値をそのまま保存する。

管理者は roles を明示的に更新して `system_admin` を除去できる。

アップグレードノートには User、Group、Agent の検出対象と、管理 API または CSV を使った除去手順を記載する。

### 規範シナリオ

既存の `REQ-IDMANAGEMENT-014` は管理 API を呼び出す側のロールを扱い、`REQ-IDMANAGEMENT-015` は Group 由来の有効ロールを扱うため、予約ロールの保存規則をどちらにも追加しない。

新しいシナリオは、制御面 Group へ `system_admin` を割り当てて制御面 User の有効ロールへ反映される成功経路を中心にする。

通常テナントの User と Group、Agent、User CSV、Group CSV の拒否を `ALT` とし、拒否時に対象の roles と他の同時更新項目が変わらないことを確認する。

## Plan

1. IdentityManagement の決定と用語集へ予約ロールの規則を追加し、新しい規範シナリオと TypeSpec 文書コメントを作る。
2. JSON API、CSV、Agent API の全書き込み経路を一覧化し、現在は通常テナントまたは Agentへ `system_admin` を保存できることを受け入れ境界で観測する。
3. `ValidateRoleAssignment` の Unit RED を確認し、対象種別と tenant ID に基づく規則を実装する。
4. User と Group の作成、更新、CSV プレビュー、CSV 適用、および Agent の登録と更新を共通検査へ接続する。
5. JIT、SCIM、外部プロビジョニング、開発用 seed が予約ロール規則と矛盾しないことを確認する。
6. 既存データの検出対象と手動除去手順をアップグレードノートへ記載する。
7. 拒否された書き込みが roles と他の項目を一切変更しないことを確認し、検査を通す。

## Tasks

- [ ] T001 [Spec] 予約ロールの決定、用語、新しい規範シナリオ、TypeSpec 文書コメントを追加する。
- [ ] T002 [Inventory] User、Group、Agent、CSV、JIT、SCIM、外部プロビジョニング、seed の roles 書き込み経路を確定する。
- [ ] T003 [Acceptance] 通常テナントまたは Agent へ `system_admin` を保存できる現在の挙動を観測し、RED を確認する。
- [ ] T004 [App] `ValidateRoleAssignment` の Unit RED を確認してから予約ロール規則を実装する。
- [ ] T005 [App] User と Group の JSON API、CSV プレビュー、CSV 適用を共通検査へ接続する。
- [ ] T006 [App] Agent の登録と更新を共通検査へ接続する。
- [ ] T007 [Acceptance] 拒否時に roles と同時更新項目が変わらず、制御面 User と Group への正規割当ては成功することを確認する。
- [ ] T008 [Docs] 既存データの検出対象と手動除去手順をアップグレードノートへ記載する。
- [ ] T009 [Verify] 仕様生成物を再生成し、検査を通す。

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

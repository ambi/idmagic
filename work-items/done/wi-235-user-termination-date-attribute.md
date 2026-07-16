---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-07-17
depends_on: []
---

# 退職日 (termination_date) を組み込みユーザー属性として追加する

## Motivation
`spec.User` の組み込み属性カタログ (wi-19 / ADR-039 / ADR-040) には入社日
`hire_date` (`AttributeTypeDate`) があるが、対になるはずの退職日が無い。
Okta / Workday 系の IdP 連携でも `hireDate` と対をなす離職日相当の属性
(Entra ID の `employeeLeaveDateTime` 等) は一般的であり、入社日だけを
持つ非対称な状態になっている。

退職は現状 `UserStatus` の状態変更 (Active → Disabled 等) としてのみ
表現され、「いつ退職する予定/したか」という日付情報を構造化して保持
する手段が無い。この日付は [[wi-225-lifecycle-workflow-date-based-triggers]]
が導入する `date_attribute_offset` trigger (退職日の N 日後にアクセス
剥奪する、等) の対象属性としても必要になる。

退職日はテナント固有の値ではなく、入社日と同様にほぼ全テナントが
必要とする組織属性であるため、テナント任意定義のカスタム属性
(`TenantAttributeSchema`) ではなく、`hire_date` と同じ組み込みカタログ
に対称な形で追加する。

## Scope
- `spec/contexts/identity-management.yaml` に `termination_date`
  (`AttributeTypeDate`) を、`hire_date` と同じ SCIM enterprise:User
  拡張相当の組織属性として追記する (該当箇所があれば)。
- `backend/identitymanagement/domain/attributes.go` の `builtinDefs` に
  `org("termination_date", AttributeTypeDate)` を追加し、
  `builtinAttributeLabels` に日本語ラベル「退職日」を追加する。
  `hire_date` と同じく `EditableByUser: false` /
  `AttrVisibilitySelfReadable` とする。

## Out of Scope
- 退職日到来時にアクセス剥奪等を自動実行する仕組み。これは
  [[wi-225-lifecycle-workflow-date-based-triggers]] のスコープ。
- 異動日相当の属性・予約変更メカニズム。属性ではなく別の仕組みが
  適切という結論に至っており、別 WI で扱う。
- 退職日変更の承認フロー。wi-19 の踏襲で admin のフラットな変更に
  留める。

## Plan
- `hire_date` の実装パターン (org() ヘルパー、AttributeTypeDate、
  admin-only 編集) をそのまま踏襲する。新しい属性型や visibility
  ルールは導入しない。

## Tasks
- [x] T001 [SCL] `identity-management.yaml` に `termination_date` を
      追記する (対象箇所があれば)。→ 調査の結果、`hire_date` を含め
      組み込み属性カタログの個別キーは SCL に列挙されず (`UserAttributeDef`
      という型のみ SCL 上で定義され、実データは Go の `attributes.go` が
      唯一の正本)、対象箇所が無いため SCL 変更なしで妥当と判断した。
- [x] T002 [App] `attributes.go` の `builtinDefs` /
      `builtinAttributeLabels` に `termination_date` / 「退職日」を追加する。
- [x] T003 [Verify] admin UI の属性編集画面で退職日を設定・表示できる
      ことを確認する。→ ブラウザ自動化ツールが無く実ブラウザでの目視確認は
      未実施。代わりに (a) 属性の検証・監査・claim 露出ロジックが
      `BuiltinUserAttributeDefs()` 駆動でキー名に対する分岐が存在しない
      ことをコード確認、(b) フロントエンドが属性一覧を `hire_date` 等の
      固定リストではなく backend の attribute defs から動的に描画している
      ことを確認、で代替した。

## Verification
- `just yaml-check`
- `just test-go`
- `just verify-go`
- 手動: admin が `/admin/users/{sub}` の detail で退職日を設定 →
  リロード後に反映され、`hire_date` と同様に監査イベントに現れる。
- 手動: self-service 経路 (`/api/account/profile`) では退職日が
  編集できないこと (admin only) を確認する。

## Risk Notes
`hire_date` と全く同じパターンの追加であり、リスクは低い。既存の
属性 diff / 監査 / claim 露出ロジックは属性カタログ駆動のため、
新しいコード分岐を追加する必要はない想定。

## Completion
- **Completed At**: 2026-07-17
- **Summary**:
  `backend/identitymanagement/domain/attributes.go` の組み込み属性カタログに
  `termination_date` (`AttributeTypeDate`、`hire_date` と同じ `org()` 定義:
  `EditableByUser: false` / `AttrVisibilitySelfReadable`) と日本語ラベル
  「退職日」を追加した。SCL (`spec/scl.yaml` / `spec/contexts/*.yaml`) には
  個別の組み込み属性キーを列挙する箇所が存在しない (`UserAttributeDef` は
  型としてのみ SCL に現れ、実データは Go 側が正本) ため、SCL 変更は不要と
  判断した。フロントエンドは attribute defs を動的に取得して描画するため
  UI コード変更も不要。
- **Verification Results**:
  - `just yaml-check` - passed
  - `just test-go` (`just verify-go` 経由) - passed
  - `just verify-go` (lint + race test) - passed
  - 手動 (admin UI での退職日設定・自己編集不可の確認): 未実施。ブラウザ
    自動化ツールが利用できず、実ブラウザでの目視確認は次回対応者へ持ち越し。
    代替としてコードレベルで属性検証・監査・claim 露出ロジックが
    キー名非依存(カタログ駆動)であることと、UI が attribute defs を
    動的描画していることを確認済み。
- **Affected Guarantees State**:
  - tenant isolation: 影響なし (既存の attribute def 検証経路をそのまま利用)。
  - admin RBAC: `termination_date` は `hire_date` と同じく `EditableByUser: false`
    のため admin のみ編集可能、self-service (`/api/account/profile`) からは
    既存の visibility チェックにより拒否される (コードパス上は `hire_date` と
    同一)。
  - OIDC ID Token / UserInfo: `termination_date` は `org()` 定義 (claim 非露出)
    のため既存 RP のクレーム集合に変化なし。
  - 監査ログ: 既存の attribute diff / changedFields 経路をそのまま使うため
    新規イベント型は不要。
  - SCL coherence: SCL 側の変更なし。`just yaml-check` で確認済み。
  - backwards compatibility: 追加のみで既存フィールド・APIレスポンス形状に
    破壊的変更なし。

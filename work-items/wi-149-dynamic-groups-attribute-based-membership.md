---
depends_on: []
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-10
---

# 属性ベースの動的グループ所属を導入する

## Motivation
現状の Group は管理者が明示的に membership を追加・削除する静的なロール束であり、
部署、雇用区分、勤務地、manager、メールドメインなどのユーザー属性に応じた自動所属を
表現できない。Okta group rules、Microsoft Entra dynamic groups、Google Cloud
Identity dynamic groups のような機能がないと、入社・異動・退職に伴うアクセス変更を
手作業で追従する必要があり、過剰権限と運用漏れが起きやすい。

本 WI は、テナント内の User 属性に基づいて Group membership を計算・同期する
DynamicGroup を導入し、属性変更時にアプリ割当や有効ロールが自動更新される基盤を作る。

## Scope
- **scl**:
  - `IdentityManagement` に DynamicGroup / DynamicGroupRule / DynamicMembershipEvaluation を追加する。
  - GroupMembership を手動所属と動的所属で区別し、動的所属は評価結果からのみ変更できることを明示する。
  - User 属性変更、Group rule 変更、ユーザー lifecycle 変更時の再評価 events / scenarios を追加する。
  - `Tenancy` の TenantUserAttributeSchema と組み込み属性を rule の参照元として明示する。
- **go**:
  - 動的グループ rule の parser / validator / evaluator、membership 同期 usecase、memory / postgres adapter を追加する。
  - User 属性更新と lifecycle 遷移から再評価を呼び出し、effective_roles と application assignment 判定に反映する。
- **http**:
  - DynamicGroup rule の作成・更新・有効化・無効化・評価プレビュー API を追加する。
- **ui**:
  - Group detail に rule 設定、プレビュー、評価結果、手動所属との差分表示を追加する。
- **documentation**:
  - README に rule の語彙、評価タイミング、手動所属との違いを記録する。

## Out of Scope
- 外部ディレクトリ同期そのもの。
- 汎用ポリシー言語 (Rego / Cedar 等) の導入。
- 大規模リアルタイムストリーム評価や分散キャッシュ最適化。
- cross-tenant の属性参照。

## Plan
- まず安全な構造化 rule DSL を SCL で定義し、参照可能な属性 key / operator / literal 型を TenantUserAttributeSchema から検証する。
- 動的所属は通常の GroupMember と同じ effective_roles に寄与するが、管理 UI では手動所属と区別して表示し、管理者が直接削除できないようにする。
- 評価は同期 usecase から始め、重い一括再評価は `wi-126-async-job-runner` 完了後に job 化できる境界を残す。
- 既定は fail-closed とし、rule が不正・参照属性が削除済み・評価不能の場合は新規動的所属を付与しない。

## Tasks
- [ ] T001 [SCL] DynamicGroup、rule、membership 種別、評価 events / constraints/contracts を追加する。
- [ ] T002 [Decision] rule DSL、評価タイミング、手動所属との責務分離を ADR に記録する。
- [ ] T003 [App] rule 検証・評価・membership 同期を domain/usecase/persistence に実装する。
- [ ] T004 [HTTP] 管理 API と評価プレビュー API を追加する。
- [ ] T005 [UI] Group detail に動的 rule と評価結果の管理 UI を追加する。
- [ ] T006 [Verify] SCL、Go、UI、手動シナリオを検証する。

## Verification
- `just yaml-check`
- `just check-ids`
- `just test-go`
- `just verify-ui`
- 手動: department が `Engineering` のユーザーだけを対象にする dynamic group を作成し、属性変更で所属と application assignment が更新されることを確認する。
- 手動: rule が参照する属性を削除または無効化した場合、評価不能として新規所属が付与されないことを確認する。

## Risk Notes
動的所属はアプリ割当とロールに直結するため、評価誤りは過剰権限になる。rule DSL を自由文字列にせず、参照属性と operator を構造化して検証する。評価不能時は拒否側に倒し、手動所属と動的所属の更新経路を分離して監査イベントで追跡する。

---
status: pending
authors: ["tn"]
risk: high
created_at: 2026-07-12
depends_on: [wi-192-tenant-branding-logo-display-regression, wi-193-simplify-tenant-branding-color-validation]
---

# テナントブランディングを管理コンソール全体のテーマへ適用する

## Motivation
テナント管理者が設定したロゴ・ブランドカラーが利用者向け hosted UI にしか反映されず、日常的に利用する管理コンソールとのブランド体験が分断されている。テナント管理コンソールを同じブランドで一貫して操作できるようにする。

## Scope
- `spec/scl.yaml` の `TenantBranding` の適用先、GetTenantBranding の consumer、管理コンソールテーマ適用 scenario と fail-open invariant。
- tenant admin console の shell、ナビゲーション、ページ見出し、主要 action、選択・focus・link・status の色を branding の primary / accent color から導く semantic theme token へ移行する。
- tenant branding の logo / product name を管理コンソール chrome に表示する。
- branding 未設定・取得失敗・低コントラスト色でも管理コンソールを操作可能にする自動 foreground / neutral fallback と、主要画面の視覚回帰テスト。

## Out of Scope
- system admin console (`/system`) と運用者専用画面へのテナントテーマ適用。
- 任意 CSS / HTML、テナントごとのレイアウト・フォント・コンポーネント構造変更。
- 個々の業務データに含まれる既存の意味色（成功・警告・危険）のブランド色への置換。

## Plan
- `wi-192` でロゴ表示経路を安定化し、`wi-193` で色コントラストの保存制約を廃止した後に着手する。
- AdminLayout の最上位で branding を取得し、primary / accent から `--admin-theme-*` の semantic token（surface、emphasis、interactive、focus ring、on-color）を導出する。直接の `blue-*` 一括置換は避け、共通 UI 部品と shell を入口に置換する。
- 任意の色を受理しても可読性を失わないよう、彩色面に載せる前景色は相対輝度から黒 / 白を自動選択し、淡色のリンク本文には中立色または派生した濃色 token を用いる。
- `GET /api/branding` の失敗または未設定時は IdMagic 既定テーマを同期的に表示し、管理操作を阻害しない。tenant-scoped console だけに適用し、system console は既定テーマを固定する。

## Tasks
- [ ] T001 [SCL] 管理コンソールを TenantBranding の適用先として契約・scenario・fail-open invariant を更新する。
- [ ] T002 [Theme] branding から安全な semantic token を導出する theme provider と共通 UI token を実装する。
- [ ] T003 [Admin UI] shell、ナビゲーション、主要 action、選択・focus・link 状態、主要ページを token ベースへ移行し、ロゴ / 製品名を chrome に反映する。
- [ ] T004 [Isolation] tenant realm 切替、未設定・取得失敗、system console 非適用をテストする。
- [ ] T005 [Visual/Verify] 代表管理画面の visual / UI 回帰とキーボード focus を検証する。

## Verification
- `just scl-render`
- `just test-ui-unit`
- `just test-ui-e2e`
- `just verify-ui`
- 手動: 二つの realm に異なるロゴ・配色を設定し、それぞれの tenant admin console にだけ反映され、`/system` は IdMagic 既定テーマのままであることを確認する。

## Risk Notes
管理コンソールは多数の固定色 class を持つため、全置換は表示崩れと意味色の喪失を招く。semantic token を共通部品から導入し、主要導線を visual regression で確認する。色コントラストを入力制約にしない一方、導出した foreground と focus ring は常に操作可能な見え方を維持する。

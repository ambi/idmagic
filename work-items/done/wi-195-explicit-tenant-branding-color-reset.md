---
status: completed
authors: ["tn"]
risk: low
created_at: 2026-07-12
depends_on: []
---

# テナントブランディングの色を既定値へ明示的に戻せるようにする

## Motivation
空の色値を保存すればシステム既定へ戻る実装はあるが、color picker は既定の濃色を表示し、画面上にリセット操作もない。そのため管理者が元に戻す方法を発見できず、意図しないブランド設定を残す。

## Scope
- `spec/contexts/tenancy.yaml` の `models.TenantBranding`、`models.TenantBrandingUpdateRequest`、`interfaces.UpdateTenantBranding`、`invariants.TenantBrandingSafeTokens`、branding `scenarios`、`user_experience.screens.AdminSettings`。
- `frontend/src/features/admin-settings/BrandingTab.tsx` の primary / accent color 入力に、未設定（IdMagic 既定）へ戻す明示的な操作と現在値の説明を追加する。
- リセット後に空文字列が UpdateTenantBranding へ送られ、hosted UI が既定色へフォールバックする UI 回帰テスト。

## Out of Scope
- backend の UpdateTenantBranding 契約変更（空文字列による未設定化は現行で対応済み）。
- ロゴ、favicon、製品名、リンクの一括リセット。

## Plan
- 色ごとに「既定に戻す」操作を置き、空値時は色入力の見かけの値ではなく未設定状態であることを表示する。
- 保存前の変更と保存後の public branding 応答をテストし、既存の空値セマンティクスを利用する。

## Tasks
- [x] T001 [UI] primary / accent color の明示的なリセット操作と未設定表示を追加する。
- [x] T002 [Test] リセット、保存 payload、再読み込み時の表示をテストする。
- [x] T003 [Verify] UI 検証を実行する。

## Verification
- `just test-ui-unit`
- `just verify-ui`
- 手動: 色を保存後、それぞれを「既定に戻す」で空にして保存し、login 画面が既定色へ戻ることを確認する。

## Risk Notes
UI のみの変更だが、空文字列が「未設定」以外に解釈されないことを既存 API 契約とテストで確認する。

## Completion

- **Completed At**: 2026-07-12
- **Summary**:
  primary / accent color ごとに、現在値または未設定（IdMagic 既定）の状態を表示し、設定済みの色を個別に既定へ戻せるようにした。
  空文字列を送る既存の UpdateTenantBranding 契約を SCL に明記し、保存後も未設定状態を表示する。
- **Affected Guarantees State**:
  色は引き続き `#rrggbb` 形式または未設定だけを受け付ける。各色のリセットは空文字列として送信され、hosted UI では CSS 既定色へフォールバックする。
- **Verification Results**:
  - `just yaml-check` — passed
  - `just scl-render` — passed
  - `just test-ui-unit` — passed (292 tests)
  - `just verify-ui` — passed
  - `just verify` — passed
- **Evidence**:
  - 実行日: 2026-07-12
  - 実行環境: ローカル開発環境 (macOS)
  - 実行主体: Codex (GPT-5)
  - 対象ソース版: main (コミット前作業ツリー)
  - 保存先: 外部成果物なし。BrandingTab のコンポーネントテストでリセット、PUT payload、保存応答後の未設定表示を確認。

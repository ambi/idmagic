---
status: completed
authors: ["tn"]
risk: medium
created_at: 2026-07-12
depends_on: []
---

# テナントブランディングの色コントラスト制約を廃止する

## Motivation
現行実装は primary / accent color を相互比較するのではなく、各色が白背景に対して WCAG AA 4.5:1 を満たすことを更新 API で強制する。しかし管理画面にその条件や判定結果がなく、正しいブランドカラーを設定できない利用者が発生する。配色の責任をテナントへ委ね、設定可能性を優先する。

## Scope
- `spec/contexts/tenancy.yaml` の `models.TenantBranding`、`interfaces.UpdateTenantBranding`、`invariants.TenantBrandingSafeTokens`、branding scenarios、`user_experience.screens.AdminSettings`。
- domain / database / HTTP / UI にある WCAG AA コントラスト比の必須検証と文言の削除。
- `#rrggbb` 形式のみを維持する色入力検証と、低コントラスト色を保存できる回帰テスト。

## Out of Scope
- arbitrary CSS、alpha channel、色以外のテーマ token の追加。
- hosted UI の文字色・背景色を自動算出するテーマエンジン。

## Plan
- 色の構文検証は残し、コントラスト比を保存拒否の条件から外す。
- アクセシビリティ上の注意は UI の補助文言として任意に提示できるが、保存可否には用いない。

## Tasks
- [x] T001 [SCL] 色の invariant と UpdateTenantBranding の契約を更新する。
- [x] T002 [Domain/HTTP] コントラスト比の拒否と関連エラー文言を削除する。
- [x] T003 [UI] 制約を前提とした説明を削除し、形式エラーだけを表示する。
- [x] T004 [Verify] 低コントラストを含む有効な hex 色の保存・表示をテストする。

## Verification
- `just scl-render`
- `just test-go`
- `just test-ui-unit`
- `just verify-ui`

## Risk Notes
低コントラストの設定により hosted UI の可読性が下がり得る。これは明示的なテナント選択として扱い、形式制約と任意のアクセシビリティ注意で誤入力を抑える。

## Completion

- **Completed At**: 2026-07-12
- **Summary**:
  primary_color / accent_color は `#rrggbb` 形式だけを検証し、低コントラストの色を保存・取得できるようにした。
  UI と HTTP のエラー文言を形式検証に合わせ、ADR-097 でテナントが可読性を判断する方針を記録した。
- **Affected Guarantees State**:
  任意 CSS・HTML・JS は引き続き受理せず、色は `#rrggbb`、リンクは https、テキストと画像は既存の安全制約を維持する。低コントラストは保存拒否の理由にならない。
- **Verification Results**:
  - `just yaml-check` — passed
  - `just scl-render` — passed
  - `just test-go` — passed
  - `just test-ui-unit` — passed (289 tests)
  - `just verify-ui` — passed
- **Evidence**:
  - 実行日: 2026-07-12
  - 実行環境: ローカル開発環境 (macOS)
  - 実行主体: Codex (GPT-5)
  - 対象ソース版: main (コミット前作業ツリー)
  - 保存先: 外部成果物なし。上記検証結果を本記録に要約。

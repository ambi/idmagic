---
id: wi-141-consent-and-connected-app-friendly-names
title: 同意・連携済みアプリ画面で client_id を人間可読名で表示する
created_at: 2026-07-05
authors: [tn]
status: pending
risk: low
---

# Motivation
[idp-ADR-084](../decisions/idp-ADR-084-postgres-column-type-policy.md)（wi-127）で
`clients.client_id` を UUID 型に閉じた結果、client_id をそのまま生表示している画面では、
エンドユーザー / 管理者に `demo-client` のような可読名の代わりに
`00000000-0000-4000-8000-000000000021` のような UUID が見えるようになった。機能は正常だが
UX が劣化している。

該当箇所（現状はいずれも raw id を font-mono で表示）:
- `ui/src/features/account/AccountApplicationsPage.tsx`: エンドユーザーの「連携済みアプリ」
  一覧・取り消しメッセージが `consent.client_id` を表示（`{consent.client_id}` /
  `${clientId} へのアクセスを取り消しました。`）。
- `ui/src/features/admin-consents/AdminConsentsPage.tsx`: 管理者の同意一覧・詳細・確認
  ダイアログが `client_id`（および `user_id`）を raw 表示。

同意 API 応答（`admin_consent_handler.go` / account consents）は現状 `client_id` のみを持ち、
`client_name` やカタログ上のアプリ名を含まない。

# Scope
- **implementation**:
  - 同意 / 連携済みアプリの応答に、client_id を解決した表示名（OAuth2Client の
    `client_name`、無ければ Application カタログの名前、いずれも無ければ client_id に
    フォールバック）を含める。usecase / HTTP handler / shared spec 型を最小限拡張する。
  - 管理者側で user_id を表示している箇所も、可能なら `preferred_username` 等の可読名を
    併記する（UUID 化で同様に読みにくくなったため。副次対応、任意）。
- **UI**:
  - `AccountApplicationsPage` / `AdminConsentsPage` を、主表示は可読名、UUID は補助情報
    （tooltip / セカンダリ表記）に変更する。取り消し確認メッセージも可読名にする。
- **spec**:
  - 表示名フィールドを公開 contract に足す場合は SCL-first で `spec/scl.yaml` を最小限更新し
    derived artifacts を再生成する。
- **test**:
  - フォールバック順（client_name → app 名 → client_id）の単体テストと、e2e の該当アサーション
    （現状は UUID を待つ `ui-scenario-actions.spec.ts`）を可読名待ちに戻す。

# Out of Scope
- client_id を UUID から可読値へ戻すこと（ADR-084 の方針は維持し、表示層のみで解決する）。
- 同意・カタログのデータモデル自体の再設計。

# Verification
- `just yaml-check-work-items`
- `just check-ids`
- `just yaml-check`（SCL を変更した場合）
- `just verify`
- 手動確認: 連携済みアプリ / 管理同意画面で client が可読名で表示され、UUID は補助表記に
  留まる。表示名が無い client では client_id にフォールバックする。

# Risk Notes
表示層と応答の拡張が中心で波及は小さい。ただし表示名の解決順（client_name / catalog / fallback）
を一貫させないと画面ごとに表示が食い違うため、解決ロジックは 1 箇所に集約する。

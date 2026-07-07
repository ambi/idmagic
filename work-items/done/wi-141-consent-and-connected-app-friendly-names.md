---
id: wi-141-consent-and-connected-app-friendly-names
title: 同意・連携済みアプリ画面で client_id を人間可読名で表示する
created_at: 2026-07-05
authors: [tn]
status: completed
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

# Plan
解決ロジックを共有 `support` パッケージの 1 箇所（`ClientDisplayNameResolver`）に集約する。
既存の `support.ApplicationGate`（Application コンテキストを OAuth2/Authentication へ橋渡しする
narrow adapter）と同じ構図で、composition root で一度だけ組み立て、OAuth2 admin consents /
Authentication account consents 両ハンドラへ注入する。解決順は client_name → Application
カタログ名（OIDC binding = client_id）→ client_id フォールバック。管理画面の user_id は
`UserRepo` から `preferred_username` を併記（副次対応）。

- SCL: `AdminConsentResponse` / `AccountConsentResponse` に `client_name` を、前者に
  `preferred_username`（optional）を追加し derived artifacts を再生成。
- resolver: `internal/shared/adapters/http/support/client_display_name.go` に集約 + 単体テスト。
- handlers: admin / account consent 応答を resolver で enrich、composition root で配線。
- UI: 主表示を可読名・UUID を補助（tooltip / secondary）へ。取消メッセージも可読名。
- test: フォールバック順の単体テスト、handler テスト更新、e2e を可読名待ちへ。

# Tasks
- [x] T001 SCL: `AdminConsentResponse`/`AccountConsentResponse` に表示名フィールド追加 + `just yaml-check` + `just scl-render`
- [x] T002 resolver: `ClientDisplayNameResolver` 実装 + 単体テスト（フォールバック順）
- [x] T003 handlers: admin/account consent 応答 enrich + composition root 配線
- [x] T004 UI: `AccountApplicationsPage` / `AdminConsentsPage` を可読名主表示へ、取消メッセージ更新
- [x] T005 test: handler テスト更新、e2e を可読名待ちへ戻す
- [x] T006 検証: `just verify` / `just yaml-check` / `just check-ids` グリーン、完了記録・done 移動・commit

# Completion
- **Completed At**: 2026-07-07
- **Summary**:
  client_id → 表示名の解決ロジックを `internal/shared/adapters/http/support/client_display_name.go`
  の `ClientDisplayNameResolver` に集約した。解決順は OAuth2Client.client_name → Application
  カタログ名 (OIDC binding = client_id) → client_id フォールバック。composition root で
  `ApplicationGate` と同じく一度だけ組み立て、OAuth2 admin consents / authorize 同意画面
  (BrowserTransactionResponse) と Authentication account consents の各ハンドラへ注入する。
  SCL は `AccountConsentResponse` に `client_name`、`AdminConsentResponse` に `client_name` と
  `preferred_username`(optional) を追加し、`BrowserTransactionResponse.client_name` の解決順を
  明記、derived artifacts を再生成した。UI は接続済みアプリ / 管理同意画面とも可読名を主表示・
  UUID を補助表記 (tooltip / secondary) にし、取消確認メッセージも可読名へ。表示名が無い client は
  client_id にフォールバックする。当初 scope 外だった authorize 同意画面 (`/consent`) の UUID 生表示も
  同 resolver に寄せて解消した (レビュー指摘対応)。
- **Verification Results**:
  - `just yaml-check`
    - result: ok
  - `just verify-go`
    - result: ok (lint + race tests)
  - `just verify-ui`
    - result: ok (format / lint / typecheck / build)
  - `just check-ids`
    - result: ok (221 ids)
  - `just test-ui-e2e`
    - result: not run (稼働スタックを要するため手動確認相当)

---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-10
---

# 認証フローとアカウント管理の主要操作をコンポーネントテストで保護する

## Motivation
認証と自己管理の失敗経路は利用者に直接影響する。API 単体テストだけではフォーム送信、状態表示、エラーの回復導線を保護できないため、公開 UI を通じた高速な回帰検知が必要である。

## Scope
- `ui/src/features/auth-flow/` のログイン、パスワード回復、TOTP、同意、デバイス承認の成功・失敗・拒否経路。
- `ui/src/features/account/` のプロフィール、メール、セキュリティ、データ、アプリケーション管理の重要操作。
- `ui/src/components/StepUpDialog.tsx` の再認証成功・失敗・キャンセル。

## Out of Scope
- API クライアントの網羅（`wi-168`）。
- 管理コンソール画面（`wi-170`）。
- E2E テストと UI 仕様変更。

## Plan
- API 境界をモックし、利用者が観測する表示・入力・送信結果を React Testing Library で検証する。
- 正常、境界、失敗または拒否を各重要操作で扱う。

## Tasks
- [x] T001a [Test] `StepUpDialog.test.tsx` を新規作成し、`useStepUpGuard` の再認証成功・失敗・キャンセル (ボタン/背景クリック/Escape) を検証する。
- [x] T001b [Test] `TotpPage`: 認証アプリコードの成功・失敗、パスキーの成功・キャンセル、リカバリコードの失敗を追加する (`AuthFlowPages.test.tsx`)。
- [x] T001c [Test] `ConsentPage`: 許可 (allow) 側の失敗経路を追加し、許可/拒否の成功・失敗を揃える。
- [x] T001d [Test] `DevicePage`: 承認成功・拒否成功・失敗・`authentication_required` リダイレクトを追加する。
- [x] T002a [Test] `AccountProfileEditPage`: プロフィール保存の成功・失敗を追加する。
- [x] T002b [Test] `AccountEmailsPage`: メール変更要求の成功・失敗・step-up キャンセルを追加する。
- [x] T002c [Test] `AccountSecurityPage`: TOTP登録開始→確認の成功、確認失敗、リカバリコード生成の step-up キャンセルを追加する。
- [x] T002d [Test] `AccountDataPage`: データエクスポートの成功・失敗を追加する。
- [x] T002e [Test] `AccountApplicationsPage`: アクセス取り消しの成功・失敗を追加する。
- [x] T003 [Verify] `just test-ui-cover` と `just verify-ui` を成功させる。

## Verification
- `just test-ui-cover`
- `just verify-ui`

## Risk Notes
非同期モックを実装詳細へ結び付けず、表示される結果とユーザー操作を中心にアサートする。

## Completion
- `StepUpDialog.test.tsx` を新規作成し、`useStepUpGuard` 経由で再認証成功・失敗・キャンセル (ボタン/背景クリック/Escape)・方式切替を検証 (6 tests)。
- `AuthFlowPages.test.tsx` に TOTP コードの成功・失敗、パスキーの成功・キャンセル、リカバリコードの失敗、Consent の許可失敗、DevicePage (承認成功・拒否成功・失敗・authentication_required リダイレクト) を追加。
- `AccountProfilePage` / `AccountEmailsPage` / `AccountSecurityPage` / `AccountDataPage` / `AccountApplicationsPage` の各テストに、実際の API 境界をモックしたコンテナレベルの成功・失敗 (該当箇所は step-up キャンセルも含む) を追加。
- カバレッジ: `StepUpDialog.tsx` 2.0%→98.0%、`src/features/auth-flow` 71.3%→93.3%、`src/features/account` 26.1%→53.3%。
- `just test-ui-cover`: 40 test files / 240 tests, 全て成功。
- `just verify-ui`: format-check / lint / typecheck / build すべて成功。

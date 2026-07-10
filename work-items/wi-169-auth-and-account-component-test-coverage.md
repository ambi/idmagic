---
status: in_progress
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
- [ ] T001 [Test] 認証フローの主要フォームの成功・失敗・拒否を追加する。
- [ ] T002 [Test] アカウント管理と step-up 再認証の主要操作を追加する。
- [ ] T003 [Verify] `just test-ui-cover` と `just verify-ui` を成功させる。

## Verification
- `just test-ui-cover`
- `just verify-ui`

## Risk Notes
非同期モックを実装詳細へ結び付けず、表示される結果とユーザー操作を中心にアサートする。

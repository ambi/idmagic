---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-10
---

# 認証フロー画面におけるプレゼンテーションコンポーネントとロジックの分離および単体テストの拡充

## Motivation
`wi-133` でアカウントポータル配下（`ui/src/features/account/`）の主要 `*Page.tsx` はすべて Container / Presentation 分離と単体テスト追加、`ui/ARCHITECTURE.md` へのガイドライン明記まで完了した。一方、認証フロー（`ui/src/features/auth-flow/`）の主要画面には API 呼び出し・状態管理・フォーム表示が混在している。
`wi-133` と同じ分離アプローチ・テスト方針を認証フローへ適用し、テストカバレッジと実装構造の一貫性を保つ。管理コンソールは `wi-166` と `wi-167` に分割する。

## Scope
- `ui/src/features/auth-flow/` 配下の主要 `*Page.tsx`: `CallbackPage`, `ConsentPage`, `DevicePage`, `EmailVerifyPage`, `ForgotPasswordPage`, `HomePage`, `LoginPage`, `ResetPasswordPage`, `StatusPage`, `TotpPage`。
- 上記画面から API 呼び出し・状態管理（Container）を分離し、フォーム表示・バリデーション・一覧表示などのセクション単位でプレゼンテーションコンポーネントへ切り出す。
- 各画面の入力バリデーションロジックを pure な関数（`ui/src/lib/validation.ts` か画面ローカルの pure function）へ切り出す。
- 切り出したすべての Presentation コンポーネントおよび pure function に対する Vitest/React Testing Library 単体テストを追加する。

## Out of Scope
- 各画面の挙動やスタイルの変更（UI の見た目や動作仕様は一切変更しない）。
- E2E テストシナリオの変更。
- `ui/src/features/account/` 配下（`wi-133` で対応済み）の再分割。
- 管理コンソールとシステムコンソールの画面分離・テスト追加（`wi-166` / `wi-167`）。

## Plan
- SCL のドメイン仕様・外部契約・画面遷移仕様は変更しない。`spec/scl.yaml` は更新しない。
- 分離方針は [ui/ARCHITECTURE.md](file:///Users/tn/src/idmagic/ui/ARCHITECTURE.md) の「Container / Presentation component split」セクション（`wi-132`/`wi-133` で明記済み）にそのまま従う。
  - ページ全体を 1 個の `XxxPresentation` に丸ごと包む split は避ける。意味のあるセクション（フォーム・一覧・カードなど）ごとに小さいプレゼンテーションコンポーネントへ切り出し、props は概ね 10 未満に抑える。
  - 静的な read-only マークアップは無理に切り出さず、container にインラインで残してよい。
  - `AccountShell`/`AdminShell`/`AuthShell`/`SystemShell` など TanStack Router の `Link` を使うラッパーをテストで render する場合は `src/test/renderWithRouter.tsx` を使う。
- `CallbackPage`、`HomePage`、`StatusPage` は副作用や状態を持たない静的画面のため、無理に分割せず page 自体を presentation としてテストする。

## Tasks
- [x] T001 [UI] `auth-flow` 配下の stateful な主要 `*Page.tsx` を Container / Presentation に分離する。
- [x] T002 [Test] 抽出した Presentation / pure function と静的画面の単体テストを追加する。
- [x] T003 [Verify] `just yaml-check`、`just test-ui-unit`、`just verify-ui` を通す。

## Verification
- `just verify-ui`
- `just test-ui-unit`
- 対象画面が開発サーバー（`just dev-ui`）で正常にレンダリング・動作することの確認。

## Risk Notes
管理コンソールの主要画面構造を広く変更するため、ルーティング接続や API 呼び出しとの接続が壊れるリスクがある。グループごとに検証ゲート（`just verify-ui`）を通してから次のグループへ進み、未検証の変更を積み上げない。仕上げに既存の E2E テスト（`just test-ui-e2e`）で回帰がないことを確認する。

## Completion

- **Completed At**: 2026-07-10
- **Summary**: `auth-flow` の stateful 7 画面からフォーム・操作領域を小さな Presentation コンポーネントとして切り出し、Container に API 呼び出しと状態管理を残した。デバイスコード正規化と第二要素方式の選択は pure function に抽出し、静的な Callback / Home / Status 画面を含む表示分岐と抽出物を単体テストでカバーした。
- **Affected Guarantees State**: 既存の画面契約、API 呼び出し、ルーティング、表示スタイルは維持されている。SCL のドメイン仕様・外部契約・画面遷移に変更はない。
- **Verification Results**:
  - `just yaml-check` — passed
  - `just test-ui-unit` — passed (166 tests, 29 files)
  - `just verify-ui` — passed (format check, lint, typecheck, unit tests, production build)
- **Evidence**:
  - 実行日: 2026-07-10
  - 実行環境: ローカル開発環境
  - 実行主体: Codex
  - 対象ソース版: `main`（コミット前）
  - 保存先: CI 外部成果物なし。上記コマンドの成功結果を本記録に要約。

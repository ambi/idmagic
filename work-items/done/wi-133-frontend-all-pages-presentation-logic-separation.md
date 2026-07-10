---
status: completed
authors: [tn]
risk: medium
created_at: 2026-07-05
---

# アカウントポータル画面におけるプレゼンテーションコンポーネントとロジックの分離および単体テストの拡充

## Motivation
フロントエンドの `AdminSignInPolicyPage.tsx` においては `wi-132` でコンテナ・プレゼンタパターンによる UI と副作用の分離が実証されたが、他の主要画面（Page.tsx）は依然として密結合なままである。段階的なリファクタリングに任せる形では、ルールや期限が曖昧になり、未修正の画面が長期間放置されてコードベースの一貫性が損なわれるリスクがある。また、テストカバレッジ全体の早期向上も妨げられる。
そのため、まずアカウントポータル（`ui/src/features/account/`）配下の主要画面に対してこの分離アプローチを適用し、テスト容易性を高めるとともにテストコードを拡充する。管理コンソールおよび認証フロー画面は範囲が大きいため `wi-162` に分割して引き継ぐ。

## Scope
- `ui/src/features/account/` 配下の主要 `*Page.tsx`（`AccountHomePage`, `AccountEmailsPage`, `AccountActivityPage`, `AccountAppsPage`, `ChangePasswordPage`, `AccountProfilePage`/`AccountProfileEditPage`, `AccountSecurityPage`, `AccountApplicationsPage`, `AccountDataPage`）から、ビジネスロジックやフォーム表示・バリデーションを担当するプレゼンテーションコンポーネントを切り出す。
- 各画面から API 呼び出しや状態管理（Container）を分離し、Props を介して疎結合にする。
- 各画面の入力バリデーションロジックを pure な関数へユーティリティ化して切り出す。
- 切り出したすべての Presentation コンポーネントおよびユーティリティに対する Vitest/React Testing Library 単体テストを追加し、テストカバレッジを向上させる。
- 今後新規追加される画面においても、このコンテナ・プレゼンタ分離パターンを原則として適用することを、[ui/ARCHITECTURE.md](file:///Users/tn/src/idmagic/ui/ARCHITECTURE.md) にガイドラインとして明記し、開発ルール化する。

## Out of Scope
- 各画面の挙動やスタイルの変更（UI の見た目や動作仕様は一切変更しない）。
- E2E テストシナリオの変更。
- 管理コンソール（`admin-*`, `system-tenants`）および認証フロー（`auth-flow`）画面の分離・テスト追加 — `wi-162` で対応する。

## Plan
- SCL のドメイン仕様・外部契約・画面遷移仕様は変更しない。今回は UI 実装構造と開発ルールの変更なので、`spec/scl.yaml` は更新せず、`ui/ARCHITECTURE.md` にコンテナ・プレゼンタ分離方針を追記する。
- `ui/src/features/account/**Page.tsx` を、API 呼び出し・副作用・ページ遷移を持つ Container と、props のみで描画できる `*Presentation` / 小さな form/list コンポーネントへ段階的に切り分ける。
- 入力値の整形・検証・表示派生値は既存の `ui/src/lib/validation.ts` か画面ローカルの pure function に寄せ、単体テスト対象にする。
- 既存 UI の DOM・文言・遷移は変えず、ルートファイルからの props 契約を維持する。
- 当初は「ページ全体を 1 個の `XxxPresentation` に丸ごと包む」方式で着手したが、`AccountSecurityPage` で props が 27 個に膨らみ可読性・テスト容易性の改善効果が薄いと判明したため、途中でセクション単位（フォーム・一覧・カードごと）の小さいプレゼンテーションコンポーネントへ切り出す方式（`wi-132` の `DefaultPolicyFormPresentation` と同じ粒度）に変更した。この方針を `ui/ARCHITECTURE.md` の「Container / Presentation component split」節に明文化した。

## Tasks
- [x] T001 [Architecture] `ui/ARCHITECTURE.md` に新規ページ実装時のコンテナ・プレゼンタ分離ルールを追加する。
- [x] T002 [UI] 管理画面の主要 `*Page.tsx` を Container / Presentation に分離する。→ `wi-162` へ分割。
- [x] T003 [UI] アカウント画面の主要 `*Page.tsx` を Container / Presentation に分離する。
- [x] T004 [Test] 抽出した Presentation / pure function の単体テストを追加する。
- [x] T005 [Verify] `just yaml-check`、`just test-ui-unit`、`just verify-ui` を通す。

## Verification
- `just verify-ui`
- `just test-ui-unit`
- すべての画面でリファクタリング後に画面が正常にレンダリング・動作することの確認。

## Risk Notes
アカウントポータルの主要画面コンポーネントの構造を変更するため、ルーティングの接続や API 呼び出しとの接続が壊れるリスクがあります。リファクタリング完了後は、既存の E2E テスト（`just test-ui-e2e`）および開発サーバー（`just dev-ui`）での手動動作確認で慎重に確認する必要があります。

## Completion
- **Completed At**: 2026-07-10
- **Summary**:
  `ui/src/features/account/` 配下の9画面（`AccountHomePage`, `AccountEmailsPage`, `AccountActivityPage`, `AccountAppsPage`, `ChangePasswordPage`, `AccountProfilePage`/`AccountProfileEditPage`, `AccountSecurityPage`, `AccountApplicationsPage`, `AccountDataPage`）を Container / Presentation に分離した。複数セクションを持つ画面（`AccountSecurityPage`, `AccountActivityPage`）はセクション単位の小さいプレゼンテーションコンポーネント（`TotpEnrollmentForm`, `TotpRemovalForm`, `PasskeyList`, `PasskeyRegisterForm`, `RecoveryCodesPanel`, `SessionsSection`, `ActivityHistorySection` など）へ切り出し、単一画面の全 state を受け取る巨大な Presentation コンポーネントは作らない方針にした。`AccountAppsPage` は dnd-kit の都合上 1 コンポーネントに留めたが、派生値（sections/grouped/itemIDs）の計算を container から Presentation 側の `useMemo` へ移し props 数を削減した。
  `AccountShell`/`AuthShell` など TanStack Router の `Link` を使うコンポーネントをテストで render できるよう、`src/test/renderWithRouter.tsx` を新設した。
  `ui/ARCHITECTURE.md` に「Container / Presentation component split」節を追加し、セクション単位の分離・小さい props・副作用を持たない・抽出物には単体テストを付けるという4原則を明文化した。
  対象9画面 + pure function 群に対し Vitest/Testing Library の単体テストを83件追加した（既存分含めユニットテスト全体で149件 green）。
  管理コンソール（`admin-*`, `system-tenants`）と認証フロー（`auth-flow`）画面は範囲が大きいため `wi-162` に分割して引き継いだ。
- **Verification Results**:
  - `just yaml-check` - passed
  - `just verify-ui` (format-check / lint / typecheck / build) - passed
  - `just test-ui-unit` - passed (149 tests, 24 files)

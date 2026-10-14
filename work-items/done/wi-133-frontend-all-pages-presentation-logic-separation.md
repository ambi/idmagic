---
status: completed
archived_from: c846d5e549cbbe72c662efb5054e4abdb53d670b
---

# アカウントポータル画面におけるプレゼンテーションコンポーネントとロジックの分離および単体テストの拡充

`ui/src/features/account/` 配下の9画面（`AccountHomePage`, `AccountEmailsPage`, `AccountActivityPage`, `AccountAppsPage`, `ChangePasswordPage`, `AccountProfilePage`/`AccountProfileEditPage`, `AccountSecurityPage`, `AccountApplicationsPage`, `AccountDataPage`）を Container / Presentation に分離した。複数セクションを持つ画面（`AccountSecurityPage`, `AccountActivityPage`）はセクション単位の小さいプレゼンテーションコンポーネント（`TotpEnrollmentForm`, `TotpRemovalForm`, `PasskeyList`, `PasskeyRegisterForm`, `RecoveryCodesPanel`, `SessionsSection`, `ActivityHistorySection` など）へ切り出し、単一画面の全 state を受け取る巨大な Presentation コンポーネントは作らない方針にした。`AccountAppsPage` は dnd-kit の都合上 1 コンポーネントに留めたが、派生値（sections/grouped/itemIDs）の計算を container から Presentation 側の `useMemo` へ移し props 数を削減した。
`AccountShell`/`AuthShell` など TanStack Router の `Link` を使うコンポーネントをテストで render できるよう、`src/test/renderWithRouter.tsx` を新設した。
`ui/ARCHITECTURE.md` に「Container / Presentation component split」節を追加し、セクション単位の分離・小さい props・副作用を持たない・抽出物には単体テストを付けるという4原則を明文化した。
対象9画面 + pure function 群に対し Vitest/Testing Library の単体テストを83件追加した（既存分含めユニットテスト全体で149件 green）。
管理コンソール（`admin-*`, `system-tenants`）と認証フロー（`auth-flow`）画面は範囲が大きいため `wi-162` に分割して引き継いだ。

全文は `git show c846d5e549cbbe72c662efb5054e4abdb53d670b:<このファイルのパス>` で読む。

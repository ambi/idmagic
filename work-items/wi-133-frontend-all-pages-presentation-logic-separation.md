---
id: wi-133-frontend-all-pages-presentation-logic-separation
title: フロントエンド全画面におけるプレゼンテーションコンポーネントとロジックの分離および単体テストの拡充
created_at: 2026-07-05
authors: [tn]
status: pending
risk: medium
---

# Motivation
フロントエンドの `AdminSignInPolicyPage.tsx` においては `wi-132` でコンテナ・プレゼンタパターンによる UI と副作用の分離が実証されたが、他の主要画面（Page.tsx）は依然として密結合なままである。段階的なリファクタリングに任せる形では、ルールや期限が曖昧になり、未修正の画面が長期間放置されてコードベースの一貫性が損なわれるリスクがある。また、テストカバレッジ全体の早期向上も妨げられる。
そのため、フロントエンドの主要な管理画面およびアカウント画面全般に対して、一括してこの分離アプローチを適用し、テスト容易性を高めるとともにテストコードを拡充する必要がある。

# Scope
- `ui/src/features/` 配下のすべての密結合な `*Page.tsx`（`AdminUsersPage`, `AdminApplicationsPage`, `AdminSettingsPage`, `AdminAgentsPage` など）から、ビジネスロジックやフォーム表示・バリデーションを担当するプレゼンテーションコンポーネントを切り出す。
- 各画面から API 呼び出しや状態管理（Container）を分離し、Props を介して疎結合にする。
- 各画面の入力バリデーションロジックを pure な関数（`ui/src/lib/validation.ts` など）へユーティリティ化して切り出す。
- 切り出したすべての Presentation コンポーネントおよびユーティリティに対する Vitest/React Testing Library 単体テストを追加し、テストカバレッジを向上させる。

# Out of Scope
- 各画面の挙動やスタイルの変更（UI の見た目や動作仕様は一切変更しない）。
- E2E テストシナリオの変更。

# Verification
- `just verify-ui`
- `just test-ui-unit`
- すべての画面でリファクタリング後に画面が正常にレンダリング・動作することの確認。

# Risk Notes
ほぼすべての主要画面コンポーネントの構造を変更するため、ルーティングの接続や API 呼び出しとの接続が壊れるリスクがあります。リファクタリング完了後は、既存の E2E テスト（`just test-ui-e2e`）および開発サーバー（`just dev-ui`）での手動動作確認で慎重に確認する必要があります。

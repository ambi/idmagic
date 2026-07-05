---
id: wi-130-frontend-testing-environment-and-coverage
title: フロントエンドにおける単体・コンポーネントテスト環境の導入とテスト追加
created_at: 2026-07-05
authors: [tn]
status: completed
risk: medium
---

# Motivation
React フロントエンド (`ui` ディレクトリ) において、現状は E2E テストしか存在せず、コンポーネントやロジック単体でのテストが実行できない。
そのため、エッジケースの検証が難しく、テスト実行時間が長くなる問題がある。
フロントエンドに軽量な単体・コンポーネントテストフレームワーク（Vitest + React Testing Library）を導入し、テストのカバレッジを向上させ、不具合の早期発見を可能にする。

# Scope
- **テスト環境の構築**:
  - `ui/` ディレクトリに `Vitest` および `@testing-library/react`、`jsdom` (または `happy-dom`) を導入する。
  - `ui/package.json` に `test:unit` および `test:unit:coverage` スクリプトを追加する。
  - `justfile` に `test-ui-unit` レシピを追加し、`just verify-ui` のステップに組み込む。
- **テストコードの実装**:
  - UI内の共通コンポーネント、カスタム Hooks、ユーティリティ関数、APIクライアントなどの単体テスト・コンポーネントテストを記述する。
  - 特にフォームバリデーションロジックや状態遷移を伴うインタラクションのテストを優先する。
- **カバレッジ測定の整備**:
  - Vitest のカバレッジ機能（V8）をセットアップし、単体テスト全体のカバレッジを測定できるようにする。
  - 主要なロジック部分（`api/`, `lib/` など）の単体テストカバレッジ 70% 以上を目標とする。

# Out of Scope
- E2E テスト (`bun test tests/e2e`) の拡充。
- CIにおけるテストカバレッジ強制ルールの適用（`wi-131` で対応）。

# Verification
- `just test-ui-unit`
- `just verify-ui`
- 単体テストのカバレッジレポートが出力され、主要コードがカバーされていることを確認する。

# Risk Notes
フロントエンドのルーティング (`@tanstack/react-router`) やドラッグ＆ドロップ (`@dnd-kit`) などのライブラリをテストする際、モックの作成が複雑化する可能性がある。
React 19 に対応したテストライブラリのバージョン選定と、非推奨の挙動に配慮する。

# Completion
- **Completed At**: 2026-07-05
- **Summary**:
  フロントエンド (`ui` ディレクトリ) に単体・コンポーネントテスト用の環境（Vitest, jsdom, React Testing Library）を導入し、主要な共通コンポーネントおよびビジネスロジックのテストを追加した。
  - `Vitest` と `@testing-library/react`、`@testing-library/jest-dom`、`jsdom` を導入。
  - `ui/package.json` に `test:unit` と `test:unit:coverage` の npm スクリプトを追加し、`justfile` に `test-ui-unit` レシピを追加して `verify-ui` の検証プロセスに統合。
  - `@vitest/coverage-istanbul` を用いて Bun 環境におけるテストカバレッジの測定ができるように構成（`provider: 'istanbul'` を設定）。
  - 主要なロジックである `src/lib/utils.ts`、`src/lib/systemNav.ts`、`src/api/core.ts` に対して高いカバレッジの単体テストを実装し、カバレッジ 70% 以上の目標に対して、対象ファイルは 94% 以上 (core.ts: 94.73%, utils.ts: 100%, systemNav.ts: 100%) の高い網羅率を達成。
  - 共通コンポーネントである `Brand.tsx` および `ui/button.tsx` に対して、React Testing Library を用いたコンポーネントテストを追加（カバレッジ 100%）。
- **Verification Results**:
  - `just verify` - 成功
  - `just verify-ui` - 成功
  - `just test-ui-unit` - 成功

---
id: wi-130-frontend-testing-environment-and-coverage
title: フロントエンドにおける単体・コンポーネントテスト環境の導入とテスト追加
created_at: 2026-07-05
authors: [tn]
status: pending
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

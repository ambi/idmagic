---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# フロントエンドにおける単体・コンポーネントテスト環境の導入とテスト追加

フロントエンド (`ui` ディレクトリ) に単体・コンポーネントテスト用の環境（Vitest, jsdom, React Testing Library）を導入し、主要な共通コンポーネントおよびビジネスロジックのテストを追加した。
- `Vitest` と `@testing-library/react`、`@testing-library/jest-dom`、`jsdom` を導入。
- `ui/package.json` に `test:unit` と `test:unit:coverage` の npm スクリプトを追加し、`justfile` に `test-ui-unit` レシピを追加して `verify-ui` の検証プロセスに統合。
- `@vitest/coverage-istanbul` を用いて Bun 環境におけるテストカバレッジの測定ができるように構成（`provider: 'istanbul'` を設定）。
- 主要なロジックである `src/lib/utils.ts`、`src/lib/systemNav.ts`、`src/api/core.ts` に対して高いカバレッジの単体テストを実装し、カバレッジ 70% 以上の目標に対して、対象ファイルは 94% 以上 (core.ts: 94.73%, utils.ts: 100%, systemNav.ts: 100%) の高い網羅率を達成。
- 共通コンポーネントである `Brand.tsx` および `ui/button.tsx` に対して、React Testing Library を用いたコンポーネントテストを追加（カバレッジ 100%）。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。

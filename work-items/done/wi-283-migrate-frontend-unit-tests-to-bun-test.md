---
status: completed
archived_from: 1191ef86a61ec87a10bf22f8328a16ebcbe85650
---

# フロントエンド単体・コンポーネントテストを Vitest から Bun test へ全面移行する

フロントエンドの単体・コンポーネントテスト基盤を Vitest から Bun test へ全面移行した。
JS DOM の課題を解決するため Happy DOM への置き換えと 2段階の preload (`register-dom.ts`, `setup.ts`) を導入し、location snapshot を使った互換性を確保。
`vi` API 依存部分 (spyOn/mock など) を Bun ネイティブへ書き換え、185箇所の `vi.stubGlobal` については独自ヘルパー `src/test/globals.ts` へ集約した。
残存する非同期タイミング差テスト 2件の対応、及び lcov カバレッジレポーター生成への切り替えを行い、CI (GitHub Actions) から Vitest への依存を根絶。
すべての 427 テストが green で通過し、実行速度の大幅な短縮と互換性の確保を達成した。

全文は `git show 1191ef86a61ec87a10bf22f8328a16ebcbe85650:<このファイルのパス>` で読む。

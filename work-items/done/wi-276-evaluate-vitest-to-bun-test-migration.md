---
status: completed
archived_from: 295a9c119b8f1ae81a5010cfd53aacf10592c20c
---

# フロントエンド単体・コンポーネントテストの Vitest から Bun test への移行可否を実証する

Vitest → Bun test 移行の可否を実証し、判断は **Go（移行は現実的）**。`frontend/` 全 77 test files / 427 tests を
隔離した spike へ機械変換し、Happy DOM を登録する 2 段 preload と実使用 6 API のみの互換シムで Bun 実行した結果、
preload のみ・テスト無改変で **425/427 が pass**、同一 green セットで **Bun は Vitest の約 3.6 倍速**
（75 files / 420 tests: Bun 4.0s vs Vitest 14.7s）であることを確認した。
互換性の主因は「Happy DOM の `Location` がスプレッド不可で、多用される `vi.stubGlobal('location', {...originalLocation})`
が `pathname`/`origin` を失いアプリ側でサイレント失敗する」点で、preload でのスプレッド可能な location snapshot 注入により解消した。
spike 一式と探索用依存は判断確定後に撤去し、本 Work Item では独立の安全整理として未使用の `@vitest/coverage-v8` のみ削除した。
全面移行は後続 Work Item [[wi-283-migrate-frontend-unit-tests-to-bun-test]] へ委譲する。

全文は `git show 295a9c119b8f1ae81a5010cfd53aacf10592c20c:<このファイルのパス>` で読む。

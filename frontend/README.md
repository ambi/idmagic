# IdMagic の UI

このディレクトリには、IdMagic の Web フロントエンド・アプリケーション（React の SPA と、それを配信する Caddy のゲートウェイ）がある。

設計と開発の手順は、次の文書に記載している。

- [フロントエンド設計](../docs/design/application/frontend.md)：コードの分け方、ルーティング、データの取得、ビルドと配信、テスト、依存の宣言、開発時の作業
- [ユーザーインターフェース設計](../docs/design/application/user-interface.md)：画面を跨いで揃える表示と操作の規範

## よく使うタスク

| 作業 | タスク |
| --- | --- |
| 開発サーバーの起動 | `mise run dev-ui`（API と合わせて起動するなら `mise run dev`） |
| UI の変更の検証 | `mise run verify-ui` |
| 単体テスト | `mise run test-ui-unit` |
| E2E テスト | `mise run test-ui-e2e` |
| 依存宣言の検査 | `mise run check-ui-dependencies` |
| 基本部品の追加 | `mise run add-ui-component -- <component>...` |

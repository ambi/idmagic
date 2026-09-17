# IdMagic の UI

このディレクトリは、ホステッド認証画面、アカウントポータル、管理コンソール、システムコンソールの SPA と、それを配信するゲートウェイを持つ。
この文書は、UI を変更するときに実行する手順を持つ。
コードの分け方、ルーティング、状態の持ち方、ライブラリの選定理由は[フロントエンド設計](../docs/design/application/frontend.md)が、画面を跨いで揃える利用者向けの規範は[ユーザーインターフェース設計](../docs/design/application/user-interface.md)が持つ。

## 変更の検証

UI を変更したときは、次の検証を実行する。

```bash
mise run verify-ui
```

`mise run verify-ui` は依存宣言の検査、整形の検査、静的解析、型検査、単体テスト、ビルドを実行する。
同じ一式がリポジトリ全体の `mise run verify` と CI にも入っているので、どの入口から実行しても結果は変わらない。

API 契約を変更したときは、Go の HTTP E2E テストを実行し、Cookie、CSRF 防御、OAuth のリダイレクト、JSON スキーマが正しいことを検証する。

Vite CLI は `#!/usr/bin/env node` を使うが、開発とビルドのスクリプトは `bun` で JavaScript の起点を直接実行する。
これにより Node.js のランタイムを必要とせず、実行するプロセスを `bun .../vite.js` に統一する。

## 依存の宣言

`idmagic-ui` は公開ライブラリではない私用アプリケーションなので、互換範囲を利用者へ伝える必要がない。
直接依存は `package.json` で範囲指定せずに固定し、推移依存は `bun.lock` で固定する。
更新は Renovate か明示的な保守作業で行い、協調して動くパッケージ（React と型定義、TanStack Router と Vite プラグイン、Tailwind CSS と Vite プラグインなど）は同じ更新で揃える。

ブラウザー向けのソースから参照されて Vite が成果物へ組み込むものが `dependencies`、ビルド、生成、整形、静的解析、型検査、テストのためだけに実行するものが `devDependencies` である。
`shadcn` は `src/styles.css` へ CSS を供給し、コンポーネント生成 CLI も提供するが、どちらもビルド時にしか動かないので `devDependencies` に置く。

```bash
mise run check-ui-dependencies
```

この検査は Knip が担う。
直接依存の要否を TypeScript の import だけで決めず、CSS の `@import`、Vite と Biome の設定、Bun のテスト preload、パッケージスクリプト、ルート生成器も入口として数える。
未使用の直接依存と、推移依存へ暗黙に頼っている未宣言の依存の両方を失敗として報告する。
入口は `knip.jsonc` に書き写さず、実在する設定ファイルから Knip のプラグインが読み取る。

## shadcn コンポーネントの追加

```bash
mise run add-ui-component -- button card
```

`devDependencies` に固定した版の CLI を使い、`components.json` の設定で `src/components/ui` へ生成する。
`npx shadcn@latest` は使わない。
実行のたびに取得する版が変わり、Bun に統一した実行環境からも外れるためである。
生成されたソースと、それが必要とする直接依存は通常の差分としてレビューする。

## E2E スモークテスト

次のコマンドで SPA の主要経路（`/authorize → login → consent → callback`）を検証する。

```bash
mise run test-ui-e2e
```

テストランナーは `bun test` と組み込みの `Bun.WebView` を使う。
macOS では WKWebView、Linux と Windows では CDP 経由の Chrome を使うため、大規模なブラウザー自動化フレームワークや手動でのドライバーのダウンロードは不要である。

テスト一式（`tests/e2e/`）は次のライフサイクルを自動的に管理する。

1. **Go API**：ブラウザーのオリジンと一致させて CSRF 検査を通すため、`memory` モードでポート `:8081`（`ADDR=:8081 ISSUER=http://localhost:5173`）に起動する。
2. **Vite 開発サーバー**：ポート `:5173` に起動し、ブラウザーフロー、API、公開プロトコルのエンドポイントへのリクエストを `8081` へプロキシする。
3. **模擬コールバックサーバー**：ポート `:3000` に起動し、`development` seed の外部デモクライアントに設定した `redirect_uri`（`http://localhost:3000/callback`、クライアント ID `00000000-0000-4000-8000-000000000021`）で認可コードを受け取る。

この構成はクライアント側のルーティング（`meta[name="idmagic:page"]`）を検証し、オリジンを跨ぐリダイレクトで `code` と `iss` のパラメーターが保たれることを確認する（RFC 9207）。
`PATH` に必要なのは `go` と `bun` だけである。
テストの構成と拡張時の規則は `tests/e2e/README.md` が持つ。

---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# フロントエンドの依存宣言と開発設定を現行のツールチェーンへそろえる

`mise run spec-diff` は `no normative specification change against main` を返す。規範仕様の差分は無く、変わったのは依存の宣言、開発ツールの設定、検証入口である。`shadcn` は `devDependencies` へ移り、未使用の `@tanstack/react-table` と、`@happy-dom/global-registrator` の推移依存でしかなかった `happy-dom` を削除し、`scripts/generate-routes.ts` が暗黙に使っていた `@tanstack/router-generator` を直接依存として宣言した。全直接依存を範囲指定せずに固定し、`@happy-dom/global-registrator` だけが 20.11.13 から 20.11.15 へ上がった。Knip を `devDependencies` に加え、`frontend/knip.jsonc` で依存関係の問題種別だけを有効にして `mise run check-ui-dependencies` を新設し、`verify-ui`、リポジトリ全体の `verify` と `verify-serial`、CI へ配線した。`verify-ui` には `typecheck-ui` も加わり、三つの入口が同じフロントエンド検査を実行する。固定版の CLI を使う `mise run add-ui-component` を追加した。`vite.config.ts` の `__dirname` は `import.meta.dirname` へ移した。`bunfig.toml` の preload が E2E の起動待ちを遮断していた問題を直し、E2E は 4 ファイル 24 件が通るようになった。
着手時に版が食い違うと見ていた TanStack Router の実行時パッケージと Vite プラグインは、実際には食い違いではなかった。`@tanstack/react-router` 1.170.32 と `@tanstack/router-plugin` 1.168.35 はどちらも公開されている最新版であり、プラグインは `@tanstack/router-generator` 1.167.33 を厳密に固定している。TanStack が各パッケージを独立に採番しているだけなので、版をそろえる変更はしていない。Bun も 1.4.0 が最新で、`mise.toml`、`packageManager`、`Dockerfile` の三つは既に一致していた。上げる差分は無いが、次にずれたときに気づけるよう版一致の検査だけを残した。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。

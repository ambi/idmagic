---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# SPA E2E スモークテストを Bun のヘッドレスブラウザ (Bun.WebView) で 1 本だけ導入する

SPA の golden path (authorize -> login -> consent -> callback) を 1 本の
E2E スモークとして導入した。ランナーは Bun 組み込みの `Bun.WebView` +
`bun test` で、外部のブラウザ自動化フレームワークやブラウザバイナリ取得に
依存しない (macOS は WKWebView)。テストは Go API (memory, :8081)・Vite dev
(:5173)・demo-client の redirect_uri を受ける最小コールバックサーバ (:3000) を
自動起動/停止し、(1) ログイン画面で `meta[name="idmagic:page"]` が `login` で
あること + `input[name="username"]` の存在、(2) ログイン後に `consent` へ遷移
すること、(3) 「許可して続行」後の cross-origin redirect で callback URL に
`code` / `iss` / `state` が乗ること、を assert する。dispatcher 不変条件を
成立させるため `ui/src/main.tsx` に `meta[name="idmagic:page"]` 出力を追加した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。

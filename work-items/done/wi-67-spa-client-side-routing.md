---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 管理コンソール / アカウントポータルの画面遷移を client-side routing 化して全リロードを無くす

SPA を TanStack Router の loader ベースの client-side routing に作り替えた。各パスは
共有 loader (resolvePageData) で「その route のデータだけ」を取得し、PageView が
解決済み PageData の kind に応じてページを描画する。サイドバー / パンくず / アカウント
メニュー / ロゴのナビゲーションを `<Link>` に置き換え、画面遷移でドキュメントを再読込
しない。OIDC ログイン guard (ensureLoggedIn) は loader 内で動くため、初期ロードと
in-app 遷移の両方に適用される。pageData は window.location 依存を排し、遷移先 location
(pathname / searchStr) を受け取る resolvePageData に分離した。markPage は Suspense 境界の
内側で呼び、lazy chunk のロード完了 (ページ DOM 実在) と meta 表明を一致させた。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。

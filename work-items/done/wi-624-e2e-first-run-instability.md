---
status: completed
archived_from: e8a8f2b210761ec5f21ac6f6d681b771dadd0ed3
---

# ブラウザー E2E が実行ごとに結果を変えるのをやめさせる

`mise run spec-diff` は、`main` と比較して規範仕様の変更がないと判定した。
`openWebView` が返す WebView は呼び出し単位の期限を持ち、応答しない呼び出しをメソッド、引数、期限、URL とともに `WebViewCallExpired` として報告する。
期限切れ後の追試が占有済みの `evaluate` を示す場合は oven-sh/bun#43412 を名指しし、Go、Vite、ページ console の記録場所も失敗へ添える。
待ちヘルパーは遷移中の通常の `evaluate` 失敗だけを再試行し、`WebViewCallExpired` は隠さず呼び出し元へ返す。

全文は `git show e8a8f2b210761ec5f21ac6f6d681b771dadd0ed3:<このファイルのパス>` で読む。

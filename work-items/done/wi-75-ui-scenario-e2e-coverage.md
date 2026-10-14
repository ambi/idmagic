---
status: completed
archived_from: 33db0e6cffc661750a20ca19f0549e73c950c3ed
---

# 追加した UI シナリオ (管理 / マイアカウント / ログイン補助) の E2E テストを整備する

Bun.WebView ベースの UI E2E fixture を整備し、Go サーバ、Vite、
callback listener、ローカル SMTP sink をテスト内で起動する構成にした。
メール変更とパスワードリセットは外部送信せず、SMTP sink で確認リンクを
捕捉して検証する。ログイン補助、マイアカウント、管理画面の主要ブラウザ
シナリオを `ui-scenario-smoke.spec.ts` と `ui-scenario-actions.spec.ts`
に追加した。

全文は `git show 33db0e6cffc661750a20ca19f0549e73c950c3ed:<このファイルのパス>` で読む。

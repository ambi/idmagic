---
status: completed
archived_from: 74aaa87411353d6e9fe6079107d7f46a184b50c3
---

# 使われていないページング補助の扱いを決める

`mise run spec-diff -- main` は規範仕様の差分なしと報告した。
URL と `PageNavigation` によるページ送りへ移行した後に残っていた二つの補助と専用テストを削除し、共通辞書から三つのキーを両言語で削除した。
未着手の wi-371 が削除対象を名指していた箇所を、現行の `PageNavigation` の説明へ更新した。
画面のページ送り、HTTP 契約、保存状態、通知とイベントは維持した。

全文は `git show 74aaa87411353d6e9fe6079107d7f46a184b50c3:<このファイルのパス>` で読む。

---
status: completed
archived_from: fa43cfa33d260afc3c42da870fe5558987677a5c
---

# main に取り込んだ完了記録を要約だけにし、全文は git 履歴に任せる

`mise run spec-diff` に規範の差分はない。
`mise run archive-work-items` と `move-work-item` は、main に取り込み済みで作業中の変更に含まれない完了記録を、`status`、`archived_from`、題名、要約だけに縮める。
要約は完了節の「要約」から取り、ない記録は動機の最初の段落を使う。リンクは文字だけを残す。
`check-work-items` は縮めた記録の形を検査し、`archived_from` のコミットに全文があることを `git cat-file --batch-check` の一回の呼び出しで確かめる。証拠と参照の検査は全文のコミットで済んでいるので適用しない。
既存の 614 件を縮め、`work-items/done/` は 9,001,002 バイトから 834,237 バイトになった。
計画したアンカー付きリンクの書き換えツールは作らなかった。該当は wi-82135 の 1 件だけで、以後の発生は `check-links` が報告するため、手で直した。
縮約を完了のたびに `move-work-item` から実行するようにした。手で実行する運用では、縮めていない記録が再び増えるためである。
wi-29959 が扱う完了記録の参照の追従は、縮めた記録が `affected_spec` を持たなくなったので、残りの必要性を見直す。

全文は `git show fa43cfa33d260afc3c42da870fe5558987677a5c:<このファイルのパス>` で読む。

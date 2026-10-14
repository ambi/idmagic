---
status: completed
archived_from: 0d89fe6d56243c535512758cbada9f7034aeb79d
---

# 正規仕様の差分を git から導出する spec-diff を追加する

`just spec-diff [ref]`（既定 `main`）を追加した。git のリビジョンと作業ツリーの双方から正規シナリオ、
状態遷移行、TypeSpec 宣言を抽出し、追加・削除・変更を集合差分として出す。シナリオ本文は空白と空行を
正規化して比較するため、整形だけの変更は差分に出ない。operation ごとのトランスポートラッパ
（`*HttpRequest` / `*HttpResponse` / `*Error<status>` / `*Success_<status>`）は宣言一覧から除外し、
出力を読める分量に保つ。ゲートは一切追加していない。
`just spec-diff HEAD~3` で、直近 3 コミットが追加した 3 シナリオと Group 属性まわりの宣言が
出ることを確認した。

全文は `git show 0d89fe6d56243c535512758cbada9f7034aeb79d:<このファイルのパス>` で読む。

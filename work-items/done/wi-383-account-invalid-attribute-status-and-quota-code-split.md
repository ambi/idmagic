---
status: completed
archived_from: 0500683499be50c11cc042b1d4814431d5d36788
---

# 同じ error code が接点によって別の意味を持つ 2 件を実装側で解消する

`mise run spec-diff` は `no normative specification change against main` を返す。規範は動いておらず、契約 (`wi-382` が既に正しい側を書いていた) に実装が追いついた変更である。`PATCH /api/account/v1/profile` の属性スキーマ違反が 400 から 422 になり、`UpdateAdminUser` と `UpdateUserProfileError422` の宣言に揃った。export 開始が実行中ジョブ数の上限で拒否されるとき、テナント資源クォータと共有していた `quota_exceeded` をやめて `active_job_quota_exceeded` を返すようになった。status (429) は変えていない。この 2 つは「待てば通る一時的な状態」と「解析できた内容の業務規則違反」で再試行の判断が逆になるので、code が分かれて初めてクライアントが区別できる。`ActiveJobQuotaExceededError` の `@doc` から、共有が server 側の欠陥であるという注記を外した。

全文は `git show 0500683499be50c11cc042b1d4814431d5d36788:<このファイルのパス>` で読む。

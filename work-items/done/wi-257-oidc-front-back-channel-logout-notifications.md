---
status: completed
archived_from: d517ed8a51f32f50ccb998fa6694ae416c9c3984
---

# OIDC Front-Channel / Back-Channel Logout の通知配送を実装する

既存の TypeSpec と規範文書が宣言していた OIDC Front-Channel / Back-Channel Logout を実装した。
認可コード交換で RP と `sid` の参加関係を保存し、`/end_session` のローカル失効後に iframe 応答と永続ジョブによる logout token 配送を開始する。
配送失敗はローカルログアウトを巻き戻さず、通知状態とジョブの試行回数へ残る。
`mise run spec-diff` 相当の確認では TypeSpec の規範差分は無く、実装と既存仕様の乖離を閉じた。

全文は `git show d517ed8a51f32f50ccb998fa6694ae416c9c3984:<このファイルのパス>` で読む。

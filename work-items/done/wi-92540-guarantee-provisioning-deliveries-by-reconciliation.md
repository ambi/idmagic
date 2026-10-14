---
status: completed
archived_from: c614638ae2d0c1a3c4c9b7fd74a2b7c4cab9d335
---

# プロビジョニングタスクの作成を、書き込み時の捕捉ではなく定期的な照合で保証する

`mise run spec-diff` は、追加したシナリオ `REQ-PROVISIONING-019`、変更したシナリオ `REQ-PLATFORM-003`、`REQ-PROVISIONING-003`、`REQ-PROVISIONING-017`、変更した標準行 `RFC7644-OUT-FILTERING` を挙げた。
`REQ-PLATFORM-003` は、プロビジョニングタスクを発火元の変更と同じトランザクションで作る保証から、書き込み時の捕捉または定期的な照合のどちらかでプロビジョニングタスクにする保証へ変わった。
具体例は、捕捉による通常経路、捕捉の失敗を照合が回収する経路、捕捉を呼ばない経路を照合が回収する経路の 3 つになった。
`REQ-PROVISIONING-019` は照合の規則として、リンクのない有効な User への `create`、割り当てを解除された User への `on_unassign` に従う無効化、未確定のタスクがある User を飛ばすこと、猶予期間つき削除を予約にすることを宣言した。
`REQ-PROVISIONING-003` と `REQ-PROVISIONING-017` は、捕捉が発火元のコミットの後に行われる現状に合わせて Given の文面だけを改めた。
`RFC7644-OUT-FILTERING` は、照合（Reconciliation）と紛れる「照合属性」を「一致判定の属性」へ言い換えただけである。
実装では、`RemoteResourceLink` に `active` を加え、User の削除を下流へ送ったらリンクを消すようにし、`worker` に `PROVISIONING_RECONCILE_INTERVAL`（デフォルト 5 分）ごとの照合のループを加えた。
書き込み時の捕捉と照合のバージョンを揃える当初の計画（旧 T004）は、未確定のタスクがある User を飛ばすことで二重作成を防げるため、採らなかった。

全文は `git show c614638ae2d0c1a3c4c9b7fd74a2b7c4cab9d335:<このファイルのパス>` で読む。

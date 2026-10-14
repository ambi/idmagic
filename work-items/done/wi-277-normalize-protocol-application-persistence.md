---
status: completed
archived_from: 772e5d30b3e0b04f435838ff635d1265ff84d587
---

# Application と単一 protocol 設定を通常の外部キーで正規化する

`clients` を `oauth2_clients` へ改名し、Application を optional な単一 protocol
契約へ変更した。3 protocol table は既存 identity を維持した nullable unique
`application_id` と固定 discriminator を持ち、複合 FK が tenant / protocol type を保証する。
Application insert と relation 設定は transaction で commit し、削除は protocol row へ cascade
する。catalog 外 record は NULL のまま有効で、linked record の低レベル削除は 409 とした。
JSON bindings、attach/detach API、複数形 response/UI を削除し、単一 `protocol` projection へ揃えた。

全文は `git show 772e5d30b3e0b04f435838ff635d1265ff84d587:<このファイルのパス>` で読む。

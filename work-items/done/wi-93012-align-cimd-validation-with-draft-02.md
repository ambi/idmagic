---
status: completed
---

# CIMD の取得と入力検証を draft 02 の必須条件に合わせる

CIMD の採用表を draft-02 へ固定し直し、規範 ID を `CIMD00-*` から `CIMD02-*` へ改めた（7 件を改名、3 件を追加）。
`CIMD02-URL-SHAPE` は単独の `.` と `..` のパス要素を拒否し、パーセント符号化した `%2e` の要素も同じ扱いにした。query は推奨に反するが受理を維持した。
`CIMD02-NO-REDIRECT` を加え、取得器が HTTP クライアントを複製してリダイレクトを追従しないようにした。3xx は 200 以外として拒否し、共有の `safehttp` は変えていない。
`CIMD02-CREDENTIAL-MATERIAL` を加え、`client_secret` と `client_secret_expires_at`（`null` を含む）、秘密鍵か対称鍵の成分を持つ `jwks`、鍵の配列として読めない `jwks` を含む文書を解決しないようにした。4.1 節は文書の発行者への義務であり、拒否は IdMagic の受け取り側の方針として採用表に書いた。公開鍵だけの `jwks` と未知の拡張は受理する。
`CIMD02-NO-ERROR-CACHE` は既存の振る舞いを規範として明示し、回帰検査を加えた。`CIMD02-CACHE` は一律 5 分の partial のままである。

残る制約：8.4 節の metadata 変更時の同意や付与の無効化は採らない。応答の上限は 64 KiB で、8.7 節が推奨する 5 KB より大きい。`jwks_uri` の先は取得しないので、その内容の秘密鍵は検査しない。
